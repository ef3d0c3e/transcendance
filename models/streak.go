package models

import (
	"time"
)

// A user's daily connection streak, which grows with every consecutive day
// they show up and resets when a day is missed
type DailyStreak struct {
	UserID uint `gorm:"primaryKey"`
	User   User `gorm:"constraint:OnDelete:CASCADE"`

	// Day of the streak the last reward was credited on, capped
	Day uint `gorm:"not null;default:0"`
	// Start of the day the last reward was credited on, nil until the first one
	LastDay *time.Time
}
