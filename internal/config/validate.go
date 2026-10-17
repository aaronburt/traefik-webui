package config

import (
	"fmt"
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

var httpMiddlewareTypes = map[string]bool{
	"addPrefix": true, "basicAuth": true, "buffering": true, "chain": true,
	"circuitBreaker": true, "compress": true, "contentType": true, "digestAuth": true,
	"errors": true, "forwardAuth": true, "headers": true, "inFlightReq": true,
	"ipAllowList": true, "ipWhiteList": true, "passTLSClientCert": true, "plugin": true,
	"rateLimit": true, "redirectRegex": true, "redirectScheme": true, "replacePath": true,
	"replacePathRegex": true, "retry": true, "stripPrefix": true, "stripPrefixRegex": true,
}

var ruleMatcher = regexp.MustCompile(`\b(Host|HostRegexp|Path|PathPrefix|PathRegexp|Method|Header|HeaderRegexp|Query|QueryRegexp|ClientIP|HostSNI|HostSNIRegexp|ALPN)\(`)

// Names lists the router, service and middleware names a file defines, per
// protocol, so references can be resolved across the whole config directory.
type Names struct {
	Routers     map[string]bool
	Services    map[string]bool
	Middlewares map[string]bool
}

func newNames() Names {
	return Names{map[string]bool{}, map[string]bool{}, map[string]bool{}}
}

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

// collect gathers the HTTP names defined in a document.
func collect(d doc) Names {
	n := newNames()
	h := section(d, "http")
	for k := range section(h, "routers") {
		n.Routers[k] = true
	}
	for k := range section(h, "services") {
		n.Services[k] = true
	}
	for k := range section(h, "middlewares") {
		n.Middlewares[k] = true
	}
	return n
}

// Validate checks content as a Traefik dynamic configuration file. others holds
// the names defined by the other files in the directory.
func Validate(content []byte, others Names) []Problem {
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
	h := section(d, "http")
	routers, services, middlewares := section(h, "routers"), section(h, "services"), section(h, "middlewares")

	hasService := func(name string) bool {
		return strings.Contains(name, "@") || own.Services[name] || others.Services[name]
	}
	hasMiddleware := func(name string) bool {
		return strings.Contains(name, "@") || own.Middlewares[name] || others.Middlewares[name]
	}

	for _, name := range sortedKeys(routers) {
		p := "http.routers." + name
		if others.Routers[name] {
			add(p, "router %q is already defined in another file", name)
		}
		r, ok := routers[name].(map[string]any)
		if !ok {
			add(p, "router must be a mapping")
			continue
		}
		rule, _ := r["rule"].(string)
		if msg := checkRule(rule); msg != "" {
			add(p+".rule", "%s", msg)
		}
		svc, _ := r["service"].(string)
		switch {
		case svc == "":
			add(p+".service", "service is required")
		case !hasService(svc):
			add(p+".service", "service %q is not defined", svc)
		}
		for _, m := range stringList(r["middlewares"]) {
			if !hasMiddleware(m) {
				add(p+".middlewares", "middleware %q is not defined", m)
			}
		}
	}

	for _, name := range sortedKeys(services) {
		p := "http.services." + name
		if others.Services[name] {
			add(p, "service %q is already defined in another file", name)
		}
		s, ok := services[name].(map[string]any)
		if !ok {
			add(p, "service must be a mapping")
			continue
		}
		kinds := 0
		for _, k := range []string{"loadBalancer", "weighted", "mirroring", "failover"} {
			if _, ok := s[k]; ok {
				kinds++
			}
		}
		if kinds != 1 {
			add(p, "service must define exactly one of loadBalancer, weighted, mirroring or failover")
			continue
		}
		if lb := section(s, "loadBalancer"); lb != nil {
			servers, _ := lb["servers"].([]any)
			if len(servers) == 0 {
				add(p+".loadBalancer.servers", "at least one server is required")
			}
			for i, sv := range servers {
				m, _ := sv.(map[string]any)
				u, _ := m["url"].(string)
				if pu, err := url.Parse(u); err != nil || pu.Scheme == "" || pu.Host == "" {
					add(fmt.Sprintf("%s.loadBalancer.servers[%d].url", p, i), "server url %q must look like http://host:port", u)
				}
			}
		}
	}

	for _, name := range sortedKeys(middlewares) {
		p := "http.middlewares." + name
		if others.Middlewares[name] {
			add(p, "middleware %q is already defined in another file", name)
		}
		m, ok := middlewares[name].(map[string]any)
		if !ok || len(m) != 1 {
			add(p, "middleware must define exactly one type")
			continue
		}
		for typ, v := range m {
			if !httpMiddlewareTypes[typ] {
				add(p+"."+typ, "unknown middleware type %q", typ)
			}
			if typ == "chain" {
				for _, c := range stringList(section(m, "chain")["middlewares"]) {
					if !hasMiddleware(c) {
						add(p+".chain.middlewares", "middleware %q is not defined", c)
					}
				}
			}
			_ = v
		}
	}
	return ps
}

func checkRule(rule string) string {
	switch {
	case strings.TrimSpace(rule) == "":
		return "rule is required"
	case strings.Count(rule, "`")%2 != 0:
		return "unbalanced backticks in rule"
	case strings.Count(rule, "(") != strings.Count(rule, ")"):
		return "unbalanced parentheses in rule"
	case !ruleMatcher.MatchString(rule):
		return "rule has no recognised matcher such as Host(`example.com`)"
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
