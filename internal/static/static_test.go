package static

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func get(t *testing.T, v *Viewer) response {
	t.Helper()
	mux := http.NewServeMux()
	v.Routes(mux, func(h http.Handler) http.Handler { return h })
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/static", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var res response
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	return res
}

func TestUnconfigured(t *testing.T) {
	if res := get(t, New("")); res.Configured || res.Content != "" {
		t.Fatalf("expected unconfigured, got %+v", res)
	}
}

func TestMissingFile(t *testing.T) {
	res := get(t, New(filepath.Join(t.TempDir(), "traefik.yml")))
	if !res.Configured || res.Error == "" || res.Content != "" {
		t.Fatalf("expected an error for a missing file, got %+v", res)
	}
}

func TestMasksSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traefik.yml")
	yml := `entryPoints:
  web:
    address: ":80"
certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@example.com
      tokenSecret: hunter2
api:
  dashboard: true
  basicAuth:
    password: "$2y$05$abc"
`
	if err := os.WriteFile(path, []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	res := get(t, New(path))
	if res.Error != "" {
		t.Fatalf("unexpected error: %s", res.Error)
	}
	for _, want := range []string{"address: :80", "email: admin@example.com", "dashboard: true"} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("content lacks %q:\n%s", want, res.Content)
		}
	}
	for _, secret := range []string{"hunter2", "$2y$05$abc"} {
		if strings.Contains(res.Content, secret) {
			t.Errorf("secret %q leaked:\n%s", secret, res.Content)
		}
	}
}

func TestInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traefik.yml")
	if err := os.WriteFile(path, []byte("entryPoints: [unclosed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if res := get(t, New(path)); res.Error == "" {
		t.Fatalf("expected a parse error, got %+v", res)
	}
}
