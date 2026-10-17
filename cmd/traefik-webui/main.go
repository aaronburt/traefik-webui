package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/aaronburt/traefik-webui/internal/server"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadConfig() (server.Config, string) {
	cfg := server.Config{
		TraefikAPI: getenv("TRAEFIK_API_URL", "http://traefik:8080"),
		DataDir:    getenv("DATA_DIR", "./data"),
		ConfigDir:  getenv("CONFIG_DIR", "./dynamic"),
	}
	addr := getenv("LISTEN_ADDR", ":8081")
	return cfg, addr
}

func serve(cfg server.Config, l net.Listener) error {
	h, err := server.New(cfg)
	if err != nil {
		return err
	}
	return http.Serve(l, h)
}

func main() {
	cfg, addr := loadConfig()
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	log.Printf("listening on %s, proxying Traefik API at %s", addr, cfg.TraefikAPI)
	log.Fatal(serve(cfg, l))
}
