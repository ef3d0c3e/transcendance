package main

import (
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	// __Host- cookies are only sent over HTTPS to this exact host
	sessionCookie = "__Host-admin_session"
	pendingCookie = "__Host-admin_pending"
)

//go:embed static
var staticFS embed.FS

var pages = template.Must(template.ParseFS(staticFS, "static/*.html"))

type server struct {
	keys       *keyring
	secret     []byte
	challenges *challengeStore
	// Limits sign-in requests per address
	challengeLimit *limiter
	sessionTTL     time.Duration
	// Shown in the command and on the page
	sshPort      string
	hostKeyPrint string
	now          func() time.Time
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /auth/static/", http.StripPrefix("/auth/static/", http.FileServerFS(static)))
	mux.HandleFunc("GET /auth/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /auth/{$}", s.indexGet)
	mux.HandleFunc("POST /auth/challenge", s.challengePost)
	mux.HandleFunc("GET /auth/status", s.statusGet)
	mux.HandleFunc("GET /auth/verify", s.verifyGet)
	mux.HandleFunc("GET /auth/whoami", s.whoamiGet)
	mux.HandleFunc("POST /auth/logout", s.logoutPost)
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; form-action 'self'")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// Address of the browser, the gateway is the only client and sets X-Real-IP
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func setCookie(w http.ResponseWriter, name string, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// The signed in admin, if the session is valid and its key still authorized
func (s *server) currentSession(r *http.Request) (session, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return session{}, false
	}
	sess, ok := parseSession(s.secret, cookie.Value, s.now())
	if !ok {
		return session{}, false
	}
	if _, ok := s.keys.lookup(sess.Key); !ok {
		return session{}, false
	}
	return sess, true
}

func (s *server) indexGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pages.ExecuteTemplate(w, "index.html", map[string]string{
		"Port":        s.sshPort,
		"Fingerprint": s.hostKeyPrint,
	})
}

func (s *server) challengePost(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !s.challengeLimit.allow(ip) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Too many sign-in requests, wait a minute."})
		return
	}
	c, err := s.challenges.create(ip, r.UserAgent())
	if errors.Is(err, errTooManyChallenges) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Too many sign-in requests, try again later."})
		return
	}
	if err != nil {
		logf("WARN cannot create a challenge: %s", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal error."})
		return
	}
	logf("INFO sign-in requested by %s", ip)
	setCookie(w, pendingCookie, c.Browser, c.Expires)
	writeJSON(w, http.StatusCreated, map[string]any{
		"code":    c.Code,
		"expires": c.Expires.Unix(),
	})
}

func (s *server) statusGet(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(pendingCookie)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": string(statusExpired)})
		return
	}
	c := s.challenges.claim(cookie.Value)
	if c.Status != statusPending {
		clearCookie(w, pendingCookie)
	}
	if c.Status == statusApproved {
		expires := s.now().Add(s.sessionTTL)
		value, err := signSession(s.secret, session{User: c.User, Key: c.Key, Expires: expires.Unix()})
		if err != nil {
			logf("WARN cannot sign a session: %s", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal error."})
			return
		}
		setCookie(w, sessionCookie, value, expires)
		logf("INFO %s signed in from %s", c.User, c.IP)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(c.Status)})
}

// Called by the gateway before proxying an admin tool
func (s *server) verifyGet(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.currentSession(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Remote-User", sess.User)
	w.WriteHeader(http.StatusOK)
}

func (s *server) whoamiGet(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.currentSession(r)
	if !ok {
		http.Redirect(w, r, "/auth/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pages.ExecuteTemplate(w, "whoami.html", map[string]string{
		"User":    sess.User,
		"Expires": time.Unix(sess.Expires, 0).UTC().Format("2006-01-02 15:04 UTC"),
	})
}

func (s *server) logoutPost(w http.ResponseWriter, r *http.Request) {
	clearCookie(w, sessionCookie)
	http.Redirect(w, r, "/auth/", http.StatusSeeOther)
}

// Printable form of browser supplied text, so it cannot drive the admin's terminal
func printable(text string, limit int) string {
	var out strings.Builder
	for _, r := range text {
		if out.Len() >= limit {
			out.WriteString("…")
			break
		}
		if r >= 0x20 && r < 0x7f {
			out.WriteRune(r)
		} else {
			out.WriteByte('?')
		}
	}
	return out.String()
}
