package controllers

import (
	"context"
	"encoding/json"
	"log"
	"time"
	"transcendance/models"

	"gorm.io/gorm/clause"
)

// How long a user can be ignored for, by `ignore` request duration
var chatIgnoreDurations = map[string]time.Duration{
	"hour": time.Hour,
	"day":  24 * time.Hour,
	"week": 7 * 24 * time.Hour,
}

type chatIgnoredUser struct {
	Username string `json:"username"`
	Until    string `json:"until"`
}

// Ignores the hub keeps for users the user blocked from their profile, which have no end
var blockedUntil = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

// The users a user ignores, by ID for the hub and as sent to clients. The hub also hides the messages of the users
// they blocked. Expired ignores are cleared.
func (sc *ShoutboxController) loadIgnores(ctx context.Context, userID uint) (map[uint]time.Time, []chatIgnoredUser, error) {
	now := time.Now()
	db := sc.DB.WithContext(ctx)
	if err := db.Where("user_id = ? AND until <= ?", userID, now).Delete(&models.UserIgnore{}).Error; err != nil {
		return nil, nil, err
	}

	var rows []struct {
		IgnoredID uint
		Username  string
		Until     time.Time
	}
	if err := db.
		Table("user_ignores AS i").
		Select("i.ignored_id, u.username, i.until").
		Joins("JOIN users AS u ON u.id = i.ignored_id AND u.deleted_at IS NULL").
		Where("i.user_id = ?", userID).
		Order("u.username").
		Scan(&rows).
		Error; err != nil {
		return nil, nil, err
	}

	var blocked []uint
	if err := db.Model(&models.Relation{}).Where("user_id = ? AND type = ?", userID, models.RelationBlocked).
		Pluck("target_id", &blocked).Error; err != nil {
		return nil, nil, err
	}

	ids := make(map[uint]time.Time, len(rows)+len(blocked))
	for _, id := range blocked {
		ids[id] = blockedUntil
	}
	users := make([]chatIgnoredUser, len(rows))
	for i, row := range rows {
		if _, ok := ids[row.IgnoredID]; !ok {
			ids[row.IgnoredID] = row.Until
		}
		users[i] = chatIgnoredUser{Username: row.Username, Until: row.Until.Format(time.RFC3339)}
	}
	return ids, users, nil
}

func (sc *ShoutboxController) registerClient(ctx context.Context, client *shoutboxClient) (bool, []chatIgnoredUser, error) {
	sc.ignoreMu.Lock()
	defer sc.ignoreMu.Unlock()

	ids, users, err := sc.loadIgnores(ctx, client.userID)
	if err != nil {
		return false, nil, err
	}
	return sc.hub.register(client, ids), users, nil
}

func (sc *ShoutboxController) ignores(ctx context.Context, userID uint, otherID uint) (bool, error) {
	var count int64
	err := sc.DB.WithContext(ctx).
		Model(&models.UserIgnore{}).
		Where("user_id = ? AND ignored_id = ? AND until > ?", userID, otherID, time.Now()).
		Count(&count).
		Error
	return count > 0, err
}

// ignore stops ignoring when duration is 0, and sends the new list to every tab
func (sc *ShoutboxController) ignore(ctx context.Context, client *shoutboxClient, username string, duration time.Duration) {
	other, ok := sc.findPartner(ctx, client, username)
	if !ok {
		return
	}

	sc.ignoreMu.Lock()
	defer sc.ignoreMu.Unlock()

	db := sc.DB.WithContext(ctx)
	var err error
	if duration > 0 {
		err = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&models.UserIgnore{
			UserID:    client.userID,
			IgnoredID: other.ID,
			Until:     time.Now().Add(duration),
		}).Error
	} else {
		err = db.Where("user_id = ? AND ignored_id = ?", client.userID, other.ID).Delete(&models.UserIgnore{}).Error
	}
	if err != nil {
		log.Printf("Shoutbox: failed to save ignore: %s", err)
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		return
	}

	ids, users, err := sc.loadIgnores(ctx, client.userID)
	if err != nil {
		log.Printf("Shoutbox: failed to load ignored users: %s", err)
		queueShoutboxEvent(client, shoutboxEvent{Type: "error", Error: shoutboxErrorInternal})
		return
	}
	sc.hub.setIgnores(client.userID, ids)
	payload, err := json.Marshal(shoutboxIgnoredEvent{Type: "ignored", Ignored: users})
	if err != nil {
		log.Printf("Shoutbox: failed to encode ignored users: %s", err)
		return
	}
	sc.hub.deliver(payload, client.userID)
	queueShoutboxEvent(client, shoutboxEvent{Type: "ack"})
}

// The `ignored` event, with the list even when empty
type shoutboxIgnoredEvent struct {
	Type    string            `json:"type"`
	Ignored []chatIgnoredUser `json:"ignored"`
}
