package main

import (
	"log"
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

func main() {
	cfg := server.Config{
		TraefikAPI: getenv("TRAEFIK_API_URL", "http://traefik:8080"),
		DataDir:    getenv("DATA_DIR", "./data"),
	}
	addr := getenv("LISTEN_ADDR", ":8081")

	h, err := server.New(cfg)
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	log.Printf("listening on %s, proxying Traefik API at %s", addr, cfg.TraefikAPI)
	log.Fatal(http.ListenAndServe(addr, h))
}
