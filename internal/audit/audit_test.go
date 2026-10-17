package audit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestAuditLogAndRecent(t *testing.T) {
	dir := t.TempDir()
	l, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return now }

	if err := l.Log("auth_login", "admin", "127.0.0.1", "successful login"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if err := l.Log("config_save", "admin", "127.0.0.1", "saved app.yml (+10/-2)"); err != nil {
		t.Fatal(err)
	}

	entries, err := l.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Event != "config_save" {
		t.Fatalf("expected newest event first, got %s", entries[0].Event)
	}
	if entries[1].Event != "auth_login" {
		t.Fatalf("expected second event auth_login, got %s", entries[1].Event)
	}
}

func TestAuditConcurrent(t *testing.T) {
	dir := t.TempDir()
	l, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = l.Log("test_event", "admin", "127.0.0.1", fmt.Sprintf("iteration %d", idx))
		}(i)
	}
	wg.Wait()

	entries, err := l.Recent(50)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 20 {
		t.Fatalf("expected 20 entries, got %d", len(entries))
	}
}

func TestAuditDownloadRoute(t *testing.T) {
	dir := t.TempDir()
	l, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = l.Log("auth_setup", "admin", "10.0.0.1", "initial admin created")

	mux := http.NewServeMux()
	l.Routes(mux, func(next http.Handler) http.Handler { return next })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/audit/download", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/csv" {
		t.Fatalf("expected text/csv, got %s", rec.Header().Get("Content-Type"))
	}
}
