package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aaronburt/traefik-webui/internal/audit"
	"github.com/aaronburt/traefik-webui/internal/config"
)

const smokeYAML = `http:
  routers:
    smoke-router:
      rule: Host(` + "`smoke.example.com`" + `)
      service: smoke-svc
  services:
    smoke-svc:
      loadBalancer:
        servers:
          - url: http://10.0.0.1:8080
`

func TestSmokeFullLifecycle(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"Version":"3.1.0","Codename":"cousteau"}`))
		case "/api/overview":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"providers":["file"]}`))
		case "/api/entrypoints":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"name":"web","address":":80"}]`))
		case "/api/http/routers":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"name":"smoke-router@file","provider":"file","status":"enabled"}]`))
		case "/api/http/services":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[{"name":"smoke-svc@file","provider":"file","status":"enabled"}]`))
		case "/api/http/middlewares":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	dataDir := t.TempDir()
	configDir := t.TempDir()

	h, err := New(Config{
		TraefikAPI: upstream.URL,
		DataDir:    dataDir,
		ConfigDir:  configDir,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	exec := func(method, path string, body string, cookie *http.Cookie, csrf bool) *httptest.ResponseRecorder {
		var rdr io.Reader
		if body != "" {
			rdr = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, path, rdr)
		if csrf {
			req.Header.Set("X-Requested-With", "traefik-webui")
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := exec("GET", "/healthz", "", nil, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /healthz 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff header")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing X-Frame-Options DENY")
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("missing CSP frame-ancestors 'none'")
	}

	rec = exec("GET", "/", "", nil, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected / 200, got %d", rec.Code)
	}

	rec = exec("GET", "/api/auth/status", "", nil, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var st map[string]bool
	json.NewDecoder(rec.Body).Decode(&st)
	if !st["setupRequired"] || st["authenticated"] {
		t.Fatalf("unexpected auth status on first run: %+v", st)
	}

	if r := exec("GET", "/api/config/files", "", nil, false); r.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated config access must be 401, got %d", r.Code)
	}
	if r := exec("GET", "/api/audit", "", nil, false); r.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated audit access must be 401, got %d", r.Code)
	}
	if r := exec("GET", "/api/traefik/version", "", nil, false); r.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated proxy access must be 401, got %d", r.Code)
	}

	if r := exec("POST", "/api/auth/setup", `{"password":"correct horse battery"}`, nil, false); r.Code != http.StatusForbidden {
		t.Fatalf("setup without CSRF header must be 403, got %d", r.Code)
	}
	if r := exec("POST", "/api/auth/setup", `{"password":"short"}`, nil, true); r.Code != http.StatusBadRequest {
		t.Fatalf("setup with short password must be 400, got %d", r.Code)
	}

	rec = exec("POST", "/api/auth/setup", `{"password":"correct horse battery"}`, nil, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup failed: %d", rec.Code)
	}
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly {
		t.Fatalf("expected HttpOnly session cookie, got %+v", sessionCookie)
	}

	if r := exec("POST", "/api/auth/setup", `{"password":"another horse battery"}`, sessionCookie, true); r.Code != http.StatusConflict {
		t.Fatalf("second setup must be 409, got %d", r.Code)
	}

	rec = exec("GET", "/api/auth/status", "", sessionCookie, false)
	json.NewDecoder(rec.Body).Decode(&st)
	if st["setupRequired"] || !st["authenticated"] {
		t.Fatalf("expected setupRequired=false, authenticated=true: %+v", st)
	}

	rec = exec("GET", "/api/traefik/version", "", sessionCookie, false)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "3.1.0") {
		t.Fatalf("proxy version failed: %d %s", rec.Code, rec.Body.String())
	}
	rec = exec("GET", "/api/traefik/overview", "", sessionCookie, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("proxy overview failed: %d", rec.Code)
	}
	rec = exec("GET", "/api/traefik/entrypoints", "", sessionCookie, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("proxy entrypoints failed: %d", rec.Code)
	}

	rec = exec("GET", "/api/config/files", "", sessionCookie, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("list config files failed: %d", rec.Code)
	}

	rec = exec("POST", "/api/config/files/smoke.yml/validate", `{"content":"http: ["}`, sessionCookie, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("validate endpoint failed: %d", rec.Code)
	}
	var valRes config.Result
	json.NewDecoder(rec.Body).Decode(&valRes)
	if len(valRes.Problems) == 0 {
		t.Fatalf("expected validation problems for invalid YAML")
	}

	rec = exec("POST", "/api/config/files/smoke.yml/diff", `{"content":"`+escapeJSON(smokeYAML)+`"}`, sessionCookie, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("diff endpoint failed: %d", rec.Code)
	}

	rec = exec("PUT", "/api/config/files/smoke.yml", `{"content":"`+escapeJSON(smokeYAML)+`"}`, sessionCookie, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("save dynamic config failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = exec("GET", "/api/config/files/smoke.yml", "", sessionCookie, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("read dynamic config failed: %d", rec.Code)
	}

	updatedYAML := strings.Replace(smokeYAML, "smoke.example.com", "updated.example.com", 1)
	rec = exec("PUT", "/api/config/files/smoke.yml", `{"content":"`+escapeJSON(updatedYAML)+`"}`, sessionCookie, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("update dynamic config failed: %d", rec.Code)
	}

	rec = exec("GET", "/api/config/files/smoke.yml/backups", "", sessionCookie, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("backups list failed: %d", rec.Code)
	}
	var backups []config.Backup
	json.NewDecoder(rec.Body).Decode(&backups)
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}

	backupID := backups[0].ID
	rec = exec("GET", "/api/config/files/smoke.yml/backups/"+backupID, "", sessionCookie, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("read backup content failed: %d", rec.Code)
	}

	rec = exec("POST", "/api/config/files/smoke.yml/backups/"+backupID+"/restore", "", sessionCookie, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("restore backup failed: %d", rec.Code)
	}

	rec = exec("DELETE", "/api/config/files/smoke.yml", "", sessionCookie, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete config failed: %d", rec.Code)
	}

	rec = exec("GET", "/api/audit", "", sessionCookie, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit endpoint failed: %d", rec.Code)
	}
	var auditEntries []audit.Entry
	json.NewDecoder(rec.Body).Decode(&auditEntries)
	if len(auditEntries) < 4 {
		t.Fatalf("expected at least 4 audit events, got %d", len(auditEntries))
	}

	rec = exec("GET", "/api/audit/download", "", sessionCookie, false)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/csv" {
		t.Fatalf("audit download failed: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "timestamp,event,user,source_ip,details") {
		t.Fatalf("audit CSV header missing")
	}

	if r := exec("POST", "/api/auth/password", `{"oldPassword":"wrong","newPassword":"new password 1234"}`, sessionCookie, true); r.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong old password, got %d", r.Code)
	}
	if r := exec("POST", "/api/auth/password", `{"oldPassword":"correct horse battery","newPassword":"short"}`, sessionCookie, true); r.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short new password, got %d", r.Code)
	}
	rec = exec("POST", "/api/auth/password", `{"oldPassword":"correct horse battery","newPassword":"new password 1234"}`, sessionCookie, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("password change failed: %d", rec.Code)
	}

	rec = exec("POST", "/api/auth/logout", "", sessionCookie, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout failed: %d", rec.Code)
	}

	if r := exec("POST", "/api/auth/login", `{"password":"correct horse battery"}`, nil, true); r.Code != http.StatusUnauthorized {
		t.Fatalf("old password must be rejected after rotation, got %d", r.Code)
	}

	rec = exec("POST", "/api/auth/login", `{"password":"new password 1234"}`, nil, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("login with new password failed: %d", rec.Code)
	}
}

func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	return strings.Trim(string(b), `"`)
}
