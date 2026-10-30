package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/url"
	"time"
	"transcendance/models"
	"unicode/utf8"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	// Conversations sent to clients when they connect
	chatConversationsSize = 50
	// Messages of a conversation sent when it is viewed
	chatConversationSize = 100
	// Characters of the latest message shown in notifications
	chatPreviewLength = 80
)

// Private message as sent to clients
type chatPrivateMessage struct {
	ID        uint   `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

// Conversation with another user as sent to clients, with the latest message and the number of unread ones
type chatConversation struct {
	Username string             `json:"username"`
	Last     chatPrivateMessage `json:"last"`
	Unread   int                `json:"unread"`
}

// Private messages with their sender and recipient usernames, to be filtered and ordered by the caller
func (sc *ShoutboxController) privateQuery(ctx context.Context) *gorm.DB {
	return sc.DB.WithContext(ctx).
		Table("private_messages AS m").
		Select("m.id, s.username AS sender, r.username AS recipient, m.body, m.created_at").
		Joins("JOIN users AS s ON s.id = m.sender_id AND s.deleted_at IS NULL").
		Joins("JOIN users AS r ON r.id = m.recipient_id AND r.deleted_at IS NULL")
}

func scanPrivate(query *gorm.DB) ([]chatPrivateMessage, error) {
	var rows []struct {
		ID        uint
		Sender    string
		Recipient string
		Body      string
		CreatedAt time.Time
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	messages := make([]chatPrivateMessage, len(rows))
	for i, row := range rows {
		messages[i] = chatPrivateMessage{
			ID:        row.ID,
			From:      row.Sender,
			To:        row.Recipient,
			Body:      row.Body,
			CreatedAt: row.CreatedAt.Format(time.RFC3339),
		}
	}
	return messages, nil
}

// The user's latest conversations, latest first
func (sc *ShoutboxController) conversations(ctx context.Context, userID uint) ([]chatConversation, error) {
	var summaries []struct {
		PartnerID uint
		LastID    uint
		Unread    int
	}
	if err := sc.DB.WithContext(ctx).Raw(`
		SELECT t.partner_id, MAX(t.id) AS last_id,
			SUM(CASE WHEN t.recipient_id = ? AND t.read_at IS NULL THEN 1 ELSE 0 END) AS unread
		FROM (
			SELECT id, recipient_id, read_at,
				CASE WHEN sender_id = ? THEN recipient_id ELSE sender_id END AS partner_id
			FROM private_messages
			WHERE sender_id = ? OR recipient_id = ?
		) AS t
		GROUP BY t.partner_id
		ORDER BY last_id DESC
		LIMIT ?`, userID, userID, userID, userID, chatConversationsSize).
		Scan(&summaries).
		Error; err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, nil
	}

	ids := make([]uint, len(summaries))
	for i, summary := range summaries {
		ids[i] = summary.LastID
	}
	lasts, err := scanPrivate(sc.privateQuery(ctx).Where("m.id IN ?", ids))
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]chatPrivateMessage, len(lasts))
	for _, last := range lasts {
		byID[last.ID] = last
	}

	var user models.User
	if err := sc.DB.WithContext(ctx).Select("username").First(&user, userID).Error; err != nil {
		return nil, err
	}
	conversations := make([]chatConversation, 0, len(summaries))
	for _, summary := range summaries {
		// Missing when the other user was deleted
		last, ok := byID[summary.LastID]
		if !ok {
			continue
		}
		partner := last.From
		if partner == user.Username {
			partner = last.To
		}
		conversations = append(conversations, chatConversation{Username: partner, Last: last, Unread: summary.Unread})
	}
	return conversations, nil
}

// Find the other user of a conversation, queues the error for the client if there is none
func (sc *ShoutboxController) findPartner(ctx context.Context, client *shoutboxClient, username string) (*models.User, bool) {
	var user models.User
	err := sc.DB.WithContext(ctx).Where("username = ?", username).First(&user).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorUserNotFound})
		return nil, false
	case err != nil:
		log.Printf("Shoutbox: failed to find user: %s", err)
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		return nil, false
	case user.ID == client.userID:
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorSelf})
		return nil, false
	}
	return &user, true
}

// Save a private message and send it to both users' tabs. Users who ignore the sender don't receive it.
func (sc *ShoutboxController) sendPrivate(ctx context.Context, client *shoutboxClient, username string, to string, body string) {
	recipient, ok := sc.findPartner(ctx, client, to)
	if !ok {
		return
	}
	blocked, err := sc.ignores(ctx, recipient.ID, client.userID)
	if err != nil {
		log.Printf("Shoutbox: failed to check ignores: %s", err)
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		return
	}
	if blocked {
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorBlocked})
		return
	}

	sc.postMu.Lock()
	defer sc.postMu.Unlock()

	msg := models.PrivateMessage{SenderID: client.userID, RecipientID: recipient.ID, Body: body}
	// Already read when the recipient has the conversation open
	if sc.hub.viewing(recipient.ID, client.userID) {
		now := time.Now()
		msg.ReadAt = &now
	}
	if err := sc.DB.WithContext(ctx).Create(&msg).Error; err != nil {
		log.Printf("Shoutbox: failed to save private message: %s", err)
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		return
	}

	payload, err := json.Marshal(shoutboxEvent{
		Type: "private",
		Private: &chatPrivateMessage{
			ID:        msg.ID,
			From:      username,
			To:        recipient.Username,
			Body:      msg.Body,
			CreatedAt: msg.CreatedAt.Format(time.RFC3339),
		},
	})
	if err != nil {
		log.Printf("Shoutbox: failed to encode private message: %s", err)
		return
	}
	sc.hub.deliver(payload, client.userID, recipient.ID)
	if msg.ReadAt == nil {
		if err := sc.notifyPrivate(ctx, recipient.ID, client.userID, username, msg.Body); err != nil {
			log.Printf("Shoutbox: failed to notify private message: %s", err)
		}
	}
	queueShoutboxEvent(client, shoutboxEvent{Type: "ack"})
}

// Notify a user of unread private messages from a sender. There is one unread notification per sender, updated
// with the number of unread messages and the latest one, and moved back to the top.
func (sc *ShoutboxController) notifyPrivate(ctx context.Context, recipientID uint, senderID uint, username string, body string) error {
	db := sc.DB.WithContext(ctx)

	var unread int64
	if err := db.Model(&models.PrivateMessage{}).
		Where("recipient_id = ? AND sender_id = ? AND read_at IS NULL", recipientID, senderID).
		Count(&unread).
		Error; err != nil {
		return err
	}

	preview := body
	if utf8.RuneCountInString(preview) > chatPreviewLength {
		preview = string([]rune(preview)[:chatPreviewLength-1]) + "…"
	}
	data := datatypes.JSONMap{
		"sender":   senderID,
		"username": username,
		"count":    unread,
		"preview":  preview,
	}
	// The sender's profile, with the conversation open
	action := "/profile/" + url.PathEscape(username) + "?chat=" + url.QueryEscape(username)

	var notif models.Notification
	err := db.
		Where("user_id = ? AND type = ? AND read_at IS NULL", recipientID, models.NotificationPrivateMessage).
		Where("JSON_VALUE(data, '$.sender') = ?", senderID).
		Order("id DESC").
		First(&notif).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&models.Notification{
			UserID: recipientID,
			Type:   models.NotificationPrivateMessage,
			Data:   data,
			Action: action,
		}).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&notif).Updates(map[string]any{
		"data":       data,
		"action":     action,
		"created_at": time.Now(),
	}).Error
}

// Show the conversation with a user on a client, or none when empty. It is marked read on every tab, with its
// notification, and sent to the client.
func (sc *ShoutboxController) view(ctx context.Context, client *shoutboxClient, with string) {
	if with == "" {
		sc.hub.setViewing(client, 0)
		return
	}
	partner, ok := sc.findPartner(ctx, client, with)
	if !ok {
		sc.hub.setViewing(client, 0)
		return
	}
	// Before marking read, so messages sent meanwhile are read too
	sc.hub.setViewing(client, partner.ID)

	marked, err := sc.markRead(ctx, client.userID, partner.ID)
	if err != nil {
		log.Printf("Shoutbox: failed to mark conversation read: %s", err)
	}
	if marked {
		if payload, err := json.Marshal(shoutboxEvent{Type: "read", With: partner.Username}); err == nil {
			sc.hub.deliver(payload, client.userID)
		}
	}

	latest, err := scanPrivate(sc.privateQuery(ctx).
		Where("(m.sender_id = ? AND m.recipient_id = ?) OR (m.sender_id = ? AND m.recipient_id = ?)",
			client.userID, partner.ID, partner.ID, client.userID).
		Order("m.id DESC").
		Limit(chatConversationSize))
	if err != nil {
		log.Printf("Shoutbox: failed to load conversation: %s", err)
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		return
	}
	privates := make([]chatPrivateMessage, len(latest))
	for i := range latest {
		privates[i] = latest[len(latest)-1-i]
	}
	queueShoutboxEvent(client, shoutboxEvent{Type: "conversation", With: partner.Username, Privates: privates})
}

// Mark the messages from a partner and their notification read, returns whether there were unread ones
func (sc *ShoutboxController) markRead(ctx context.Context, userID uint, partnerID uint) (bool, error) {
	now := time.Now()
	db := sc.DB.WithContext(ctx)
	result := db.Model(&models.PrivateMessage{}).
		Where("recipient_id = ? AND sender_id = ? AND read_at IS NULL", userID, partnerID).
		Update("read_at", now)
	if result.Error != nil {
		return false, result.Error
	}
	if err := db.Model(&models.Notification{}).
		Where("user_id = ? AND type = ? AND read_at IS NULL", userID, models.NotificationPrivateMessage).
		Where("JSON_VALUE(data, '$.sender') = ?", partnerID).
		Update("read_at", now).
		Error; err != nil {
		return false, err
	}
	return result.RowsAffected > 0, nil
}
