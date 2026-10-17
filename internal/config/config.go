// Package config manages Traefik dynamic configuration files in a directory
// watched by Traefik's file provider: listing, validated atomic writes,
// backups with rollback, and a post-apply check against the Traefik API.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxFileSize = 1 << 20

var fileName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.ya?ml$`)

// Store owns the config directory and the backup directory.
type Store struct {
	dir     string
	backups string
	traefik string        // Traefik API base URL, for the post-apply check; may be empty
	poll    time.Duration // how often to ask Traefik whether it has loaded a change
	timeout time.Duration // how long to wait for it to do so

	mu sync.Mutex // serialises writes

	epMu sync.Mutex      // guards the entrypoint cache
	ep   map[string]bool // entrypoints Traefik last reported; nil when unknown
	epAt time.Time       // when ep was fetched
}

// NewStore creates a Store, making the directories if needed.
func NewStore(configDir, dataDir, traefikAPI string) (*Store, error) {
	s := &Store{
		dir:     configDir,
		backups: filepath.Join(dataDir, "backups"),
		traefik: strings.TrimRight(traefikAPI, "/"),
		poll:    500 * time.Millisecond,
		timeout: 8 * time.Second, // Traefik throttles file-provider reloads by 2s
	}
	for _, d := range []string{s.dir, s.backups} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// FileInfo describes one config file.
type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// Backup describes one saved version of a file.
type Backup struct {
	ID   string    `json:"id"`
	Time time.Time `json:"time"`
	Size int64     `json:"size"`
}

// Result is returned after saving or restoring a file.
type Result struct {
	Problems []Problem `json:"problems,omitempty"` // validation errors; nothing was written
	Applied  bool      `json:"applied"`
	Warnings []string  `json:"warnings,omitempty"` // Traefik objects from the file provider that are not enabled
}

var errBadName = errors.New("invalid file name: use letters, digits, '.', '_' or '-' and end in .yml or .yaml")

func (s *Store) path(name string) (string, error) {
	if !fileName.MatchString(name) {
		return "", errBadName
	}
	return filepath.Join(s.dir, name), nil
}

// List returns the YAML files in the config directory.
func (s *Store) List() ([]FileInfo, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	out := []FileInfo{}
	for _, e := range entries {
		if e.IsDir() || !fileName.MatchString(e.Name()) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, FileInfo{e.Name(), fi.Size(), fi.ModTime()})
	}
	return out, nil
}

// Read returns a file's content.
func (s *Store) Read(name string) (string, error) {
	p, err := s.path(name)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	return string(b), err
}

// othersNames gathers names defined in every file except skip.
func (s *Store) othersNames(skip string) Names {
	all := Names{}
	files, _ := s.List()
	for _, f := range files {
		if f.Name == skip {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.dir, f.Name))
		if err != nil {
			continue
		}
		d, err := parse(b)
		if err != nil {
			continue
		}
		for key, names := range collect(d) {
			for name := range names {
				all.add(key, name)
			}
		}
	}
	return all
}

// Validate checks content as it would be saved under name.
func (s *Store) Validate(name, content string) ([]Problem, error) {
	if _, err := s.path(name); err != nil {
		return nil, err
	}
	if len(content) > maxFileSize {
		return []Problem{{"", "file is larger than 1 MB"}}, nil
	}
	return Validate([]byte(content), s.othersNames(name), s.entrypoints()), nil
}

// entrypoints returns the entrypoint names Traefik reports, cached for 10 seconds.
// It returns nil when Traefik cannot be reached, and entrypoint references are
// then not checked.
func (s *Store) entrypoints() map[string]bool {
	if s.traefik == "" {
		return nil
	}
	s.epMu.Lock()
	defer s.epMu.Unlock()
	if time.Since(s.epAt) < 10*time.Second {
		return s.ep
	}
	var ep map[string]bool
	client := http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Get(s.traefik + "/api/entrypoints"); err == nil {
		var items []struct {
			Name string `json:"name"`
		}
		if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&items) == nil {
			ep = map[string]bool{}
			for _, it := range items {
				ep[it.Name] = true
			}
		}
		resp.Body.Close()
	}
	s.ep, s.epAt = ep, time.Now()
	return ep
}

// Save validates content and, if it is clean, backs up the current version and
// writes the new one atomically, then checks what Traefik made of it.
func (s *Store) Save(name, content string) (Result, error) {
	p, err := s.path(name)
	if err != nil {
		return Result{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	problems, err := s.Validate(name, content)
	if err != nil || len(problems) > 0 {
		return Result{Problems: problems}, err
	}
	if err := s.backup(name); err != nil {
		return Result{}, fmt.Errorf("backup failed, file not changed: %w", err)
	}
	if err := writeAtomic(p, []byte(content)); err != nil {
		return Result{}, err
	}
	return Result{Applied: true, Warnings: s.check(content)}, nil
}

// Delete removes a file after backing it up.
func (s *Store) Delete(name string) error {
	p, err := s.path(name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.backup(name); err != nil {
		return fmt.Errorf("backup failed, file not deleted: %w", err)
	}
	return os.Remove(p)
}

func (s *Store) backupDir(name string) string { return filepath.Join(s.backups, name) }

// backup copies the current file, if any, into the backup directory.
func (s *Store) backup(name string) error {
	b, err := os.ReadFile(filepath.Join(s.dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.backupDir(name), 0o700); err != nil {
		return err
	}
	id := time.Now().UTC().Format("20060102T150405.000000000Z")
	return writeAtomic(filepath.Join(s.backupDir(name), id), b)
}

// Backups lists saved versions of a file, newest first.
func (s *Store) Backups(name string) ([]Backup, error) {
	if _, err := s.path(name); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.backupDir(name))
	if errors.Is(err, os.ErrNotExist) {
		return []Backup{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Backup{}
	for _, e := range entries {
		t, err := time.Parse("20060102T150405.000000000Z", e.Name())
		if err != nil {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Backup{e.Name(), t, fi.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

// BackupContent returns the content of one saved version.
func (s *Store) BackupContent(name, id string) (string, error) {
	if _, err := s.path(name); err != nil {
		return "", err
	}
	if _, err := time.Parse("20060102T150405.000000000Z", id); err != nil {
		return "", errors.New("invalid backup id")
	}
	b, err := os.ReadFile(filepath.Join(s.backupDir(name), id))
	return string(b), err
}

// Restore writes a saved version back as the current file. The version being
// replaced is itself backed up, so a restore can be undone.
func (s *Store) Restore(name, id string) (Result, error) {
	content, err := s.BackupContent(name, id)
	if err != nil {
		return Result{}, err
	}
	return s.Save(name, content)
}

type apiItem struct {
	Name     string   `json:"name"`
	Provider string   `json:"provider"`
	Status   string   `json:"status"`
	Error    []string `json:"error"`
}

// fetch returns the HTTP routers, services and middlewares Traefik reports.
func (s *Store) fetch() (map[string][]apiItem, error) {
	out := map[string][]apiItem{}
	for _, kind := range []string{"routers", "services", "middlewares"} {
		resp, err := http.Get(s.traefik + "/api/http/" + kind + "?per_page=1000")
		if err != nil {
			return nil, err
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		var items []apiItem
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, fmt.Errorf("unexpected response from Traefik for %s", kind)
		}
		out[kind] = items
	}
	return out, nil
}

// check waits for Traefik to load the objects the saved content defines and
// reports any from the file provider that are not enabled.
func (s *Store) check(content string) []string {
	if s.traefik == "" {
		return nil
	}
	want := map[string][]string{}
	if d, err := parse([]byte(content)); err == nil {
		// The post-apply check reads only Traefik's HTTP objects.
		n := collect(d)
		for k := range n["http/routers"] {
			want["routers"] = append(want["routers"], k+"@file")
		}
		for k := range n["http/services"] {
			want["services"] = append(want["services"], k+"@file")
		}
		for k := range n["http/middlewares"] {
			want["middlewares"] = append(want["middlewares"], k+"@file")
		}
	}

	var got map[string][]apiItem
	var missing []string
	deadline := time.Now().Add(s.timeout)
	for {
		var err error
		got, err = s.fetch()
		if err != nil {
			return []string{"could not reach the Traefik API to confirm the change: " + err.Error()}
		}
		missing = missing[:0]
		for kind, names := range want {
			have := map[string]bool{}
			for _, it := range got[kind] {
				have[it.Name] = true
			}
			for _, n := range names {
				if !have[n] {
					missing = append(missing, n)
				}
			}
		}
		if len(missing) == 0 || !time.Now().Before(deadline) {
			break
		}
		time.Sleep(s.poll)
	}

	var warnings []string
	for _, n := range missing {
		warnings = append(warnings, n+" has not appeared in Traefik yet")
	}
	for _, kind := range []string{"routers", "services", "middlewares"} {
		for _, it := range got[kind] {
			if it.Provider == "file" && it.Status != "enabled" {
				msg := fmt.Sprintf("%s %s is %s", strings.TrimSuffix(kind, "s"), it.Name, it.Status)
				if len(it.Error) > 0 {
					msg += ": " + strings.Join(it.Error, "; ")
				}
				warnings = append(warnings, msg)
			}
		}
	}
	return warnings
}

func writeAtomic(path string, b []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
