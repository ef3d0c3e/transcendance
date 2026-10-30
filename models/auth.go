package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID uint `gorm:"primaryKey"`

	Email        string         `gorm:"size:254;not null;uniqueIndex" json:"-"`
	Username     string         `gorm:"size:16;uniqueIndex;not null"`
	PasswordHash string         `gorm:"not null"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Avatar       []byte         `gorm:"type:mediumblob" json:"-"`
	Rank         uint           `gorm:"not null;default:0"`
	// Coins is the in-shop currency balance
	Coins uint `gorm:"not null;default:0"`
	// Last authenticated request, refreshed at most every LastSeenRefresh
	LastSeenAt *time.Time `gorm:"index"`
}

const (
	// Users seen within this window are online
	OnlineWindow = 5 * time.Minute
	// Minimum delay between two last seen updates of a user
	LastSeenRefresh = time.Minute
)

// Whether a user last seen at the given time is online
func IsOnline(lastSeen *time.Time) bool {
	return lastSeen != nil && time.Since(*lastSeen) < OnlineWindow
}

type UserSession struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint `gorm:"not null;index"`
	User   User `gorm:"constraint:OnDelete:CASCADE"`

	Token     string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
}

type UserProfile struct {
	UserID uint `gorm:"primaryKey"`
	User   User `gorm:"constraint:OnDelete:CASCADE"`

	Bio         string
	BirthDate   time.Time
	ContactInfo string
}
