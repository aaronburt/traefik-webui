package config

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const good = `http:
  routers:
    app:
      rule: Host(` + "`app.example.com`" + `)
      service: app
      middlewares: [strip]
  services:
    app:
      loadBalancer:
        servers:
          - url: http://10.0.0.5:8080
  middlewares:
    strip:
      stripPrefix:
        prefixes: [/app]
`

func newStore(t *testing.T, traefik string) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir(), t.TempDir(), traefik)
	if err != nil {
		t.Fatal(err)
	}
	s.poll, s.timeout = time.Millisecond, 20*time.Millisecond
	return s
}

func problemText(ps []Problem) string {
	var b strings.Builder
	for _, p := range ps {
		b.WriteString(p.Path + ": " + p.Message + "\n")
	}
	return b.String()
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name, yaml, want string
	}{
		{"valid", good, ""},
		{"bad yaml", "http: [", "invalid YAML"},
		{"unknown key", "htp: {}", "unknown top-level key"},
		{"missing service", strings.Replace(good, "service: app", "service: nope", 1), `service "nope" is not defined`},
		{"missing middleware", strings.Replace(good, "[strip]", "[ghost]", 1), `middleware "ghost" is not defined`},
		{"bad rule", strings.Replace(good, "Host(`app.example.com`)", "Host(`app.example.com`", 1), "unbalanced"},
		{"no matcher", strings.Replace(good, "Host(`app.example.com`)", "something", 1), "no recognised matcher"},
		{"bad url", strings.Replace(good, "http://10.0.0.5:8080", "10.0.0.5", 1), "must look like"},
		{"unknown middleware", strings.Replace(good, "stripPrefix", "stripPrefx", 1), "unknown middleware type"},
		{"cross-provider ref ok", strings.Replace(good, "service: app", "service: api@internal", 1), ""},
	}
	for _, c := range cases {
		got := problemText(Validate([]byte(c.yaml), newNames()))
		if c.want == "" && got != "" || !strings.Contains(got, c.want) {
			t.Errorf("%s: want %q, got %q", c.name, c.want, got)
		}
	}
}

func TestSaveBackupRestoreAndDuplicates(t *testing.T) {
	s := newStore(t, "")

	res, err := s.Save("app.yml", good)
	if err != nil || !res.Applied {
		t.Fatalf("first save: %+v %v", res, err)
	}
	changed := strings.Replace(good, "8080", "9090", 1)
	if res, _ := s.Save("app.yml", changed); !res.Applied {
		t.Fatalf("second save: %+v", res)
	}
	if got, _ := s.Read("app.yml"); got != changed {
		t.Fatal("content not updated")
	}

	bs, _ := s.Backups("app.yml")
	if len(bs) != 1 {
		t.Fatalf("want 1 backup, got %d", len(bs))
	}
	if old, _ := s.BackupContent("app.yml", bs[0].ID); old != good {
		t.Fatal("backup does not hold the previous version")
	}
	if res, _ := s.Restore("app.yml", bs[0].ID); !res.Applied {
		t.Fatal("restore failed")
	}
	if got, _ := s.Read("app.yml"); got != good {
		t.Fatal("restore did not bring the old content back")
	}
	if bs, _ := s.Backups("app.yml"); len(bs) != 2 {
		t.Fatalf("restore should itself be backed up, got %d backups", len(bs))
	}

	res, _ = s.Save("other.yml", good)
	if res.Applied || !strings.Contains(problemText(res.Problems), "already defined in another file") {
		t.Fatalf("duplicate router accepted: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(s.dir, "other.yml")); err == nil {
		t.Fatal("invalid file was written")
	}
}

func TestInvalidSaveLeavesFileAlone(t *testing.T) {
	s := newStore(t, "")
	s.Save("app.yml", good)
	res, _ := s.Save("app.yml", "http: [")
	if res.Applied || len(res.Problems) == 0 {
		t.Fatal("invalid content was applied")
	}
	if got, _ := s.Read("app.yml"); got != good {
		t.Fatal("file changed despite validation failure")
	}
}

func TestFileNamesAreRestricted(t *testing.T) {
	s := newStore(t, "")
	for _, n := range []string{"../evil.yml", "a/b.yml", ".hidden.yml", "x.txt", "", "a b.yml"} {
		if _, err := s.Save(n, good); err == nil {
			t.Errorf("name %q accepted", n)
		}
	}
	if _, err := s.BackupContent("app.yml", "../../etc/passwd"); err == nil {
		t.Error("backup id traversal accepted")
	}
}

func TestDeleteIsBackedUp(t *testing.T) {
	s := newStore(t, "")
	s.Save("app.yml", good)
	if err := s.Delete("app.yml"); err != nil {
		t.Fatal(err)
	}
	if bs, _ := s.Backups("app.yml"); len(bs) != 1 {
		t.Fatal("delete did not leave a backup")
	}
}

func TestPostApplyWarnings(t *testing.T) {
	traefik := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/http/routers" {
			w.Write([]byte(`[{"name":"app@file","provider":"file","status":"disabled","error":["service not found"]},{"name":"x@docker","provider":"docker","status":"disabled"}]`))
			return
		}
		if r.URL.Path == "/api/http/services" {
			w.Write([]byte(`[{"name":"app@file","provider":"file","status":"enabled"}]`))
			return
		}
		w.Write([]byte(`[{"name":"strip@file","provider":"file","status":"enabled"}]`))
	}))
	defer traefik.Close()

	res, _ := newStore(t, traefik.URL).Save("app.yml", good)
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "service not found") {
		t.Fatalf("warnings: %v", res.Warnings)
	}
}

func TestDiff(t *testing.T) {
	d := Diff("a\nb\nc\n", "a\nx\nc\n")
	var got string
	for _, l := range d {
		got += l.Op + l.Text + "|"
	}
	if got != " a|-b|+x| c|" {
		t.Fatalf("diff: %q", got)
	}
}
