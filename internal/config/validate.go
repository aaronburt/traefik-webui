package config

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Problem is one validation finding, located by a dotted path into the file.
type Problem struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// protocols are the sections of a dynamic config file, in the order they are checked.
var protocols = []string{"http", "tcp", "udp"}

var httpMiddlewareTypes = map[string]bool{
	"addPrefix": true, "basicAuth": true, "buffering": true, "chain": true,
	"circuitBreaker": true, "compress": true, "contentType": true, "digestAuth": true,
	"errors": true, "forwardAuth": true, "headers": true, "inFlightReq": true,
	"ipAllowList": true, "ipWhiteList": true, "passTLSClientCert": true, "plugin": true,
	"rateLimit": true, "redirectRegex": true, "redirectScheme": true, "replacePath": true,
	"replacePathRegex": true, "retry": true, "stripPrefix": true, "stripPrefixRegex": true,
}

var tcpMiddlewareTypes = map[string]bool{
	"inFlightConn": true, "ipAllowList": true, "ipWhiteList": true,
}

var middlewareTypes = map[string]map[string]bool{
	"http": httpMiddlewareTypes,
	"tcp":  tcpMiddlewareTypes,
}

// serviceKinds are the balancing kinds a service may define, per protocol.
var serviceKinds = map[string][]string{
	"http": {"loadBalancer", "weighted", "mirroring", "failover"},
	"tcp":  {"loadBalancer", "weighted"},
	"udp":  {"loadBalancer", "weighted"},
}

var ruleMatcher = regexp.MustCompile(`\b(Host|HostRegexp|Path|PathPrefix|PathRegexp|Method|Header|HeaderRegexp|Query|QueryRegexp|ClientIP|HostSNI|HostSNIRegexp|ALPN)\(`)

var tcpRuleMatcher = regexp.MustCompile(`\b(HostSNI|HostSNIRegexp|ClientIP|ALPN)\(`)

// Names lists the objects a file defines, keyed by "protocol/kind" such as
// "http/routers", so references can be resolved across the whole config directory.
type Names map[string]map[string]bool

func (n Names) add(key, name string) {
	if n[key] == nil {
		n[key] = map[string]bool{}
	}
	n[key][name] = true
}

func (n Names) has(key, name string) bool { return n[key][name] }

type doc map[string]any

