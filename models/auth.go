package models

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

// User ranks, each one has every permission of the ranks below it
const (
	RankUser uint = 0
	// Moderates the chat
	RankModerator uint = 1
	// Manages accounts and roles
	RankAdmin uint = 2
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
	Coins        uint           `gorm:"not null;default:0"`
	// Last authenticated request, refreshed at most every LastSeenRefresh
	LastSeenAt *time.Time `gorm:"index"`
}

const (
	// Users seen within this window are online
	OnlineWindow = 5 * time.Minute
	// Minimum delay between two last seen updates of a user
	LastSeenRefresh = time.Minute
)

func IsOnline(lastSeen *time.Time) bool {
	return lastSeen != nil && time.Since(*lastSeen) < OnlineWindow
}

type UserSession struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint `gorm:"not null;index"`
	User   User `gorm:"constraint:OnDelete:CASCADE"`

	// SHA-256 of the cookie's token, see HashSessionToken: a leaked table holds no usable session
	Token     string    `gorm:"size:64;not null;uniqueIndex"`
	CreatedAt time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

// HashSessionToken is the form a session cookie's token is stored and looked up in
func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type UserProfile struct {
	UserID uint `gorm:"primaryKey"`
	User   User `gorm:"constraint:OnDelete:CASCADE"`

	Bio         string
	BirthDate   time.Time
	ContactInfo string
}
