package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func chatRequest(t *testing.T, conn *websocket.Conn, request map[string]string) {
	payload, err := json.Marshal(request)
	require.NoError(t, err)
	shoutboxWrite(t, conn, string(payload))
}

func chatSendPrivate(t *testing.T, conn *websocket.Conn, to string, body string) {
	chatRequest(t, conn, map[string]string{"type": "send", "to": to, "body": body})
}

// Read events until one of the given type, skipping others
func chatReadUntil(t *testing.T, conn *websocket.Conn, kind string) shoutboxEvent {
	for {
		event := shoutboxRead(t, conn)
		if event.Type == kind {
			return event
		}
	}
}

// Create users and connect each of them
func chatUsers(t *testing.T, router *gin.Engine, server *httptest.Server, names ...string) ([]*http.Cookie, []*websocket.Conn) {
	cookies := make([]*http.Cookie, len(names))
	conns := make([]*websocket.Conn, len(names))
	for i, name := range names {
		createAccount(t, router, name, "foo-bar-baz-qux", 0)
		cookies[i] = login(t, router, name, "foo-bar-baz-qux")
		conns[i], _ = shoutboxConnect(t, server, cookies[i])
	}
	return cookies, conns
}

// The history sent on a new connection
func chatHistory(t *testing.T, server *httptest.Server, cookie *http.Cookie) shoutboxEvent {
	conn, _, err := shoutboxDial(t, server, cookie, "")
	require.NoError(t, err)
	event := shoutboxRead(t, conn)
	require.Equal(t, "history", event.Type)
	_ = conn.Close(websocket.StatusNormalClosure, "")
	return event
}

type chatNotification struct {
	ID    uint
	Type  string
	Read  bool
	Title string
	Desc  string
}

func chatNotifications(t *testing.T, router *gin.Engine, cookie *http.Cookie) []chatNotification {
	req := httptest.NewRequest(http.MethodGet, "/api/notifications?filter=unread", nil)
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)

	var body struct {
		Notifications []chatNotification
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	var private []chatNotification
	for _, notif := range body.Notifications {
		if notif.Type == "private-message" {
			notif.Title = stripIsolation(notif.Title)
			notif.Desc = stripIsolation(notif.Desc)
			private = append(private, notif)
		}
	}
	return private
}

func TestChatPrivateMessage(t *testing.T) {
	router, server := newShoutboxServer(t)
	cookies, conns := chatUsers(t, router, server, "pm_a", "pm_b", "pm_c")
	otherTab, _ := shoutboxConnect(t, server, cookies[0])

	chatSendPrivate(t, conns[0], "pm_b", "  hello\nb  ")
	for _, conn := range []*websocket.Conn{conns[0], otherTab, conns[1]} {
		event := shoutboxRead(t, conn)
		require.Equal(t, "private", event.Type)
		require.Equal(t, "pm_a", event.Private.From)
		require.Equal(t, "pm_b", event.Private.To)
		require.Equal(t, "hello b", event.Private.Body)
	}
	require.Equal(t, "ack", shoutboxRead(t, conns[0]).Type)

	// Others only get what comes next
	shoutboxSend(t, conns[1], "public")
	event := shoutboxRead(t, conns[2])
	require.Equal(t, "message", event.Type)
	require.Equal(t, "public", event.Message.Body)

	// Listed on both sides, unread by the recipient
	history := chatHistory(t, server, cookies[1])
	require.Len(t, history.Conversations, 1)
	require.Equal(t, "pm_a", history.Conversations[0].Username)
	require.Equal(t, "hello b", history.Conversations[0].Last.Body)
	require.Equal(t, 1, history.Conversations[0].Unread)
	history = chatHistory(t, server, cookies[0])
	require.Len(t, history.Conversations, 1)
	require.Equal(t, "pm_b", history.Conversations[0].Username)
	require.Zero(t, history.Conversations[0].Unread)
	require.Empty(t, chatHistory(t, server, cookies[2]).Conversations)

	// Viewing marks it read on every tab
	recipientTab, _ := shoutboxConnect(t, server, cookies[1])
	chatRequest(t, conns[1], map[string]string{"type": "view", "with": "pm_a"})
	require.Equal(t, "pm_a", chatReadUntil(t, recipientTab, "read").With)
	conversation := chatReadUntil(t, conns[1], "conversation")
	require.Equal(t, "pm_a", conversation.With)
	require.Len(t, conversation.Privates, 1)
	require.Equal(t, "hello b", conversation.Privates[0].Body)
	require.Zero(t, chatHistory(t, server, cookies[1]).Conversations[0].Unread)

	// Replies come after, oldest first
	chatSendPrivate(t, conns[1], "pm_a", "hi a")
	require.Equal(t, "ack", chatReadUntil(t, conns[1], "ack").Type)
	chatRequest(t, conns[1], map[string]string{"type": "view", "with": "pm_a"})
	conversation = chatReadUntil(t, conns[1], "conversation")
	require.Len(t, conversation.Privates, 2)
	require.Equal(t, "hi a", conversation.Privates[1].Body)
}

