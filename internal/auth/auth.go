// Package auth implements single-user password authentication: first-run
// setup, login, cookie sessions, CSRF checks and login rate limiting.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	cookieName     = "session"
	csrfHeader     = "X-Requested-With"
	csrfValue      = "traefik-webui"
	minPasswordLen = 12
	maxPasswordLen = 72 // bcrypt ignores anything beyond this
	sessionTTL     = 12 * time.Hour
	maxFailures    = 5
	lockout        = time.Minute
)

type stored struct {
	PasswordHash string `json:"password_hash"`
}

type attempt struct {
	failures int
	until    time.Time
}

// Auth holds the admin credential and active sessions.
type Auth struct {
	path string

	mu       sync.Mutex
	hash     []byte
	sessions map[string]time.Time
	attempts map[string]*attempt
	now      func() time.Time
}

// New loads the stored credential from dir, if one exists.
func New(dir string) (*Auth, error) {
	a := &Auth{
		path:     filepath.Join(dir, "auth.json"),
		sessions: map[string]time.Time{},
		attempts: map[string]*attempt{},
		now:      time.Now,
	}
	b, err := os.ReadFile(a.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		var s stored
		if err := json.Unmarshal(b, &s); err != nil {
			return nil, err
		}
		a.hash = []byte(s.PasswordHash)
	}
	return a, nil
}

// Routes registers the /api/auth/* endpoints.
func (a *Auth) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/status", a.status)
	mux.HandleFunc("POST /api/auth/setup", a.guard(a.setup))
	mux.HandleFunc("POST /api/auth/login", a.guard(a.login))
	mux.HandleFunc("POST /api/auth/logout", a.guard(a.logout))
}

// Require wraps a handler so only authenticated, CSRF-checked requests pass.
func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.authenticated(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !csrfOK(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// guard applies the CSRF check to the state-changing auth endpoints.
func (a *Auth) guard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !csrfOK(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

// csrfOK requires the custom header on unsafe methods (a cross-site form or
// fetch cannot set it without a CORS preflight) and rejects a mismatched Origin.
func csrfOK(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get(csrfHeader)), []byte(csrfValue)) != 1 {
		return false
	}
	if o := r.Header.Get("Origin"); o != "" {
		u, err := url.Parse(o)
		if err != nil || u.Host != r.Host {
			return false
		}
	}
	return true
}

func (a *Auth) authenticated(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.sessions[c.Value]
	if !ok {
		return false
	}
	if a.now().After(exp) {
		delete(a.sessions, c.Value)
		return false
	}
	return true
}

type creds struct {
	Password string `json:"password"`
}

func readCreds(w http.ResponseWriter, r *http.Request) (string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var c creds
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return "", false
	}
	return c.Password, true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func (a *Auth) status(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	setup := a.hash == nil
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]bool{
		"setupRequired": setup,
		"authenticated": a.authenticated(r),
	})
}

func (a *Auth) setup(w http.ResponseWriter, r *http.Request) {
	pw, ok := readCreds(w, r)
	if !ok {
		return
	}
	if len(pw) < minPasswordLen || len(pw) > maxPasswordLen {
		http.Error(w, "password must be 12 to 72 characters", http.StatusBadRequest)
		return
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.hash != nil {
		http.Error(w, "already configured", http.StatusConflict)
		return
	}
	if err := a.persist(h); err != nil {
		http.Error(w, "could not save credential", http.StatusInternalServerError)
		return
	}
	a.hash = h
	a.startSession(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (a *Auth) login(w http.ResponseWriter, r *http.Request) {
	pw, ok := readCreds(w, r)
	if !ok {
		return
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	a.mu.Lock()
	defer a.mu.Unlock()
	if a.hash == nil {
		http.Error(w, "setup required", http.StatusConflict)
		return
	}
	at := a.attempts[ip]
	if at == nil {
		at = &attempt{}
		a.attempts[ip] = at
	}
	if a.now().Before(at.until) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "too many attempts", http.StatusTooManyRequests)
		return
	}
	if bcrypt.CompareHashAndPassword(a.hash, []byte(pw)) != nil {
		at.failures++
		if at.failures >= maxFailures {
			at.failures = 0
			at.until = a.now().Add(lockout)
		}
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}
	delete(a.attempts, ip)
	a.startSession(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (a *Auth) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		a.mu.Lock()
		delete(a.sessions, c.Value)
		a.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
}

// startSession must be called with a.mu held.
func (a *Auth) startSession(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	tok := hex.EncodeToString(b)
	exp := a.now().Add(sessionTTL)
	a.sessions[tok] = exp
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tok,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteStrictMode,
	})
}

// persist writes the credential atomically with owner-only permissions.
func (a *Auth) persist(hash []byte) error {
	if err := os.MkdirAll(filepath.Dir(a.path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(stored{PasswordHash: string(hash)})
	if err != nil {
		return err
	}
	tmp := a.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, a.path)
}
