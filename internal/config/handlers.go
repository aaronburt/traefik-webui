package config

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
)

type handlers struct{ s *Store }

// Routes registers the /api/config/* endpoints. Wrap each with auth via wrap.
func (s *Store) Routes(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	h := handlers{s}
	reg := func(pattern string, f http.HandlerFunc) { mux.Handle(pattern, wrap(f)) }
	reg("GET /api/config/files", h.list)
	reg("GET /api/config/files/{name}", h.read)
	reg("PUT /api/config/files/{name}", h.save)
	reg("DELETE /api/config/files/{name}", h.remove)
	reg("POST /api/config/files/{name}/validate", h.validate)
	reg("POST /api/config/files/{name}/diff", h.diff)
	reg("GET /api/config/files/{name}/backups", h.backups)
	reg("GET /api/config/files/{name}/backups/{id}", h.backup)
	reg("POST /api/config/files/{name}/backups/{id}/restore", h.restore)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, os.ErrNotExist):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, errBadName):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type body struct {
	Content string `json:"content"`
}

func readBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize+4096)
	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return "", false
	}
	return b.Content, true
}

func (h handlers) list(w http.ResponseWriter, _ *http.Request) {
	files, err := h.s.List()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, files)
}

func (h handlers) read(w http.ResponseWriter, r *http.Request) {
	c, err := h.s.Read(r.PathValue("name"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, body{c})
}

func (h handlers) save(w http.ResponseWriter, r *http.Request) {
	c, ok := readBody(w, r)
	if !ok {
		return
	}
	res, err := h.s.Save(r.PathValue("name"), c)
	if err != nil {
		fail(w, err)
		return
	}
	code := 200
	if len(res.Problems) > 0 {
		code = http.StatusUnprocessableEntity
	}
	writeJSON(w, code, res)
}

func (h handlers) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.s.Delete(r.PathValue("name")); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"deleted": true})
}

func (h handlers) validate(w http.ResponseWriter, r *http.Request) {
	c, ok := readBody(w, r)
	if !ok {
		return
	}
	ps, err := h.s.Validate(r.PathValue("name"), c)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, Result{Problems: ps})
}

func (h handlers) diff(w http.ResponseWriter, r *http.Request) {
	c, ok := readBody(w, r)
	if !ok {
		return
	}
	cur, err := h.s.Read(r.PathValue("name"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"lines": Diff(cur, c)})
}

func (h handlers) backups(w http.ResponseWriter, r *http.Request) {
	bs, err := h.s.Backups(r.PathValue("name"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, bs)
}

func (h handlers) backup(w http.ResponseWriter, r *http.Request) {
	c, err := h.s.BackupContent(r.PathValue("name"), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, body{c})
}

func (h handlers) restore(w http.ResponseWriter, r *http.Request) {
	res, err := h.s.Restore(r.PathValue("name"), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, res)
}
