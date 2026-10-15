package models

import (
	"time"
)

type Card struct {
	ID uint `gorm:"primaryKey"`

	CardID uint `gorm:"not null"`
	Acquired time.Time `gorm:"not null"`

	OwnerID  uint      `gorm:"not null;index"`
	Owner    User      `gorm:"constraint:OnDelete:CASCADE"`
}
