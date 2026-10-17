package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthzAndAuthenticatedProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/version" {
			t.Errorf("unexpected upstream path %q", r.URL.Path)
		}
		w.Write([]byte(`{"Version":"3.0"}`))
	}))
	defer upstream.Close()

	h, err := New(Config{TraefikAPI: upstream.URL, DataDir: t.TempDir(), ConfigDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != 200 {
		t.Fatalf("healthz: got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/traefik/version", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("proxy without login: got %d", rec.Code)
	}

	setup := httptest.NewRequest("POST", "/api/auth/setup", strings.NewReader(`{"password":"correct horse battery"}`))
	setup.Header.Set("X-Requested-With", "traefik-webui")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, setup)
	if rec.Code != 200 {
		t.Fatalf("setup: got %d", rec.Code)
	}
	cookie := rec.Result().Cookies()[0]

	req := httptest.NewRequest("GET", "/api/traefik/version", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != `{"Version":"3.0"}` {
		t.Fatalf("proxy: got %d %q", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest("POST", "/api/traefik/version", nil)
	req.AddCookie(cookie)
	req.Header.Set("X-Requested-With", "traefik-webui")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST should be refused, got %d", rec.Code)
	}

	healthReq := httptest.NewRequest("GET", "/healthz", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, healthReq)
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff header")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing X-Frame-Options DENY")
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("missing CSP frame-ancestors 'none'")
	}

	httpsReq := httptest.NewRequest("GET", "/healthz", nil)
	httpsReq.Header.Set("X-Forwarded-Proto", "https")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httpsReq)
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Fatalf("missing HSTS header on HTTPS request")
	}

	auditReq := httptest.NewRequest("GET", "/api/audit", nil)
	auditReq.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, auditReq)
	if rec.Code != 200 {
		t.Fatalf("audit route failed: got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "auth_setup") {
		t.Fatalf("audit log does not contain auth_setup: %s", rec.Body.String())
	}
}
