package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aaronburt/traefik-webui/internal/server"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadConfig reads the settings from the environment. Settings that depend on
// the Traefik instance have no default: the UI must be pointed at the instance
// it manages. Only generic mechanics such as the listen address fall back.
func loadConfig() (server.Config, string, error) {
	var missing []string
	required := func(key, hint string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, fmt.Sprintf("  %s: %s", key, hint))
		}
		return v
	}
	cfg := server.Config{
		TraefikAPI: required("TRAEFIK_API_URL", "base URL of Traefik's API, for example http://traefik:8080"),
		ConfigDir:  required("CONFIG_DIR", "the directory Traefik's file provider watches, for example /etc/traefik/dynamic"),
		DataDir:    getenv("DATA_DIR", "./data"),
	}
	addr := getenv("LISTEN_ADDR", ":8081")
	if len(missing) > 0 {
		return cfg, addr, fmt.Errorf("missing required settings:\n%s\nsee the README for every setting", strings.Join(missing, "\n"))
	}

	if u, err := url.Parse(cfg.TraefikAPI); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return cfg, addr, fmt.Errorf("TRAEFIK_API_URL must be an http or https URL such as http://traefik:8080, got %q", cfg.TraefikAPI)
	}
	if fi, err := os.Stat(cfg.ConfigDir); err != nil || !fi.IsDir() {
		return cfg, addr, fmt.Errorf("CONFIG_DIR %q is not an existing directory; it must be the directory Traefik's file provider watches, and it is not created for you", cfg.ConfigDir)
	}
	return cfg, addr, nil
}

func serve(cfg server.Config, l net.Listener) error {
	h, err := server.New(cfg)
	if err != nil {
		return err
	}
	return http.Serve(l, h)
}

func main() {
	cfg, addr, err := loadConfig()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	log.Printf("listening on %s, proxying Traefik API at %s, config directory %s", addr, cfg.TraefikAPI, cfg.ConfigDir)
	log.Fatal(serve(cfg, l))
}