func parse(content []byte) (doc, error) {
	// Decode into the unnamed map type so nested mappings come back as
	// map[string]any too, which is what section() expects.
	var m map[string]any
	if err := yaml.Unmarshal(content, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return doc(m), nil
}

func section(m map[string]any, key string) map[string]any {
	s, _ := m[key].(map[string]any)
	return s
}

// collect gathers the names a document defines, in every protocol.
func collect(d doc) Names {
	n := Names{}
	for _, proto := range protocols {
		s := section(d, proto)
		for _, kind := range []string{"routers", "services", "middlewares"} {
			for name := range section(s, kind) {
				n.add(proto+"/"+kind, name)
			}
		}
	}
	return n
}

// Validate checks content as a Traefik dynamic configuration file. others holds
// the names defined by the other files in the directory. entrypoints is the set
// of entrypoint names Traefik has loaded. Nil means unknown, and then entrypoint
// references are not checked.
func Validate(content []byte, others Names, entrypoints map[string]bool) []Problem {
	var ps []Problem
	add := func(path, format string, args ...any) {
		ps = append(ps, Problem{path, fmt.Sprintf(format, args...)})
	}

	d, err := parse(content)
	if err != nil {
		return []Problem{{"", "invalid YAML: " + err.Error()}}
	}
	for k := range d {
		if k != "http" && k != "tcp" && k != "udp" && k != "tls" {
			add(k, "unknown top-level key %q (expected http, tcp, udp or tls)", k)
		}
	}

	own := collect(d)
	// has reports whether a reference resolves: a name with an @provider suffix,
	// or one defined in this file or in another file.
	has := func(key, name string) bool {
		return strings.Contains(name, "@") || own.has(key, name) || others.has(key, name)
	}
	checkEntryPoints := func(p string, r map[string]any) {
		if entrypoints == nil {
			return
		}
		for i, ep := range stringList(r["entryPoints"]) {
			if !entrypoints[ep] {
				add(fmt.Sprintf("%s.entryPoints[%d]", p, i), "entrypoint %q is not defined in Traefik", ep)
			}
		}
	}

	for _, proto := range protocols {
		h := section(d, proto)
		routers, services, middlewares := section(h, "routers"), section(h, "services"), section(h, "middlewares")

		for _, name := range sortedKeys(routers) {
			p := proto + ".routers." + name
			if others.has(proto+"/routers", name) {
				add(p, "router %q is already defined in another file", name)
			}
			r, ok := routers[name].(map[string]any)
			if !ok {
				add(p, "router must be a mapping")
				continue
			}
			rule, _ := r["rule"].(string)
			if msg := routerRule(proto, rule); msg != "" {
				add(p+".rule", "%s", msg)
			}
			svc, _ := r["service"].(string)
			switch {
			case svc == "":
				add(p+".service", "service is required")
			case !has(proto+"/services", svc):
				add(p+".service", "service %q is not defined", svc)
			}
			for _, m := range stringList(r["middlewares"]) {
				if !has(proto+"/middlewares", m) {
					add(p+".middlewares", "middleware %q is not defined", m)
				}
			}
			checkEntryPoints(p, r)
		}

		for _, name := range sortedKeys(services) {
			p := proto + ".services." + name
			if others.has(proto+"/services", name) {
				add(p, "service %q is already defined in another file", name)
			}
			s, ok := services[name].(map[string]any)
			if !ok {
				add(p, "service must be a mapping")
				continue
			}
			kinds := 0
			for _, k := range serviceKinds[proto] {
				if _, ok := s[k]; ok {
					kinds++
				}
			}
			if kinds != 1 {
				add(p, "service must define exactly one of %s", strings.Join(serviceKinds[proto], ", "))
				continue
			}
			if lb := section(s, "loadBalancer"); lb != nil {
				servers, _ := lb["servers"].([]any)
				if len(servers) == 0 {
					add(p+".loadBalancer.servers", "at least one server is required")
				}
				for i, sv := range servers {
					m, _ := sv.(map[string]any)
					if msg := checkServer(proto, m); msg != "" {
						add(fmt.Sprintf("%s.loadBalancer.servers[%d]", p, i), "%s", msg)
					}
				}
			}
		}

		for _, name := range sortedKeys(middlewares) {
			p := proto + ".middlewares." + name
			if others.has(proto+"/middlewares", name) {
				add(p, "middleware %q is already defined in another file", name)
			}
			m, ok := middlewares[name].(map[string]any)
			if !ok || len(m) != 1 {
				add(p, "middleware must define exactly one type")
				continue
			}
			for typ := range m {
				if !middlewareTypes[proto][typ] {
					add(p+"."+typ, "unknown middleware type %q", typ)
				}
				if typ == "chain" {
					for _, c := range stringList(section(m, "chain")["middlewares"]) {
						if !has(proto+"/middlewares", c) {
							add(p+".chain.middlewares", "middleware %q is not defined", c)
						}
					}
				}
			}
		}
	}
	return ps
}

// routerRule checks a router's rule for its protocol. UDP routers have no rule.
func routerRule(proto, rule string) string {
	switch proto {
	case "http":
		return checkRule(rule, ruleMatcher, "Host(`example.com`)")
	case "tcp":
		return checkRule(rule, tcpRuleMatcher, "HostSNI(`*`)")
	}
	return ""
}

func checkRule(rule string, matcher *regexp.Regexp, example string) string {
	switch {
	case strings.TrimSpace(rule) == "":
		return "rule is required"
	case strings.Count(rule, "`")%2 != 0:
		return "unbalanced backticks in rule"
	case strings.Count(rule, "(") != strings.Count(rule, ")"):
		return "unbalanced parentheses in rule"
	case !matcher.MatchString(rule):
		return "rule has no recognised matcher such as " + example
	}
	return ""
}

// checkServer reports a problem with one load balancer server: a URL for HTTP,
// and a host:port address for TCP and UDP.
func checkServer(proto string, m map[string]any) string {
	if proto == "http" {
		u, _ := m["url"].(string)
		if pu, err := url.Parse(u); err != nil || pu.Scheme == "" || pu.Host == "" {
			return fmt.Sprintf("server url %q must look like http://host:port", u)
		}
		return ""
	}
	addr, _ := m["address"].(string)
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return fmt.Sprintf("server address %q must look like host:port", addr)
	}
	return ""
}

func stringList(v any) []string {
	l, _ := v.([]any)
	out := make([]string, 0, len(l))
	for _, x := range l {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
