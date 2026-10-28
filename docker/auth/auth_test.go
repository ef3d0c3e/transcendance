package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// Time the tests move by hand
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type harness struct {
	t       *testing.T
	server  *server
	clock   *clock
	http    http.Handler
	sshAddr string
	hostKey ssh.PublicKey
	// Key of the authorized admin
	admin ssh.Signer
}

func newSigner(t *testing.T) (ssh.Signer, ssh.PublicKey) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(private)
	require.NoError(t, err)
	key, err := ssh.NewPublicKey(public)
	require.NoError(t, err)
	return signer, key
}

func newHarness(t *testing.T) *harness {
	dir := t.TempDir()
	c := &clock{now: time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)}
	keys := newKeyring(filepath.Join(dir, "authorized_keys"))
	secret, err := loadSecret(filepath.Join(dir, "session.key"))
	require.NoError(t, err)
	hostKey, err := loadHostKey(filepath.Join(dir, "ssh_host_ed25519_key"))
	require.NoError(t, err)

	s := &server{
		keys:           keys,
		secret:         secret,
		challenges:     newChallengeStore(5*time.Minute, c.Now),
		challengeLimit: newLimiter(10, time.Minute, c.Now),
		sessionTTL:     12 * time.Hour,
		sshPort:        "2233",
		hostKeyPrint:   ssh.FingerprintSHA256(hostKey.PublicKey()),
		now:            c.Now,
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		_ = s.serveSSH(listener, s.sshConfig(hostKey), newLimiter(1000, time.Minute, c.Now))
	}()

	admin, public := newSigner(t)
	_, err = keys.add(strings.TrimSpace(string(ssh.MarshalAuthorizedKey(public))) + " alice@laptop")
	require.NoError(t, err)

	return &harness{
		t:       t,
		server:  s,
		clock:   c,
		http:    s.routes(),
		sshAddr: listener.Addr().String(),
		hostKey: hostKey.PublicKey(),
		admin:   admin,
	}
}

func (h *harness) request(method string, target string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("X-Real-IP", "192.0.2.7")
	req.Header.Set("User-Agent", "TestBrowser/1.0")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.http.ServeHTTP(rec, req)
	return rec
}

func cookieNamed(t *testing.T, rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("no %s cookie", name)
	return nil
}

// Ask for a code like the sign-in page does
func (h *harness) challenge() (string, *http.Cookie) {
	rec := h.request(http.MethodPost, "/auth/challenge")
	require.Equal(h.t, http.StatusCreated, rec.Code)
	var body struct{ Code string }
	require.NoError(h.t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Regexp(h.t, `^[a-z]{4}-[a-z]{4}-[a-z]{4}$`, body.Code)
	pending := cookieNamed(h.t, rec, pendingCookie)
	require.True(h.t, pending.HttpOnly)
	require.True(h.t, pending.Secure)
	return body.Code, pending
}

func (h *harness) status(pending *http.Cookie) (string, *httptest.ResponseRecorder) {
	rec := h.request(http.MethodGet, "/auth/status", pending)
	require.Equal(h.t, http.StatusOK, rec.Code)
	var body struct{ Status string }
	require.NoError(h.t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body.Status, rec
}

// Run ssh -p PORT CODE@HOST and answer the prompt
func (h *harness) ssh(signer ssh.Signer, code string, answer string) (string, error) {
	client, err := ssh.Dial("tcp", h.sshAddr, &ssh.ClientConfig{
		User:            code,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.FixedHostKey(h.hostKey),
		Timeout:         5 * time.Second,
	})
	if err != nil {
		return "", err
	}
	defer func() { _ = client.Close() }()
	session, err := client.NewSession()
	require.NoError(h.t, err)
	var out bytes.Buffer
	session.Stdout = &out
	session.Stdin = strings.NewReader(answer)
	err = session.Run("")
	return out.String(), err
}

func TestDeviceSignIn(t *testing.T) {
	h := newHarness(t)
	code, pending := h.challenge()

	status, _ := h.status(pending)
	require.Equal(t, "pending", status)
	require.Equal(t, http.StatusUnauthorized, h.request(http.MethodGet, "/auth/verify").Code)

	out, err := h.ssh(h.admin, code, "y\n")
	require.NoError(t, err)
	require.Contains(t, out, "From:    192.0.2.7")
	require.Contains(t, out, "Browser: TestBrowser/1.0")
	require.Contains(t, out, "Sign in as alice@laptop? [y/N]")
	require.Contains(t, out, "Approved")

	status, rec := h.status(pending)
	require.Equal(t, "approved", status)
	sessionCookie := cookieNamed(t, rec, sessionCookie)
	require.True(t, sessionCookie.HttpOnly)
	require.True(t, sessionCookie.Secure)

	verify := h.request(http.MethodGet, "/auth/verify", sessionCookie)
	require.Equal(t, http.StatusOK, verify.Code)
	require.Equal(t, "alice@laptop", verify.Header().Get("Remote-User"))

	// A code is only used once
	status, _ = h.status(pending)
	require.Equal(t, "expired", status)
	_, err = h.ssh(h.admin, code, "y\n")
	require.Error(t, err)
}

func TestDeviceSignInDenied(t *testing.T) {
	h := newHarness(t)
	code, pending := h.challenge()

	out, err := h.ssh(h.admin, code, "n\n")
	var exit *ssh.ExitError
	require.ErrorAs(t, err, &exit)
	require.Equal(t, 1, exit.ExitStatus())
	require.Contains(t, out, "Denied")

	status, rec := h.status(pending)
	require.Equal(t, "denied", status)
	require.Empty(t, filterCookies(rec, sessionCookie))
}

func filterCookies(rec *httptest.ResponseRecorder, name string) []*http.Cookie {
	var found []*http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			found = append(found, cookie)
		}
	}
	return found
}

