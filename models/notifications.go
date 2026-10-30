package models

import (
	"time"

	"gorm.io/datatypes"
)

// Notification types, each one has `notification-<type>-title` and `notification-<type>-desc` translations
const (
	NotificationFriendRequest = "friend-request"
	NotificationInfo          = "info"
	// Unread private messages from one user, a single notification is kept per sender
	NotificationPrivateMessage = "private-message"
)

type Notification struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"not null;index:idx_notification_feed,priority:1"`
	User   User   `gorm:"constraint:OnDelete:CASCADE"`
	Type   string `gorm:"not null"`
	// Translation arguments of the notification
	Data datatypes.JSONMap
	// Page opened when clicking the notification
	Action string
	// Unread while nil
	ReadAt    *time.Time `gorm:"index:idx_notification_feed,priority:2"`
	CreatedAt time.Time  `gorm:"index:idx_notification_feed,priority:3"`
}
