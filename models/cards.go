package models

import (
	"time"
)

type Card struct {
	ID uint `gorm:"primaryKey"`

	CardID   uint      `gorm:"not null"`
	Acquired time.Time `gorm:"not null"`
	// Foil pattern the card was obtained with, empty when not foil
	Foil string `gorm:"size:32;not null;default:''"`

	OwnerID uint `gorm:"not null;index"`
	Owner   User `gorm:"constraint:OnDelete:CASCADE"`
}

// Unopened booster of a collection
type Booster struct {
	ID uint `gorm:"primaryKey"`

	CollectionID uint `gorm:"not null"`
	// Number of cards, copied from the collection when acquired
	Size     uint      `gorm:"not null"`
	Acquired time.Time `gorm:"not null"`

	OwnerID uint `gorm:"not null;index"`
	Owner   User `gorm:"constraint:OnDelete:CASCADE"`
}
