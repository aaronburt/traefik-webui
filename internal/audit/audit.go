package audit

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Timestamp string `json:"timestamp"`
	Event     string `json:"event"`
	User      string `json:"user"`
	SourceIP  string `json:"sourceIP"`
	Details   string `json:"details"`
}

type Logger struct {
	path string
	mu   sync.Mutex
	now  func() time.Time
}

func New(dataDir string) (*Logger, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	l := &Logger{
		path: filepath.Join(dataDir, "audit.csv"),
		now:  time.Now,
	}
	return l, nil
}

func (l *Logger) Log(event, user, ip, details string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	exists := false
	if _, err := os.Stat(l.path); err == nil {
		exists = true
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if !exists {
		if err := w.Write([]string{"timestamp", "event", "user", "source_ip", "details"}); err != nil {
			return err
		}
	}

	record := []string{
		l.now().UTC().Format(time.RFC3339),
		event,
		user,
		ip,
		details,
	}
	if err := w.Write(record); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func (l *Logger) Recent(limit int) ([]Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.Open(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if len(records) <= 1 {
		return []Entry{}, nil
	}

	rows := records[1:]
	out := make([]Entry, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		if len(row) < 5 {
			continue
		}
		out = append(out, Entry{
			Timestamp: row[0],
			Event:     row[1],
			User:      row[2],
			SourceIP:  row[3],
			Details:   row[4],
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (l *Logger) Routes(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	mux.Handle("GET /api/audit", wrap(http.HandlerFunc(l.handleRecent)))
	mux.Handle("GET /api/audit/download", wrap(http.HandlerFunc(l.handleDownload)))
}

func (l *Logger) handleRecent(w http.ResponseWriter, _ *http.Request) {
	entries, err := l.Recent(100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (l *Logger) handleDownload(w http.ResponseWriter, _ *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.Open(l.path)
	if errors.Is(err, os.ErrNotExist) {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=\"audit.csv\"")
		w.Write([]byte("timestamp,event,user,source_ip,details\n"))
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"audit.csv\"")
	io.Copy(w, f)
}
