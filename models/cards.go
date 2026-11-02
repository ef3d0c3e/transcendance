package models

import (
	"slices"
	"time"
)

// Foil patterns a card can be obtained with, each one has a mask in `data/assets`. A card without foil has "".
var FoilPatterns = []string{"foil1", "foil2", "foil3", "foil4", "foil5", "foil6"}

// ValidFoil reports whether a foil value from a request is exactly "" or one of the patterns
func ValidFoil(foil string) bool {
	return foil == "" || slices.Contains(FoilPatterns, foil)
}

// Column type of foil values: compared byte for byte like in Go, where the default collation would match
// "foil1" with "FOIL1 "
const FoilColumnType = "varchar(32) COLLATE utf8mb4_bin"

type Card struct {
	ID uint `gorm:"primaryKey"`

	CardID   uint      `gorm:"not null;index:idx_cards_stack,priority:2"`
	Acquired time.Time `gorm:"not null"`
	// Foil pattern the card was obtained with, empty when not foil
	Foil string `gorm:"type:varchar(32) COLLATE utf8mb4_bin;not null;default:'';index:idx_cards_stack,priority:3"`

	// The stack index serves the queries of an inventory, by card and foil
	OwnerID uint `gorm:"not null;index;index:idx_cards_stack,priority:1"`
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

// A user's pile of default boosters, which refills over time up to a cap
type BoosterPile struct {
	OwnerID uint `gorm:"primaryKey"`
	Owner   User `gorm:"constraint:OnDelete:CASCADE"`

	// Boosters currently in the pile
	Count uint `gorm:"not null;default:0"`
	// Time at which the next booster joins the pile, nil while the pile is full
	NextAt *time.Time
}
