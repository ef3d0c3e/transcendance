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
