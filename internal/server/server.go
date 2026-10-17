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

	"github.com/aaronburt/traefik-webui/internal/auth"
)

//go:embed all:dist
var distFS embed.FS

// Config holds runtime settings, all supplied by the environment.
type Config struct {
	TraefikAPI string
	DataDir    string
}

// New builds the root handler.
func New(cfg Config) (http.Handler, error) {
	target, err := url.Parse(cfg.TraefikAPI)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)

	a, err := auth.New(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	// Only authenticated GET requests are forwarded; editing goes through dedicated
	// endpoints added later, never straight to Traefik.
	a.Routes(mux)
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
	return mux, nil
}
