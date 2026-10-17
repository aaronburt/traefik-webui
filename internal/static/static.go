// Package static shows Traefik's static configuration read-only. Traefik reads
// that file once at startup, so the UI never writes to it.
package static

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// secretKey matches keys whose values are never shown, such as basic-auth hashes
// and API tokens.
var secretKey = regexp.MustCompile(`(?i)password|secret|token|credential|hash|apikey|^key$`)

// Viewer serves the static config file found at path. An empty path means none is set.
type Viewer struct{ path string }

// New returns a viewer for the YAML file at path.
func New(path string) *Viewer { return &Viewer{path: path} }

// Routes registers GET /api/static behind wrap, the auth check.
func (v *Viewer) Routes(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	mux.Handle("GET /api/static", wrap(http.HandlerFunc(v.serve)))
}

type response struct {
	Configured bool   `json:"configured"`
	Content    string `json:"content,omitempty"`
	Error      string `json:"error,omitempty"`
}

func (v *Viewer) serve(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if v.path == "" {
		json.NewEncoder(w).Encode(response{})
		return
	}
	content, err := v.load()
	res := response{Configured: true, Content: content}
	if err != nil {
		res.Error = "could not read the static config file: " + err.Error()
	}
	json.NewEncoder(w).Encode(res)
}

// load returns the file as YAML with secret values masked. Comments are not kept,
// because the file is parsed and written back out.
func (v *Viewer) load() (string, error) {
	raw, err := os.ReadFile(v.path)
	if err != nil {
		return "", err
	}
	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return "", err
	}
	out, err := yaml.Marshal(mask(doc))
	return string(out), err
}

func mask(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if secretKey.MatchString(k) {
				t[k] = "********"
			} else {
				t[k] = mask(val)
			}
		}
	case []any:
		for i := range t {
			t[i] = mask(t[i])
		}
	}
	return v
}
