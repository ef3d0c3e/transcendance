package models

import (
	"encoding/json"
	"time"

	"gorm.io/datatypes"
)

// Trade states of the offer lifecycle
const (
	TradePending   = "pending"
	TradeAccepted  = "accepted"
	TradeDeclined  = "declined"
	TradeCancelled = "cancelled"
)

// TradeItem is one stack of identical cards in an offer
type TradeItem struct {
	CardID uint   `json:"card_id"`
	Foil   string `json:"foil"`
	Count  uint   `json:"count"`
}

// Trade is an offer between two users: the sender offers their cards and
// coins against cards owned by the receiver. Countering an offer replaces
// its contents and flips the two sides.
type Trade struct {
	ID uint `gorm:"primaryKey"`

	// User making the offer, owner of the offered cards and coins
	FromID uint `gorm:"not null;index"`
	From   User `gorm:"constraint:OnDelete:CASCADE"`
	// User the offer is made to, owner of the requested cards
	ToID uint `gorm:"not null;index"`
	To   User `gorm:"constraint:OnDelete:CASCADE"`

	// Cards the sender offers, JSON list of TradeItem
	Give datatypes.JSON `gorm:"not null"`
	// Coins the sender offers
	Coins uint `gorm:"not null;default:0"`
	// Cards requested from the receiver, JSON list of TradeItem
	Want datatypes.JSON `gorm:"not null"`

	// One of the trade states, e.g pending
	State string `gorm:"not null;index"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// GiveItems of the offer
func (t *Trade) GiveItems() ([]TradeItem, error) {
	var items []TradeItem
	err := json.Unmarshal(t.Give, &items)
	return items, err
}

// WantItems of the offer
func (t *Trade) WantItems() ([]TradeItem, error) {
	var items []TradeItem
	err := json.Unmarshal(t.Want, &items)
	return items, err
}
