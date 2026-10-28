package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Enabled         bool
	Username        string
	Origin          string
	PasswordHash    string
	SessionLifetime time.Duration
}

type session struct{ expires time.Time }
type Auth struct {
	config      Config
	secure      bool
	mu          sync.Mutex
	sessions    map[[32]byte]session
	nextAttempt time.Time
	busy        bool
	verify      func(string) error
}

func New(c Config) (*Auth, error) {
	a := &Auth{config: c, sessions: make(map[[32]byte]session)}
	if !c.Enabled {
		return a, nil
	}

	if c.Username == "" || strings.ContainsAny(c.Username, "\x00\r\n") {
		return nil, errors.New("auth requires a username")
	}

	origin, err := url.Parse(c.Origin)
	if err != nil || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return nil, errors.New("invalid auth origin")
	}

	ip := net.ParseIP(origin.Hostname())
	loopback := origin.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
	if origin.Scheme != "https" && !(origin.Scheme == "http" && loopback) {
		return nil, errors.New("auth requires HTTPS except on loopback")
	}

	a.secure = strings.HasPrefix(c.Origin, "https://")
	cost, err := bcrypt.Cost([]byte(c.PasswordHash))
	if err != nil || len(c.PasswordHash) != 60 || cost < bcrypt.DefaultCost || cost > 14 {
		return nil, errors.New("auth requires a valid bcrypt hash with cost 10 to 14")
	}

	if c.SessionLifetime <= 0 {
		a.config.SessionLifetime = time.Hour
	}

	a.verify = func(password string) error {
		return bcrypt.CompareHashAndPassword([]byte(a.config.PasswordHash), []byte(password))
	}

	return a, nil
}

// Middleware will manage middle auth
func (a *Auth) Middleware(next http.Handler) http.Handler {
	if !a.config.Enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path == "/auth/login" {
			if r.Method != http.MethodPost {
				w.Header().Set("Allow", "POST")
				http.Error(w, "Method not allowed", 405)
				return
			}

			if !a.validOrigin(r) {
				http.Error(w, "Forbidden", 403)
				return
			}

			a.login(w, r)
			return
		}

		if r.Method == http.MethodGet && (r.URL.Path == "/login" || strings.HasPrefix(r.URL.Path, "/static/")) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("openlinkhub_session")
		if err != nil || !a.validSession(cookie.Value) {
			if r.Method == http.MethodGet &&
				r.URL.Path != "/api" &&
				!strings.HasPrefix(r.URL.Path, "/api/") &&
				strings.Contains(r.Header.Get("Accept"), "text/html") {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead && !a.validOrigin(r) {
			http.Error(w, "Forbidden", 403)
			return
		}

		if r.URL.Path == "/auth/logout" {
			if r.Method != http.MethodPost {
				w.Header().Set("Allow", "POST")
				http.Error(w, "Method not allowed", 405)
				return
			}
			a.mu.Lock()
			delete(a.sessions, sha256.Sum256([]byte(cookie.Value)))
			a.mu.Unlock()
			http.SetCookie(w, a.cookie("", -1))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// validOrigin will validate Origin from headers
func (a *Auth) validOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == a.config.Origin
}

// cookie will generate new cookie
func (a *Auth) cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "openlinkhub_session",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	}
}

// validSession will validate user session
func (a *Auth) validSession(token string) bool {
	if len(token) != 64 {
		return false
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	key := sha256.Sum256([]byte(token))
	s, ok := a.sessions[key]
	if !ok {
		return false
	}

	if !time.Now().Before(s.expires) {
		delete(a.sessions, key)
		return false
	}

	return true
}

// login will handle user login
func (a *Auth) login(w http.ResponseWriter, r *http.Request) {
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		http.Error(w, "JSON required", 415)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if dec.Decode(&credentials) != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	var extra interface{}
	if dec.Decode(&extra) != io.EOF {
		http.Error(w, "Invalid request", 400)
		return
	}

	a.mu.Lock()
	if a.busy || time.Now().Before(a.nextAttempt) {
		a.mu.Unlock()
		w.Header().Set("Retry-After", "2")
		http.Error(w, "Try again later", 429)
		return
	}
	a.busy = true
	a.nextAttempt = time.Now().Add(2 * time.Second)
	a.mu.Unlock()

	defer func() { a.mu.Lock(); a.busy = false; a.mu.Unlock() }()

	if credentials.Username != a.config.Username || credentials.Password == "" || len(credentials.Password) > 72 || strings.ContainsRune(credentials.Password, 0) {
		http.Error(w, "Invalid credentials", 401)
		return
	}

	if err := a.verify(credentials.Password); err != nil {
		http.Error(w, "Invalid credentials", 401)
		return
	}

	credentials.Password = ""
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		http.Error(w, "Unable to create session", 500)
		return
	}

	token := hex.EncodeToString(random[:])
	a.mu.Lock()
	now := time.Now()
	for key, s := range a.sessions {
		if !now.Before(s.expires) {
			delete(a.sessions, key)
		}
	}

	if old, err := r.Cookie("openlinkhub_session"); err == nil {
		delete(a.sessions, sha256.Sum256([]byte(old.Value)))
	}

	if len(a.sessions) >= 64 {
		a.mu.Unlock()
		http.Error(w, "Session limit reached", 429)
		return
	}

	a.sessions[sha256.Sum256([]byte(token))] = session{now.Add(a.config.SessionLifetime)}
	a.mu.Unlock()

	http.SetCookie(w, a.cookie(token, int(a.config.SessionLifetime.Seconds())))
	w.WriteHeader(http.StatusNoContent)
}