func TestChatPrivateErrors(t *testing.T) {
	router, server := newShoutboxServer(t)
	_, conns := chatUsers(t, router, server, "pm_err")

	for _, test := range []struct {
		request map[string]string
		error   string
	}{
		{map[string]string{"type": "send", "to": "pm_nobody", "body": "hi"}, "user-not-found"},
		{map[string]string{"type": "send", "to": "pm_err", "body": "hi"}, "self"},
		{map[string]string{"type": "send", "to": "pm_nobody", "body": " "}, "empty"},
		{map[string]string{"type": "view", "with": "pm_nobody"}, "user-not-found"},
		{map[string]string{"type": "ignore", "username": "pm_err", "duration": "hour"}, "self"},
		{map[string]string{"type": "unignore", "username": "pm_nobody"}, "user-not-found"},
	} {
		chatRequest(t, conns[0], test.request)
		event := shoutboxRead(t, conns[0])
		require.Equal(t, "error", event.Type, test.request)
		require.Equal(t, test.error, event.Error, test.request)
	}

	// Closing a conversation has no reply
	chatRequest(t, conns[0], map[string]string{"type": "view", "with": ""})
	shoutboxSend(t, conns[0], "still here")
	require.Equal(t, "message", shoutboxRead(t, conns[0]).Type)
}

func TestChatIgnore(t *testing.T) {
	router, server := newShoutboxServer(t)
	cookies, conns := chatUsers(t, router, server, "ign_a", "ign_b", "ign_c")
	ignorer, ignored, other := conns[1], conns[0], conns[2]

	chatRequest(t, ignorer, map[string]string{"type": "ignore", "username": "ign_a", "duration": "hour"})
	event := shoutboxRead(t, ignorer)
	require.Equal(t, "ignored", event.Type)
	require.Len(t, event.Ignored, 1)
	require.Equal(t, "ign_a", event.Ignored[0].Username)
	until, err := time.Parse(time.RFC3339, event.Ignored[0].Until)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(time.Hour), until, time.Minute)
	require.Equal(t, "ack", shoutboxRead(t, ignorer).Type)

	// Their shoutbox messages are hidden, live and in the history
	shoutboxSend(t, ignored, "can you hear me")
	require.Equal(t, "can you hear me", chatReadUntil(t, other, "message").Message.Body)
	shoutboxSend(t, other, "after")
	require.Equal(t, "after", chatReadUntil(t, ignorer, "message").Message.Body)
	for _, message := range chatHistory(t, server, cookies[1]).Messages {
		require.NotEqual(t, "ign_a", message.Username)
	}
	require.Len(t, chatHistory(t, server, cookies[1]).Ignored, 1)

	// Their private messages are rejected, not the other way around
	chatSendPrivate(t, ignored, "ign_b", "please")
	event = chatReadUntil(t, ignored, "error")
	require.Equal(t, "blocked", event.Error)
	chatSendPrivate(t, ignorer, "ign_a", "no")
	require.Equal(t, "no", chatReadUntil(t, ignored, "private").Private.Body)

	// Lifted early
	chatRequest(t, ignorer, map[string]string{"type": "unignore", "username": "ign_a"})
	event = chatReadUntil(t, ignorer, "ignored")
	require.Empty(t, event.Ignored)
	chatSendPrivate(t, ignored, "ign_b", "please")
	require.Equal(t, "please", chatReadUntil(t, ignorer, "private").Private.Body)
	shoutboxSend(t, ignored, "back")
	require.Equal(t, "back", chatReadUntil(t, ignorer, "message").Message.Body)
	require.Empty(t, chatHistory(t, server, cookies[1]).Ignored)
}

func TestChatNotifications(t *testing.T) {
	router, server := newShoutboxServer(t)
	cookies, conns := chatUsers(t, router, server, "notif_a", "notif_b", "notif_c")
	sender, recipient := conns[0], conns[1]

	send := func(conn *websocket.Conn, to string, body string) {
		chatSendPrivate(t, conn, to, body)
		chatReadUntil(t, conn, "ack")
	}

	// A single notification for the conversation
	send(sender, "notif_b", "one")
	notifications := chatNotifications(t, router, cookies[1])
	require.Len(t, notifications, 1)
	id := notifications[0].ID
	require.Equal(t, chatNotification{
		ID:    id,
		Type:  "private-message",
		Title: "New message from notif_a",
		Desc:  "one",
	}, notifications[0])
	send(sender, "notif_b", "two")
	send(sender, "notif_b", "three")
	require.Equal(t, []chatNotification{{
		ID:    id,
		Type:  "private-message",
		Title: "3 new messages from notif_a",
		Desc:  "three",
	}}, chatNotifications(t, router, cookies[1]))

	// One per sender
	send(conns[2], "notif_b", "hey")
	require.Len(t, chatNotifications(t, router, cookies[1]), 2)

	// Opens the conversation on the sender's profile
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/notification/%d", id), nil)
	req.AddCookie(cookies[1])
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	require.Equal(t, http.StatusSeeOther, res.Code)
	require.Equal(t, "/profile/notif_a?chat=notif_a", res.Header().Get("Location"))

	// Read with the conversation
	chatRequest(t, recipient, map[string]string{"type": "view", "with": "notif_a"})
	chatReadUntil(t, recipient, "conversation")
	notifications = chatNotifications(t, router, cookies[1])
	require.Len(t, notifications, 1)
	require.Equal(t, "New message from notif_c", notifications[0].Title)

	// None while it is open
	send(sender, "notif_b", "four")
	require.Len(t, chatNotifications(t, router, cookies[1]), 1)

	// A new one once closed, or once the previous one was read
	chatRequest(t, recipient, map[string]string{"type": "view", "with": ""})
	send(sender, "notif_b", "five")
	notifications = chatNotifications(t, router, cookies[1])
	require.Len(t, notifications, 2)
	require.Equal(t, "New message from notif_a", notifications[0].Title)
	require.Equal(t, "five", notifications[0].Desc)
}
