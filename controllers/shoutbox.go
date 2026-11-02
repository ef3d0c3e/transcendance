package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
	"transcendance/models"
	"transcendance/views"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
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
	// Held while a user's ignores are saved or loaded and handed to the hub, so it never gets an outdated list
	ignoreMu sync.Mutex
}

func NewShoutboxController(db *gorm.DB, renderer *views.Renderer) *ShoutboxController {
	return &ShoutboxController{
		DB:       db,
		Renderer: renderer,
		hub:      newShoutboxHub(),
	}
}

type shoutboxMessage struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

// Server to client frame, by type:
//   - `history`: on connection, the latest shoutbox `messages` (oldest first), the user's `conversations`
//     (latest first), the users they `ignored`, their `locale`, and whether they are a `moderator`
//   - `message`: a new shoutbox `message`
//   - `private`: a `private` message sent to or by the user
//   - `conversation`: in reply to `view`, the latest `privates` exchanged `with` a user, oldest first
//   - `read`: the conversation `with` a user was read in another tab
//   - `ignored`: the users the user now `ignored`
//   - `deleted`: a moderator deleted the shoutbox message `id`
//   - `ack`: the last request succeeded
//   - `error`: the last request was rejected with the `error` code, and `retryAfter` in milliseconds when rate limited
type shoutboxEvent struct {
	Type          string               `json:"type"`
	Locale        string               `json:"locale,omitempty"`
	Messages      []shoutboxMessage    `json:"messages,omitempty"`
	Message       *shoutboxMessage     `json:"message,omitempty"`
	Conversations []chatConversation   `json:"conversations,omitempty"`
	Private       *chatPrivateMessage  `json:"private,omitempty"`
	With          string               `json:"with,omitempty"`
	Privates      []chatPrivateMessage `json:"privates,omitempty"`
	Ignored       []chatIgnoredUser    `json:"ignored,omitempty"`
	Error         string               `json:"error,omitempty"`
	RetryAfter    int64                `json:"retryAfter,omitempty"`
	ID            uint                 `json:"id,omitempty"`
	Moderator     bool                 `json:"moderator,omitempty"`
}

// Client to server frame, by type:
//   - `send`: post `body` to the shoutbox, or privately `to` a user
//   - `view`: show the conversation `with` a user, marked read and returned (none when empty). New messages of a
//     shown conversation are read right away and don't notify
//   - `ignore`: ignore a user (`username`) for a `duration`: `hour`, `day` or `week`
//   - `unignore`: stop ignoring a user (`username`)
//
// Moderators only:
//   - `delete`: delete the shoutbox message `id` for everyone
//   - `mute`: refuse the shoutbox messages of a user (`username`) for a `duration`, as for `ignore`
//   - `unmute`: let a muted user (`username`) post again
type shoutboxRequest struct {
	Type     string  `json:"type"`
	Body     *string `json:"body"`
	To       *string `json:"to"`
	With     *string `json:"with"`
	Username *string `json:"username"`
	Duration *string `json:"duration"`
	ID       *uint   `json:"id"`
}

