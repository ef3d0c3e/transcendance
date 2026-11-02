package models

import (
	"time"
)

// A user's daily connection streak, which grows with every consecutive day
// they show up and resets when a day is missed
type DailyStreak struct {
	UserID uint `gorm:"primaryKey"`
	User   User `gorm:"constraint:OnDelete:CASCADE"`

	// Consecutive days the reward was claimed on in a row
	Day uint `gorm:"not null;default:0"`
	// Best streak the user has done
	Best uint `gorm:"not null;default:0"`
	// Start of the day the last reward was claimed on, nil until the first one
	LastDay *time.Time
}
