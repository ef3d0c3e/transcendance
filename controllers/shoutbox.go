package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
	"transcendance/models"
	"transcendance/views"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	// Time allowed to write a frame to a client
	shoutboxWriteTimeout = 10 * time.Second
	// Clients are pinged and their session checked this often
	shoutboxPingInterval = 30 * time.Second
)

// Close codes clients should not reconnect after
const (
	// The session expired, or the user logged out or was deleted
	shoutboxCloseUnauthorized websocket.StatusCode = 4401
	// The user has too many connections open
	shoutboxCloseTooManyConnections websocket.StatusCode = 4429
)

type ShoutboxController struct {
	DB       *gorm.DB
	Renderer *views.Renderer

	hub *shoutboxHub
	// Messages are saved and broadcast one at a time, so clients receive them in order
	postMu sync.Mutex
}

func NewShoutboxController(db *gorm.DB, renderer *views.Renderer) *ShoutboxController {
	return &ShoutboxController{
		DB:       db,
		Renderer: renderer,
		hub:      newShoutboxHub(),
	}
}

// Message as sent to clients
type shoutboxMessage struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

// Server to client frame
//
// Types:
//   - `history`: the latest `messages`, oldest first, and the user's `locale`, sent on connection
//   - `message`: a new `message` was posted
//   - `ack`: the client's last message was posted
//   - `error`: the client's last message was rejected, with the `error` code and `retryAfter` in milliseconds
//     when rate limited
type shoutboxEvent struct {
	Type       string            `json:"type"`
	Locale     string            `json:"locale,omitempty"`
	Messages   []shoutboxMessage `json:"messages,omitempty"`
	Message    *shoutboxMessage  `json:"message,omitempty"`
	Error      string            `json:"error,omitempty"`
	RetryAfter int64             `json:"retryAfter,omitempty"`
}

// Client to server frame, the only type is `send`
type shoutboxRequest struct {
	Type string  `json:"type"`
	Body *string `json:"body"`
}

// Decode a client frame, which must be exactly one valid request
func decodeShoutboxRequest(data []byte) (string, bool) {
	var req shoutboxRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return "", false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return "", false
	}
	if req.Type != "send" || req.Body == nil {
		return "", false
	}
	return *req.Body, true
}

// The latest messages, oldest first
func (sc *ShoutboxController) history(ctx context.Context) ([]shoutboxMessage, error) {
	var rows []struct {
		ID        uint
		Username  string
		Body      string
		CreatedAt time.Time
	}
	// Messages of deleted users are hidden
	if err := sc.DB.WithContext(ctx).
		Table("shoutbox_messages AS m").
		Select("m.id, u.username, m.body, m.created_at").
		Joins("JOIN users AS u ON u.id = m.user_id AND u.deleted_at IS NULL").
		Order("m.id DESC").
		Limit(shoutboxHistorySize).
		Scan(&rows).
		Error; err != nil {
		return nil, err
	}

	messages := make([]shoutboxMessage, len(rows))
	for i := range rows {
		row := &rows[len(rows)-1-i]
		messages[i] = shoutboxMessage{
			ID:        row.ID,
			Username:  row.Username,
			Body:      row.Body,
			CreatedAt: row.CreatedAt.Format(time.RFC3339),
		}
	}
	return messages, nil
}

// Whether the session the client connected with is still valid
func (sc *ShoutboxController) sessionValid(ctx context.Context, token string) bool {
	var count int64
	err := sc.DB.WithContext(ctx).
		Model(&models.UserSession{}).
		Joins("JOIN users ON users.id = user_sessions.user_id AND users.deleted_at IS NULL").
		Where("user_sessions.token = ? AND user_sessions.expires_at > ?", token, time.Now()).
		Count(&count).
		Error
	if err != nil {
		log.Printf("Shoutbox: failed to check session: %s", err)
		return false
	}
	return count > 0
}

func writeShoutboxEvent(ctx context.Context, conn *websocket.Conn, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, shoutboxWriteTimeout)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, payload)
}

// Queue an event for a single client
func queueShoutboxEvent(client *shoutboxClient, event shoutboxEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("Shoutbox: failed to encode event: %s", err)
		return
	}
	client.queue(payload)
}

