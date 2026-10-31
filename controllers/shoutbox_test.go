package controllers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"transcendance/tests"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type shoutboxMessage struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Body     string `json:"body"`
}

type chatPrivateMessage struct {
	ID   uint   `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Body string `json:"body"`
}

type chatConversation struct {
	Username string             `json:"username"`
	Last     chatPrivateMessage `json:"last"`
	Unread   int                `json:"unread"`
}

type chatIgnoredUser struct {
	Username string `json:"username"`
	Until    string `json:"until"`
}

type shoutboxEvent struct {
	Type          string               `json:"type"`
	Locale        string               `json:"locale"`
	Messages      []shoutboxMessage    `json:"messages"`
	Message       *shoutboxMessage     `json:"message"`
	Conversations []chatConversation   `json:"conversations"`
	Private       *chatPrivateMessage  `json:"private"`
	With          string               `json:"with"`
	Privates      []chatPrivateMessage `json:"privates"`
	Ignored       []chatIgnoredUser    `json:"ignored"`
	Error         string               `json:"error"`
	RetryAfter    int64                `json:"retryAfter"`
}

func shoutboxDial(t *testing.T, server *httptest.Server, cookie *http.Cookie, origin string, extraCookies ...*http.Cookie) (*websocket.Conn, *http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	header := http.Header{}
	if cookie != nil {
		header.Add("Cookie", cookie.String())
	}
	for _, extra := range extraCookies {
		header.Add("Cookie", extra.String())
	}
	if origin == "" {
		origin = server.URL
	}
	header.Set("Origin", origin)
	conn, res, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/shoutbox", &websocket.DialOptions{
		HTTPHeader: header,
	})
	if conn != nil {
		t.Cleanup(func() { _ = conn.CloseNow() })
	}
	return conn, res, err
}

// Connect and read the history
func shoutboxConnect(t *testing.T, server *httptest.Server, cookie *http.Cookie) (*websocket.Conn, []shoutboxMessage) {
	conn, _, err := shoutboxDial(t, server, cookie, "")
	require.NoError(t, err)
	event := shoutboxRead(t, conn)
	require.Equal(t, "history", event.Type)
	return conn, event.Messages
}

func shoutboxRead(t *testing.T, conn *websocket.Conn) shoutboxEvent {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	kind, data, err := conn.Read(ctx)
	require.NoError(t, err)
	require.Equal(t, websocket.MessageText, kind)
	var event shoutboxEvent
	require.NoError(t, json.Unmarshal(data, &event))
	return event
}

func shoutboxWrite(t *testing.T, conn *websocket.Conn, payload string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, conn.Write(ctx, websocket.MessageText, []byte(payload)))
}

func shoutboxSend(t *testing.T, conn *websocket.Conn, body string) {
	payload, err := json.Marshal(map[string]string{"type": "send", "body": body})
	require.NoError(t, err)
	shoutboxWrite(t, conn, string(payload))
}

// Expect the connection to be closed by the server with the given code
func shoutboxExpectClose(t *testing.T, conn *websocket.Conn, code websocket.StatusCode) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			require.Equal(t, code, websocket.CloseStatus(err), err.Error())
			return
		}
	}
}

func newShoutboxServer(t *testing.T) (*gin.Engine, *httptest.Server) {
	router := tests.Run(tests.Data, tests.DB)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return router, server
}

func TestShoutboxUnauthenticated(t *testing.T) {
	_, server := newShoutboxServer(t)

	_, res, err := shoutboxDial(t, server, nil, "")
	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestShoutboxCrossOrigin(t *testing.T) {
	router, server := newShoutboxServer(t)
	createAccount(t, router, "shout_origin", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "shout_origin", "foo-bar-baz-qux")

	_, res, err := shoutboxDial(t, server, cookie, "https://evil.example")
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestShoutboxOriginBehindGateway(t *testing.T) {
	router, server := newShoutboxServer(t)
	createAccount(t, router, "shout_gateway", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "shout_gateway", "foo-bar-baz-qux")

	// The gateway forwards the host without the port the browser used
	dial := func(origin string) (*websocket.Conn, *http.Response, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		header := http.Header{}
		header.Add("Cookie", cookie.String())
		header.Set("Origin", origin)
		conn, res, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/shoutbox", &websocket.DialOptions{
			HTTPHeader: header,
			Host:       "example.com",
		})
		if conn != nil {
			t.Cleanup(func() { _ = conn.CloseNow() })
		}
		return conn, res, err
	}

	conn, _, err := dial("https://example.com:8443")
	require.NoError(t, err)
	require.Equal(t, "history", shoutboxRead(t, conn).Type)

	_, res, err := dial("https://evil.example:8443")
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, res.StatusCode)
	_, res, err = dial("https://example.com.evil.example:8443")
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestShoutboxBroadcast(t *testing.T) {
	router, server := newShoutboxServer(t)
	createAccount(t, router, "shout_a", "foo-bar-baz-qux", 0)
	createAccount(t, router, "shout_b", "foo-bar-baz-qux", 0)
	cookieA := login(t, router, "shout_a", "foo-bar-baz-qux")
	cookieB := login(t, router, "shout_b", "foo-bar-baz-qux")

	connA, _ := shoutboxConnect(t, server, cookieA)
	connB, _ := shoutboxConnect(t, server, cookieB)

	shoutboxSend(t, connA, "  hello\nworld\u202E  ")
	for _, conn := range []*websocket.Conn{connA, connB} {
		event := shoutboxRead(t, conn)
		require.Equal(t, "message", event.Type)
		require.Equal(t, "shout_a", event.Message.Username)
		require.Equal(t, "hello world", event.Message.Body)
	}
	require.Equal(t, "ack", shoutboxRead(t, connA).Type)

	// Later connections get it in the history
	_, history := shoutboxConnect(t, server, cookieB)
	require.NotEmpty(t, history)
	last := history[len(history)-1]
	require.Equal(t, "shout_a", last.Username)
	require.Equal(t, "hello world", last.Body)
}

func TestShoutboxInvalidMessages(t *testing.T) {
	router, server := newShoutboxServer(t)
	createAccount(t, router, "shout_invalid", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "shout_invalid", "foo-bar-baz-qux")

	conn, _ := shoutboxConnect(t, server, cookie)

	shoutboxSend(t, conn, " \n\u200B ")
	event := shoutboxRead(t, conn)
	require.Equal(t, "error", event.Type)
	require.Equal(t, "empty", event.Error)

	shoutboxSend(t, conn, strings.Repeat("a", 501))
	event = shoutboxRead(t, conn)
	require.Equal(t, "error", event.Type)
	require.Equal(t, "too-long", event.Error)

	// Still connected
	shoutboxSend(t, conn, strings.Repeat("a", 500))
	require.Equal(t, "message", shoutboxRead(t, conn).Type)
	require.Equal(t, "ack", shoutboxRead(t, conn).Type)
}

func TestShoutboxMalformedRequests(t *testing.T) {
	router, server := newShoutboxServer(t)
	createAccount(t, router, "shout_malformed", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "shout_malformed", "foo-bar-baz-qux")

	for _, payload := range []string{
		`not json`,
		`{"type":"send"}`,
		`{"type":"send","body":42}`,
		`{"type":"delete","body":"hi"}`,
		`{"type":"send","body":"hi","extra":true}`,
		`{"type":"send","body":"hi","with":"someone"}`,
		`{"type":"view"}`,
		`{"type":"view","with":"someone","body":"hi"}`,
		`{"type":"ignore","username":"someone"}`,
		`{"type":"ignore","username":"someone","duration":"forever"}`,
		`{"type":"unignore"}`,
		`{"type":"send","body":"hi"}{"type":"send","body":"hi"}`,
	} {
		conn, _ := shoutboxConnect(t, server, cookie)
		shoutboxWrite(t, conn, payload)
		shoutboxExpectClose(t, conn, websocket.StatusPolicyViolation)
	}

	// Binary frames
	conn, _ := shoutboxConnect(t, server, cookie)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, conn.Write(ctx, websocket.MessageBinary, []byte(`{"type":"send","body":"hi"}`)))
	shoutboxExpectClose(t, conn, websocket.StatusUnsupportedData)

	// Oversized frames
	conn, _ = shoutboxConnect(t, server, cookie)
	shoutboxSend(t, conn, strings.Repeat("a", 10_000))
	shoutboxExpectClose(t, conn, websocket.StatusMessageTooBig)
}

func TestShoutboxRateLimit(t *testing.T) {
	router, server := newShoutboxServer(t)
	createAccount(t, router, "shout_spam", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "shout_spam", "foo-bar-baz-qux")

	// Limits are per user, not per connection
	conns := make([]*websocket.Conn, 2)
	for i := range conns {
		conns[i], _ = shoutboxConnect(t, server, cookie)
	}

	posted := 0
	for i := range 6 {
		conn := conns[i%2]
		shoutboxSend(t, conn, "spam")
		var event shoutboxEvent
		// Skip broadcasts until the reply to this message
		for event = shoutboxRead(t, conn); event.Type == "message"; event = shoutboxRead(t, conn) {
		}
		if event.Type == "ack" {
			posted++
			continue
		}
		require.Equal(t, "error", event.Type)
		require.Equal(t, "rate-limited", event.Error)
		require.Positive(t, event.RetryAfter)
	}
	require.Equal(t, 5, posted)
}

func TestShoutboxLoggedOut(t *testing.T) {
	router, server := newShoutboxServer(t)
	createAccount(t, router, "shout_logout", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "shout_logout", "foo-bar-baz-qux")

	conn, _ := shoutboxConnect(t, server, cookie)

	req := httptest.NewRequest(http.MethodGet, "/api/logout", nil)
	req.AddCookie(cookie)
	router.ServeHTTP(httptest.NewRecorder(), req)

	shoutboxSend(t, conn, "still here?")
	shoutboxExpectClose(t, conn, 4401)
}

func TestShoutboxTooManyConnections(t *testing.T) {
	router, server := newShoutboxServer(t)
	createAccount(t, router, "shout_tabs", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "shout_tabs", "foo-bar-baz-qux")

	for range 8 {
		shoutboxConnect(t, server, cookie)
	}
	conn, _, err := shoutboxDial(t, server, cookie, "")
	require.NoError(t, err)
	shoutboxExpectClose(t, conn, 4429)
}

func TestShoutboxLocale(t *testing.T) {
	router, server := newShoutboxServer(t)
	createAccount(t, router, "shout_locale", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "shout_locale", "foo-bar-baz-qux")

	conn, _, err := shoutboxDial(t, server, cookie, "", &http.Cookie{Name: "lang", Value: "fr"})
	require.NoError(t, err)
	event := shoutboxRead(t, conn)
	require.Equal(t, "history", event.Type)
	require.Equal(t, "fr", event.Locale)
}