func TestUnknownKeyOrCodeIsRefused(t *testing.T) {
	h := newHarness(t)
	code, pending := h.challenge()

	stranger, _ := newSigner(t)
	_, err := h.ssh(stranger, code, "y\n")
	require.Error(t, err)

	_, err = h.ssh(h.admin, "abcd-efgh-jkmn", "y\n")
	require.Error(t, err)

	status, _ := h.status(pending)
	require.Equal(t, "pending", status)
}

func TestChallengeExpires(t *testing.T) {
	h := newHarness(t)
	code, pending := h.challenge()

	h.clock.advance(5 * time.Minute)
	_, err := h.ssh(h.admin, code, "y\n")
	require.Error(t, err)
	status, _ := h.status(pending)
	require.Equal(t, "expired", status)
}

func signIn(t *testing.T, h *harness) *http.Cookie {
	code, pending := h.challenge()
	_, err := h.ssh(h.admin, code, "yes\n")
	require.NoError(t, err)
	_, rec := h.status(pending)
	return cookieNamed(t, rec, sessionCookie)
}

func TestSessionEnds(t *testing.T) {
	h := newHarness(t)
	session := signIn(t, h)
	require.Equal(t, http.StatusOK, h.request(http.MethodGet, "/auth/verify", session).Code)

	h.clock.advance(12 * time.Hour)
	require.Equal(t, http.StatusUnauthorized, h.request(http.MethodGet, "/auth/verify", session).Code)
}

func TestRevokedKeyEndsSession(t *testing.T) {
	h := newHarness(t)
	session := signIn(t, h)

	removed, err := h.server.keys.remove("alice@laptop")
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	require.Equal(t, http.StatusUnauthorized, h.request(http.MethodGet, "/auth/verify", session).Code)
}

func TestTamperedSessionIsRefused(t *testing.T) {
	h := newHarness(t)
	cookie := signIn(t, h)
	valid, ok := parseSession(h.server.secret, cookie.Value, h.clock.Now())
	require.True(t, ok)

	// Same session signed with another secret
	forged, err := signSession([]byte(strings.Repeat("x", 32)), valid)
	require.NoError(t, err)
	// Longer session with the original signature
	longer, err := signSession(h.server.secret, session{User: valid.User, Key: valid.Key, Expires: valid.Expires + 3600})
	require.NoError(t, err)
	payload, _, _ := strings.Cut(longer, ".")
	_, signature, _ := strings.Cut(cookie.Value, ".")

	for _, value := range []string{forged, payload + "." + signature, payload, "garbage." + signature, ""} {
		forgedCookie := &http.Cookie{Name: sessionCookie, Value: value}
		require.Equal(t, http.StatusUnauthorized, h.request(http.MethodGet, "/auth/verify", forgedCookie).Code, value)
	}
}

func TestLogout(t *testing.T) {
	h := newHarness(t)
	signIn(t, h)

	rec := h.request(http.MethodPost, "/auth/logout")
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, -1, cookieNamed(t, rec, sessionCookie).MaxAge)
}

func TestChallengeRateLimit(t *testing.T) {
	h := newHarness(t)
	for range 10 {
		h.challenge()
	}
	require.Equal(t, http.StatusTooManyRequests, h.request(http.MethodPost, "/auth/challenge").Code)

	h.clock.advance(time.Minute)
	h.challenge()
}

func TestPromptShowsOnlyPrintableText(t *testing.T) {
	require.Equal(t, "evil?[2Jagent", printable("evil\x1b[2Jagent", 64))
	require.Equal(t, "caf?", printable("café", 64))
	require.Equal(t, "abc…", printable("abcdef", 3))
}

func TestAddKey(t *testing.T) {
	keys := newKeyring(filepath.Join(t.TempDir(), "authorized_keys"))
	_, public := newSigner(t)
	line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(public)))

	_, err := keys.add(line)
	require.ErrorContains(t, err, "comment")
	_, err = keys.add("not a key")
	require.Error(t, err)

	added, err := keys.add(line + " bob@desk\x1b]")
	require.NoError(t, err)
	require.Equal(t, "bob@desk", added.User)
	_, err = keys.add(line + " bob@other")
	require.ErrorContains(t, err, "already")

	found, ok := keys.lookup(ssh.FingerprintSHA256(public))
	require.True(t, ok)
	require.Equal(t, "bob@desk", found.User)
}

func TestBrokenKeyFileAllowsNobody(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authorized_keys")
	keys := newKeyring(path)
	_, public := newSigner(t)
	_, err := keys.add(strings.TrimSpace(string(ssh.MarshalAuthorizedKey(public))) + " carol@box")
	require.NoError(t, err)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = file.WriteString("ssh-ed25519 broken\n")
	require.NoError(t, err)
	require.NoError(t, file.Close())

	_, ok := keys.lookup(ssh.FingerprintSHA256(public))
	require.False(t, ok)
}

func TestCodesAreDistinct(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		code, err := newCode()
		require.NoError(t, err)
		require.False(t, seen[code])
		require.False(t, strings.ContainsAny(code, "ilo"))
		seen[code] = true
	}
}
