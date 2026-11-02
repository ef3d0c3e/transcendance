package models

import "time"

// A message posted to the global shoutbox
type ShoutboxMessage struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint `gorm:"not null;index"`
	User   User `gorm:"constraint:OnDelete:CASCADE"`
	// Sanitized, at most `controllers.shoutboxMaxLength` characters
	Body      string    `gorm:"size:500;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

// A private message between two users
type PrivateMessage struct {
	ID          uint `gorm:"primaryKey"`
	SenderID    uint `gorm:"not null;index:idx_private_sender,priority:1"`
	Sender      User `gorm:"constraint:OnDelete:CASCADE"`
	RecipientID uint `gorm:"not null;index:idx_private_recipient,priority:1"`
	Recipient   User `gorm:"constraint:OnDelete:CASCADE"`
	// Sanitized like shoutbox messages
	Body      string    `gorm:"size:500;not null"`
	CreatedAt time.Time `gorm:"not null"`
	// Unread by the recipient while nil
	ReadAt *time.Time `gorm:"index:idx_private_recipient,priority:2"`
}

// A user ignored by another until a given time: their shoutbox messages are hidden and their private messages
// rejected
type UserIgnore struct {
	UserID    uint      `gorm:"primaryKey"`
	User      User      `gorm:"constraint:OnDelete:CASCADE"`
	IgnoredID uint      `gorm:"primaryKey"`
	Ignored   User      `gorm:"constraint:OnDelete:CASCADE"`
	Until     time.Time `gorm:"not null"`
}
