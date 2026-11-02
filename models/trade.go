package models

import (
	"time"

	"gorm.io/gorm"
)

// Trade states of the offer lifecycle
const (
	TradePending   = "pending"
	TradeAccepted  = "accepted"
	TradeDeclined  = "declined"
	TradeCancelled = "cancelled"
)

// Sides of an offer: what the sender gives, and what they want from the receiver
const (
	TradeSideGive = "give"
	TradeSideWant = "want"
)

// TradeItem is one stack of identical cards in an offer, as clients send and receive it
type TradeItem struct {
	CardID uint   `json:"card_id"`
	Foil   string `json:"foil"`
	Count  uint   `json:"count"`
}

// Trade is an offer of the sender's cards and coins against the receiver's
type Trade struct {
	ID uint `gorm:"primaryKey"`

	// User making the offer, owner of the offered cards and coins
	FromID uint `gorm:"not null;index;index:idx_trades_from_state,priority:1"`
	From   User `gorm:"constraint:OnDelete:CASCADE"`
	// User the offer is made to, owner of the requested cards
	ToID uint `gorm:"not null;index;index:idx_trades_to_state,priority:1"`
	To   User `gorm:"constraint:OnDelete:CASCADE"`

	// The cards of both sides are its TradeLines
	// Coins the sender offers
	Coins uint `gorm:"not null;default:0"`
	// Coins requested from the receiver
	WantCoins uint `gorm:"not null;default:0"`

	// One of the trade states, e.g pending
	State string `gorm:"size:16;not null;index;index:idx_trades_from_state,priority:2;index:idx_trades_to_state,priority:2"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TradeLine is a stack of cards on one side of an offer
type TradeLine struct {
	ID uint `gorm:"primaryKey"`

	TradeID uint  `gorm:"not null;uniqueIndex:idx_trade_items_stack,priority:1"`
	Trade   Trade `gorm:"constraint:OnDelete:CASCADE"`
	// TradeSideGive or TradeSideWant
	Side   string `gorm:"size:4;not null;uniqueIndex:idx_trade_items_stack,priority:2"`
	CardID uint   `gorm:"not null;uniqueIndex:idx_trade_items_stack,priority:3"`
	Foil   string `gorm:"type:varchar(32) COLLATE utf8mb4_bin;not null;default:'';uniqueIndex:idx_trade_items_stack,priority:4"`
	Count  uint   `gorm:"not null;check:chk_trade_items_count,count > 0"`
}

func (TradeLine) TableName() string {
	return "trade_items"
}

type TradeItems struct {
	Give []TradeItem
	Want []TradeItem
}

// LoadTradeItems gets the stacks of several offers at once, by offer ID. Offers without cards on a side have no
// stack there.
func LoadTradeItems(db *gorm.DB, tradeIDs ...uint) (map[uint]*TradeItems, error) {
	items := make(map[uint]*TradeItems, len(tradeIDs))
	for _, id := range tradeIDs {
		items[id] = &TradeItems{}
	}
	if len(tradeIDs) == 0 {
		return items, nil
	}
	var lines []TradeLine
	if err := db.Where("trade_id IN ?", tradeIDs).Order("id").Find(&lines).Error; err != nil {
		return nil, err
	}
	for _, line := range lines {
		item := TradeItem{CardID: line.CardID, Foil: line.Foil, Count: line.Count}
		if line.Side == TradeSideGive {
			items[line.TradeID].Give = append(items[line.TradeID].Give, item)
		} else {
			items[line.TradeID].Want = append(items[line.TradeID].Want, item)
		}
	}
	return items, nil
}

func (t *Trade) Items(db *gorm.DB) (*TradeItems, error) {
	items, err := LoadTradeItems(db, t.ID)
	if err != nil {
		return nil, err
	}
	return items[t.ID], nil
}

func SaveTradeItems(tx *gorm.DB, tradeID uint, give []TradeItem, want []TradeItem) error {
	if err := tx.Where("trade_id = ?", tradeID).Delete(&TradeLine{}).Error; err != nil {
		return err
	}
	lines := make([]TradeLine, 0, len(give)+len(want))
	for side, items := range map[string][]TradeItem{TradeSideGive: give, TradeSideWant: want} {
		for _, item := range items {
			lines = append(lines, TradeLine{TradeID: tradeID, Side: side, CardID: item.CardID, Foil: item.Foil, Count: item.Count})
		}
	}
	if len(lines) == 0 {
		return nil
	}
	return tx.Create(&lines).Error
}