// Validate, save and broadcast a message, returns false if the client must be disconnected
func (sc *ShoutboxController) post(ctx context.Context, client *shoutboxClient, token string, username string, raw string) bool {
	body, code := sanitizeShoutboxMessage(raw)
	if code != "" {
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: code})
		return true
	}
	if ok, wait := sc.hub.allow(client.userID); !ok {
		queueShoutboxEvent(client, shoutboxEvent{
			Type:       "error",
			Error:      shoutboxErrorRateLimited,
			RetryAfter: (wait + time.Millisecond - 1).Milliseconds(),
		})
		return true
	}
	// Logging out doesn't close other tabs' connections
	if !sc.sessionValid(ctx, token) {
		return false
	}

	sc.postMu.Lock()
	defer sc.postMu.Unlock()

	msg := models.ShoutboxMessage{UserID: client.userID, Body: body}
	if err := sc.DB.WithContext(ctx).Create(&msg).Error; err != nil {
		log.Printf("Shoutbox: failed to save message: %s", err)
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		return true
	}
	payload, err := json.Marshal(shoutboxEvent{
		Type: "message",
		Message: &shoutboxMessage{
			ID:        msg.ID,
			Username:  username,
			Body:      msg.Body,
			CreatedAt: msg.CreatedAt.Format(time.RFC3339),
		},
	})
	if err != nil {
		log.Printf("Shoutbox: failed to encode message: %s", err)
		return true
	}
	sc.hub.broadcast(payload)
	// Queued after the broadcast, so the sender has its message when the ack arrives
	queueShoutboxEvent(client, shoutboxEvent{Type: "ack"})
	return true
}

// Send queued events and keep the connection alive, until the context is cancelled or the connection fails
func (sc *ShoutboxController) writeLoop(ctx context.Context, conn *websocket.Conn, client *shoutboxClient, token string) {
	ticker := time.NewTicker(shoutboxPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-client.dropped:
			_ = conn.Close(websocket.StatusTryAgainLater, "too slow")
			return
		case payload := <-client.send:
			if err := writeShoutboxEvent(ctx, conn, payload); err != nil {
				return
			}
		case <-ticker.C:
			if !sc.sessionValid(ctx, token) {
				_ = conn.Close(shoutboxCloseUnauthorized, "unauthorized")
				return
			}
			pingCtx, cancel := context.WithTimeout(ctx, shoutboxWriteTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

// /api/shoutbox GET: websocket connection to the shoutbox
//
// See `shoutboxEvent` and `shoutboxRequest` for the protocol
func (sc *ShoutboxController) ApiShoutboxGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "You must be authenticated"})
		return
	}
	token, _ := c.Cookie("session_token")

	// Checks the origin matches the host, so other sites can't connect with the user's cookie
	conn, err := websocket.Accept(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(shoutboxReadLimit)

	client := newShoutboxClient(user.ID)
	if !sc.hub.register(client) {
		_ = conn.Close(shoutboxCloseTooManyConnections, "too many connections")
		return
	}
	defer sc.hub.unregister(client)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Registered first so no message is missed, clients ignore the ones they already have
	history, err := sc.history(ctx)
	if err != nil {
		log.Printf("Shoutbox: failed to load history: %s", err)
		_ = conn.Close(websocket.StatusInternalError, "internal error")
		return
	}
	payload, err := json.Marshal(shoutboxEvent{
		Type:     "history",
		Locale:   sc.Renderer.Localizer.GetLocale(c),
		Messages: history,
	})
	if err != nil {
		return
	}
	if err := writeShoutboxEvent(ctx, conn, payload); err != nil {
		return
	}

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		defer cancel()
		sc.writeLoop(ctx, conn, client, token)
	}()
	defer func() { <-writerDone }()
	defer cancel()

	for {
		kind, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if kind != websocket.MessageText {
			_ = conn.Close(websocket.StatusUnsupportedData, "text frames only")
			return
		}
		body, ok := decodeShoutboxRequest(data)
		if !ok {
			_ = conn.Close(websocket.StatusPolicyViolation, "malformed request")
			return
		}
		if !sc.post(ctx, client, token, user.Username, body) {
			_ = conn.Close(shoutboxCloseUnauthorized, "unauthorized")
			return
		}
	}
}
