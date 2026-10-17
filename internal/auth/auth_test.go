package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTest(t *testing.T) (*Auth, http.Handler) {
	t.Helper()
	a, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	a.Routes(mux)
	mux.Handle("/api/secret", a.Require(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("secret"))
	})))
	return a, mux
}

func do(h http.Handler, method, path, body string, cookie *http.Cookie, csrf bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.RemoteAddr = "10.0.0.1:1234"
	if csrf {
		r.Header.Set(csrfHeader, csrfValue)
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName {
			return c
		}
	}
	return nil
}

func TestSetupLoginFlow(t *testing.T) {
	_, h := newTest(t)

	if rec := do(h, "GET", "/api/secret", "", nil, false); rec.Code != 401 {
		t.Fatalf("unauthenticated access: got %d", rec.Code)
	}
	if rec := do(h, "POST", "/api/auth/setup", `{"password":"short"}`, nil, true); rec.Code != 400 {
		t.Fatalf("short password accepted: %d", rec.Code)
	}
	if rec := do(h, "POST", "/api/auth/setup", `{"password":"correct horse battery"}`, nil, false); rec.Code != 403 {
		t.Fatalf("setup without CSRF header: %d", rec.Code)
	}
	rec := do(h, "POST", "/api/auth/setup", `{"password":"correct horse battery"}`, nil, true)
	c := sessionCookie(rec)
	if rec.Code != 200 || c == nil || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
		t.Fatalf("setup: code %d cookie %+v", rec.Code, c)
	}
	if rec := do(h, "POST", "/api/auth/setup", `{"password":"another long password"}`, nil, true); rec.Code != 409 {
		t.Fatalf("second setup allowed: %d", rec.Code)
	}
	if rec := do(h, "GET", "/api/secret", "", c, false); rec.Code != 200 {
		t.Fatalf("authenticated access: %d", rec.Code)
	}

	do(h, "POST", "/api/auth/logout", "", c, true)
	if rec := do(h, "GET", "/api/secret", "", c, false); rec.Code != 401 {
		t.Fatalf("session survived logout: %d", rec.Code)
	}
	if rec := do(h, "POST", "/api/auth/login", `{"password":"wrong password!!"}`, nil, true); rec.Code != 401 {
		t.Fatalf("wrong password: %d", rec.Code)
	}
	rec = do(h, "POST", "/api/auth/login", `{"password":"correct horse battery"}`, nil, true)
	if rec.Code != 200 || sessionCookie(rec) == nil {
		t.Fatalf("login: %d", rec.Code)
	}
}

func TestPasswordPersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	a, _ := New(dir)
	mux := http.NewServeMux()
	a.Routes(mux)
	do(mux, "POST", "/api/auth/setup", `{"password":"correct horse battery"}`, nil, true)

	b, err := New(dir)
	if err != nil || b.hash == nil {
		t.Fatalf("credential not reloaded: %v", err)
	}
}

func TestCSRFOnProtectedWrites(t *testing.T) {
	_, h := newTest(t)
	c := sessionCookie(do(h, "POST", "/api/auth/setup", `{"password":"correct horse battery"}`, nil, true))

	if rec := do(h, "POST", "/api/secret", "", c, false); rec.Code != 403 {
		t.Fatalf("POST without header: %d", rec.Code)
	}
	r := httptest.NewRequest("POST", "/api/secret", nil)
	r.Header.Set(csrfHeader, csrfValue)
	r.Header.Set("Origin", "https://evil.example")
	r.AddCookie(c)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != 403 {
		t.Fatalf("cross-origin POST: %d", rec.Code)
	}
}

func TestLoginLockoutAndSessionExpiry(t *testing.T) {
	a, h := newTest(t)
	now := time.Now()
	a.now = func() time.Time { return now }
	c := sessionCookie(do(h, "POST", "/api/auth/setup", `{"password":"correct horse battery"}`, nil, true))
	do(h, "POST", "/api/auth/logout", "", c, true)

	for i := 0; i < maxFailures; i++ {
		do(h, "POST", "/api/auth/login", `{"password":"wrong password!!"}`, nil, true)
	}
	if rec := do(h, "POST", "/api/auth/login", `{"password":"correct horse battery"}`, nil, true); rec.Code != 429 {
		t.Fatalf("lockout not enforced: %d", rec.Code)
	}
	now = now.Add(lockout + time.Second)
	rec := do(h, "POST", "/api/auth/login", `{"password":"correct horse battery"}`, nil, true)
	if rec.Code != 200 {
		t.Fatalf("login after lockout: %d", rec.Code)
	}
	c = sessionCookie(rec)
	now = now.Add(sessionTTL + time.Second)
	if rec := do(h, "GET", "/api/secret", "", c, false); rec.Code != 401 {
		t.Fatalf("expired session accepted: %d", rec.Code)
	}
}

func TestPasswordChange(t *testing.T) {
	_, h := newTest(t)
	c := sessionCookie(do(h, "POST", "/api/auth/setup", `{"password":"correct horse battery"}`, nil, true))

	if rec := do(h, "POST", "/api/auth/password", `{"oldPassword":"wrong password!!","newPassword":"new password 1234"}`, c, true); rec.Code != 401 {
		t.Fatalf("wrong old password accepted: %d", rec.Code)
	}

	if rec := do(h, "POST", "/api/auth/password", `{"oldPassword":"correct horse battery","newPassword":"short"}`, c, true); rec.Code != 400 {
		t.Fatalf("short new password accepted: %d", rec.Code)
	}

	rec := do(h, "POST", "/api/auth/password", `{"oldPassword":"correct horse battery","newPassword":"brand new long password"}`, c, true)
	if rec.Code != 200 {
		t.Fatalf("valid password change failed: %d", rec.Code)
	}

	do(h, "POST", "/api/auth/logout", "", c, true)
	if rec := do(h, "POST", "/api/auth/login", `{"password":"correct horse battery"}`, nil, true); rec.Code != 401 {
		t.Fatalf("old password still works: %d", rec.Code)
	}

	if rec := do(h, "POST", "/api/auth/login", `{"password":"brand new long password"}`, nil, true); rec.Code != 200 {
		t.Fatalf("new password login failed: %d", rec.Code)
	}
}
