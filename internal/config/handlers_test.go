package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockAuditor struct {
	events []string
}

func (m *mockAuditor) Log(event, user, ip, details string) error {
	m.events = append(m.events, event)
	return nil
}

func setupHandlersTest(t *testing.T) (*Store, *http.ServeMux, *mockAuditor) {
	t.Helper()
	s, err := NewStore(t.TempDir(), t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	aud := &mockAuditor{}
	s.SetAuditor(aud)

	mux := http.NewServeMux()
	s.Routes(mux, func(next http.Handler) http.Handler { return next })
	return s, mux, aud
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestHandlersFileCRUD(t *testing.T) {
	s, mux, aud := setupHandlersTest(t)
	_ = s

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/config/files", nil))
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	valReq := httptest.NewRequest("POST", "/api/config/files/test.yml/validate", strings.NewReader(`{"content":"http: ["}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, valReq)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var valRes Result
	json.NewDecoder(rec.Body).Decode(&valRes)
	if len(valRes.Problems) == 0 {
		t.Fatal("expected validation problems for bad yaml")
	}

	diffReq := httptest.NewRequest("POST", "/api/config/files/test.yml/diff", strings.NewReader(`{"content":"http:\n  routers: {}\n"}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, diffReq)
	if rec.Code != 200 {
		t.Fatalf("diff failed: %d", rec.Code)
	}

	badSave := httptest.NewRequest("PUT", "/api/config/files/test.yml", strings.NewReader(`{"content":"http: ["}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, badSave)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid save, got %d", rec.Code)
	}

	goodSave := httptest.NewRequest("PUT", "/api/config/files/test.yml", strings.NewReader(`{"content":`+jsonEscape(good)+`}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, goodSave)
	if rec.Code != 200 {
		t.Fatalf("expected 200 for good save, got %d %s", rec.Code, rec.Body.String())
	}
	if len(aud.events) == 0 || aud.events[len(aud.events)-1] != "config_save" {
		t.Fatalf("expected config_save audit event")
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/config/files/test.yml", nil))
	if rec.Code != 200 {
		t.Fatalf("read failed: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/config/files/nonexistent.yml", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/config/files/invalid_name", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad name, got %d", rec.Code)
	}

	changed := strings.Replace(good, "8080", "9090", 1)
	updSave := httptest.NewRequest("PUT", "/api/config/files/test.yml", strings.NewReader(`{"content":`+jsonEscape(changed)+`}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, updSave)
	if rec.Code != 200 {
		t.Fatalf("update save failed: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/config/files/test.yml/backups", nil))
	if rec.Code != 200 {
		t.Fatalf("backups list failed: %d", rec.Code)
	}
	var bs []Backup
	json.NewDecoder(rec.Body).Decode(&bs)
	if len(bs) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(bs))
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/config/files/test.yml/backups/"+bs[0].ID, nil))
	if rec.Code != 200 {
		t.Fatalf("backup content failed: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/config/files/test.yml/backups/"+bs[0].ID+"/restore", nil))
	if rec.Code != 200 {
		t.Fatalf("restore failed: %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/config/files/test.yml", nil))
	if rec.Code != 200 {
		t.Fatalf("delete failed: %d", rec.Code)
	}

	badBody := httptest.NewRequest("PUT", "/api/config/files/test.yml", strings.NewReader(`{invalid json`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, badBody)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed json body, got %d", rec.Code)
	}
}
