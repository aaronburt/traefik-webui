package main

import (
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestGetenv(t *testing.T) {
	key := "TEST_TRAEFIK_ENV_VAR"
	os.Unsetenv(key)
	if v := getenv(key, "default_val"); v != "default_val" {
		t.Fatalf("expected default_val, got %s", v)
	}

	os.Setenv(key, "custom_val")
	defer os.Unsetenv(key)
	if v := getenv(key, "default_val"); v != "custom_val" {
		t.Fatalf("expected custom_val, got %s", v)
	}
}

func TestLoadConfigAndServe(t *testing.T) {
	os.Setenv("TRAEFIK_API_URL", "http://test-traefik:8080")
	os.Setenv("DATA_DIR", t.TempDir())
	os.Setenv("CONFIG_DIR", t.TempDir())
	os.Setenv("LISTEN_ADDR", ":9999")
	defer func() {
		os.Unsetenv("TRAEFIK_API_URL")
		os.Unsetenv("DATA_DIR")
		os.Unsetenv("CONFIG_DIR")
		os.Unsetenv("LISTEN_ADDR")
	}()

	cfg, addr, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.TraefikAPI != "http://test-traefik:8080" {
		t.Fatalf("unexpected API URL: %s", cfg.TraefikAPI)
	}
	if addr != ":9999" {
		t.Fatalf("unexpected addr: %s", addr)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer l.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- serve(cfg, l)
	}()

	resp, err := http.Get("http://" + l.Addr().String() + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	l.Close()
}

func TestLoadConfigReportsMissingSettings(t *testing.T) {
	for _, k := range []string{"TRAEFIK_API_URL", "CONFIG_DIR", "DATA_DIR", "LISTEN_ADDR"} {
		t.Setenv(k, "")
	}
	_, _, err := loadConfig()
	if err == nil {
		t.Fatal("expected an error with no settings")
	}
	for _, k := range []string{"TRAEFIK_API_URL", "CONFIG_DIR"} {
		if !strings.Contains(err.Error(), k) {
			t.Errorf("error does not name %s: %v", k, err)
		}
	}
}

func TestLoadConfigRejectsBadValues(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name, api, configDir, want string
	}{
		{"not a url", "traefik:8080", dir, "TRAEFIK_API_URL must be an http or https URL"},
		{"wrong scheme", "ftp://traefik", dir, "TRAEFIK_API_URL must be an http or https URL"},
		{"missing dir", "http://traefik:8080", dir + "/nope", "is not an existing directory"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("TRAEFIK_API_URL", c.api)
			t.Setenv("CONFIG_DIR", c.configDir)
			_, _, err := loadConfig()
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want error containing %q, got %v", c.want, err)
			}
		})
	}
}
