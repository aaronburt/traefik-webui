// Package server wires the HTTP handlers: a read-only proxy to the Traefik
// API plus the embedded Svelte frontend.
package server

import (
	"embed"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/aaronburt/traefik-webui/internal/audit"
	"github.com/aaronburt/traefik-webui/internal/auth"
	"github.com/aaronburt/traefik-webui/internal/config"
)

//go:embed all:dist
var distFS embed.FS

// Config holds runtime settings, all supplied by the environment.
type Config struct {
	TraefikAPI string
	DataDir    string
	ConfigDir  string // directory watched by Traefik's file provider
}

// New builds the root handler.
func New(cfg Config) (http.Handler, error) {
	target, err := url.Parse(cfg.TraefikAPI)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)

	aud, err := audit.New(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	a, err := auth.New(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	a.SetAuditor(aud)

	store, err := config.NewStore(cfg.ConfigDir, cfg.DataDir, cfg.TraefikAPI)
	if err != nil {
		return nil, err
	}
	store.SetAuditor(aud)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	a.Routes(mux)
	store.Routes(mux, a.Require)
	aud.Routes(mux, a.Require)
	mux.Handle("/api/traefik/", a.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.URL.Path = "/api/" + strings.TrimPrefix(r.URL.Path, "/api/traefik/")
		proxy.ServeHTTP(w, r)
	})))

	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServerFS(sub))
	return securityHeaders(mux), nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