// decodeShoutboxRequest accepts exactly one valid request with only the fields of its type
func decodeShoutboxRequest(data []byte) (shoutboxRequest, bool) {
	var req shoutboxRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return req, false
	}

	_, validDuration := chatIgnoreDurations[deref(req.Duration)]
	switch req.Type {
	case "send":
		return req, req.Body != nil && req.With == nil && req.Username == nil && req.Duration == nil && req.ID == nil
	case "view":
		return req, req.With != nil && req.Body == nil && req.To == nil && req.Username == nil && req.Duration == nil &&
			req.ID == nil
	case "ignore", "mute":
		return req, validDuration && req.Username != nil && req.Body == nil && req.To == nil && req.With == nil &&
			req.ID == nil
	case "unignore", "unmute":
		return req, req.Username != nil && req.Body == nil && req.To == nil && req.With == nil && req.Duration == nil &&
			req.ID == nil
	case "delete":
		return req, req.ID != nil && req.Body == nil && req.To == nil && req.With == nil && req.Username == nil &&
			req.Duration == nil
	default:
		return req, false
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// The latest messages, oldest first, without those of users the user ignores
func (sc *ShoutboxController) history(ctx context.Context, userID uint) ([]shoutboxMessage, error) {
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
		Where("NOT EXISTS (SELECT 1 FROM user_ignores AS i WHERE i.user_id = ? AND i.ignored_id = m.user_id AND i.until > ?)",
			userID, time.Now()).
		// Nor those of users they blocked
		Where("NOT EXISTS (SELECT 1 FROM relations AS r WHERE r.user_id = ? AND r.target_id = m.user_id AND r.type = ?)",
			userID, models.RelationBlocked).
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

func (sc *ShoutboxController) historyEvent(ctx context.Context, c *gin.Context, user *AuthenticatedUser, ignored []chatIgnoredUser) ([]byte, error) {
	history, err := sc.history(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	conversations, err := sc.conversations(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return json.Marshal(shoutboxEvent{
		Type:          "history",
		Locale:        sc.Renderer.Localizer.GetLocale(c),
		Messages:      history,
		Conversations: conversations,
		Ignored:       ignored,
		Moderator:     user.IsModerator(),
	})
}

// The session's user as it is now, false once the session is invalid. Read again on every request, so renames
// and rank changes apply to open connections too
func (sc *ShoutboxController) sessionUser(ctx context.Context, token string) (*AuthenticatedUser, bool) {
	var user models.User
	err := sc.DB.WithContext(ctx).
		Model(&models.User{}).
		Select("users.id", "users.username", "users.rank").
		Joins("JOIN user_sessions ON user_sessions.user_id = users.id AND users.deleted_at IS NULL").
		Where("user_sessions.token = ? AND user_sessions.expires_at > ?", models.HashSessionToken(token), time.Now()).
		Take(&user).
		Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Shoutbox: failed to check session: %s", err)
		}
		return nil, false
	}
	return &AuthenticatedUser{ID: user.ID, Username: user.Username, Rank: user.Rank}, true
}

func writeShoutboxEvent(ctx context.Context, conn *websocket.Conn, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, shoutboxWriteTimeout)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, payload)
}

func queueShoutboxEvent(client *shoutboxClient, event shoutboxEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("Shoutbox: failed to encode event: %s", err)
		return
	}
	client.queue(payload)
}

// prepare returns the sanitized body, or "" after queueing the error for the client
func (sc *ShoutboxController) prepare(client *shoutboxClient, raw string) string {
	body, code := sanitizeShoutboxMessage(raw)
	if code != "" {
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: code})
		return ""
	}
	if ok, wait := sc.hub.allow(client.userID); !ok {
		queueShoutboxEvent(client, shoutboxEvent{
			Type:       "error",
			Error:      shoutboxErrorRateLimited,
			RetryAfter: (wait + time.Millisecond - 1).Milliseconds(),
		})
		return ""
	}
	return body
}

func (sc *ShoutboxController) mutedFor(ctx context.Context, userID uint) (time.Duration, error) {
	var mute models.ShoutboxMute
	err := sc.DB.WithContext(ctx).Where("user_id = ? AND until > ?", userID, time.Now()).Take(&mute).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return time.Until(mute.Until), nil
}

func (sc *ShoutboxController) post(ctx context.Context, client *shoutboxClient, username string, body string) {
	sc.postMu.Lock()
	defer sc.postMu.Unlock()

	msg := models.ShoutboxMessage{UserID: client.userID, Body: body}
	if err := sc.DB.WithContext(ctx).Create(&msg).Error; err != nil {
		log.Printf("Shoutbox: failed to save message: %s", err)
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		return
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
		return
	}
	sc.hub.broadcast(client.userID, payload)
	// Queued after the broadcast, so the sender has its message when the ack arrives
	queueShoutboxEvent(client, shoutboxEvent{Type: "ack"})
}

// handle returns false if the client must be disconnected
func (sc *ShoutboxController) handle(ctx context.Context, client *shoutboxClient, token string, req shoutboxRequest) bool {
	// Logging out doesn't close other tabs' connections
	user, ok := sc.sessionUser(ctx, token)
	if !ok {
		return false
	}
	switch req.Type {
	case "send":
		body := sc.prepare(client, *req.Body)
		if body == "" {
			return true
		}
		if req.To != nil {
			sc.sendPrivate(ctx, client, user.Username, *req.To, body)
			return true
		}
		muted, err := sc.mutedFor(ctx, user.ID)
		if err != nil {
			log.Printf("Shoutbox: failed to check mutes: %s", err)
			queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		} else if muted > 0 {
			queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorMuted, RetryAfter: muted.Milliseconds()})
		} else {
			sc.post(ctx, client, user.Username, body)
		}
	case "view":
		sc.view(ctx, client, *req.With)
	case "ignore":
		sc.ignore(ctx, client, *req.Username, chatIgnoreDurations[*req.Duration])
	case "unignore":
		sc.ignore(ctx, client, *req.Username, 0)
	case "delete", "mute", "unmute":
		if !user.IsModerator() {
			queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorForbidden})
			return true
		}
		switch req.Type {
		case "delete":
			sc.deleteMessage(ctx, client, *req.ID)
		case "mute":
			sc.mute(ctx, client, user, *req.Username, chatIgnoreDurations[*req.Duration])
		case "unmute":
			sc.mute(ctx, client, user, *req.Username, 0)
		}
	}
	return true
}

