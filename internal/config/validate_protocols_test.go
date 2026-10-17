package config

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateProtocols(t *testing.T) {
	tcp := `tcp:
  routers:
    db:
      rule: HostSNI(` + "`*`" + `)
      service: db
      entryPoints: [mysql]
  services:
    db:
      loadBalancer:
        servers:
          - address: 10.0.0.9:3306
  middlewares:
    allow:
      ipAllowList:
        sourceRange: [10.0.0.0/8]
`
	udp := `udp:
  routers:
    dns:
      service: dns
      entryPoints: [dns]
  services:
    dns:
      loadBalancer:
        servers:
          - address: 10.0.0.9:53
`
	mysql := map[string]bool{"mysql": true}
	cases := []struct {
		name, yaml, want string
		eps              map[string]bool
	}{
		{"tcp valid", tcp, "", nil},
		{"udp valid", udp, "", nil},
		{"tcp bad rule", strings.Replace(tcp, "HostSNI(`*`)", "Host(`a.example`)", 1), "no recognised matcher", nil},
		{"tcp missing service", strings.Replace(tcp, "service: db", "service: nope", 1), `service "nope" is not defined`, nil},
		{"tcp bad address", strings.Replace(tcp, "10.0.0.9:3306", "10.0.0.9", 1), "must look like host:port", nil},
		{"udp bad address", strings.Replace(udp, "10.0.0.9:53", "dns", 1), "must look like host:port", nil},
		{"udp needs balancer", strings.Replace(udp, "loadBalancer", "failover", 1), "exactly one of loadBalancer, weighted", nil},
		{"tcp unknown middleware", strings.Replace(tcp, "ipAllowList", "stripPrefix", 1), `unknown middleware type "stripPrefix"`, nil},
		{"unknown entrypoint", tcp, `entrypoint "mysql" is not defined`, map[string]bool{"web": true}},
		{"known entrypoint", tcp, "", mysql},
	}
	for _, c := range cases {
		got := problemText(Validate([]byte(c.yaml), Names{}, c.eps))
		if c.want == "" && got != "" || c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("%s: want %q, got %q", c.name, c.want, got)
		}
	}
}

func TestEntrypointsFromTraefik(t *testing.T) {
	const yml = "tcp:\n  routers:\n    db:\n      rule: HostSNI(`*`)\n      service: db\n      entryPoints: [mysql]\n" +
		"  services:\n    db:\n      loadBalancer:\n        servers:\n          - address: 10.0.0.9:3306\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/entrypoints" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`[{"name":"web"},{"name":"mysql"}]`))
	}))
	defer srv.Close()

	s := newStore(t, srv.URL)
	ps, err := s.Validate("db.yml", yml)
	if err != nil || len(ps) != 0 {
		t.Fatalf("known entrypoint: err %v, problems %q", err, problemText(ps))
	}
	ps, _ = s.Validate("db.yml", strings.Replace(yml, "mysql", "ghost", 1))
	if !strings.Contains(problemText(ps), `entrypoint "ghost" is not defined`) {
		t.Fatalf("unknown entrypoint not reported: %q", problemText(ps))
	}

	// With Traefik unreachable, entrypoint references are not checked.
	down := newStore(t, "http://127.0.0.1:1")
	ps, err = down.Validate("db.yml", strings.Replace(yml, "mysql", "ghost", 1))
	if err != nil || len(ps) != 0 {
		t.Fatalf("unreachable Traefik: err %v, problems %q", err, problemText(ps))
	}
}