func (sc *ShoutboxController) deleteMessage(ctx context.Context, client *shoutboxClient, id uint) {
	// Under the posting lock, so the deletion is never broadcast before the message
	sc.postMu.Lock()
	defer sc.postMu.Unlock()
	result := sc.DB.WithContext(ctx).Delete(&models.ShoutboxMessage{}, id)
	if result.Error != nil {
		log.Printf("Shoutbox: failed to delete message %d: %s", id, result.Error)
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		return
	}
	if result.RowsAffected == 0 {
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorMessageNotFound})
		return
	}
	log.Printf("INFO Shoutbox: moderator %d deleted message %d", client.userID, id)
	if payload, err := json.Marshal(shoutboxEvent{Type: "deleted", ID: id}); err == nil {
		sc.hub.broadcastAll(payload)
	}
	queueShoutboxEvent(client, shoutboxEvent{Type: "ack"})
}

// mute lets the user post again when duration is 0. Moderators cannot mute their peers or admins
func (sc *ShoutboxController) mute(ctx context.Context, client *shoutboxClient, moderator *AuthenticatedUser, username string, duration time.Duration) {
	target, ok := sc.findPartner(ctx, client, username)
	if !ok {
		return
	}
	if target.Rank >= moderator.Rank {
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorForbidden})
		return
	}
	db := sc.DB.WithContext(ctx)
	var err error
	if duration > 0 {
		err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&models.ShoutboxMute{
			UserID:      target.ID,
			ModeratorID: moderator.ID,
			Until:       time.Now().Add(duration),
		}).Error
	} else {
		err = db.Where("user_id = ?", target.ID).Delete(&models.ShoutboxMute{}).Error
	}
	if err != nil {
		log.Printf("Shoutbox: failed to save mute: %s", err)
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		return
	}
	log.Printf("INFO Shoutbox: moderator %d muted user %d for %s", moderator.ID, target.ID, duration)
	queueShoutboxEvent(client, shoutboxEvent{Type: "ack"})
}

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
			if _, ok := sc.sessionUser(ctx, token); !ok {
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

// /api/shoutbox GET: the shoutbox WebSocket, see shoutboxEvent and shoutboxRequest for the protocol
func (sc *ShoutboxController) ApiShoutboxGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "You must be authenticated"})
		return
	}
	token, _ := c.Cookie("session_token")

	// The origin must match the host, so other sites can't connect with the user's cookie. Ports are not
	// compared: the gateway forwards the host without the port
	host := (&url.URL{Host: c.Request.Host}).Hostname()
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		OriginPatterns: []string{host + ":*"},
	})
	if err != nil {
		log.Printf("Shoutbox: handshake rejected: %s", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(shoutboxReadLimit)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	client := newShoutboxClient(user.ID)
	registered, ignored, err := sc.registerClient(ctx, client)
	if err != nil {
		log.Printf("Shoutbox: failed to load ignored users: %s", err)
		_ = conn.Close(websocket.StatusInternalError, "internal error")
		return
	}
	if !registered {
		_ = conn.Close(shoutboxCloseTooManyConnections, "too many connections")
		return
	}
	defer sc.hub.unregister(client)

	// Registered first so no message is missed, clients ignore the ones they already have
	payload, err := sc.historyEvent(ctx, c, user, ignored)
	if err != nil {
		log.Printf("Shoutbox: failed to load history: %s", err)
		_ = conn.Close(websocket.StatusInternalError, "internal error")
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
		req, ok := decodeShoutboxRequest(data)
		if !ok {
			_ = conn.Close(websocket.StatusPolicyViolation, "malformed request")
			return
		}
		if !sc.handle(ctx, client, token, req) {
			_ = conn.Close(shoutboxCloseUnauthorized, "unauthorized")
			return
		}
	}
}
