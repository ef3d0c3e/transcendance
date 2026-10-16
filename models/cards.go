package models

import (
	"time"
)

// CardEffect is a special effect attached to a single acquired card copy.
//
// `Name` is one of the storable effect kinds (see `validEffects` in
// `controllers/data/effects.go` and the `card-*` classes in
// `frontend/src/CardEffectStyles.svelte`). Glare is the standard treatment of
// every card and is not stored here.
// `Params` holds optional presentation overrides, keyed by the `data-*`
// attribute names understood by `frontend/src/CardRender.svelte`
// (e.g. `{"hue": "90deg"}`).
type CardEffect struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params,omitempty"`
}

// CardEffects is the list of special effects rolled for a card copy.
type CardEffects []CardEffect

// Card is a single acquired copy of a card in a user's inventory.
//
// Cards are identified by their **global ID**, never by the row ID:
//
//	global ID = collection_id * 1000 + card_id
//
// Both components are assigned in the card data files (`data/cards/*/card.toml`
// and `data/collections/*/collection.toml`) and resolved through `data.GetCard`.
// The auto-increment `ID` is only the inventory row key and must not be used
// as card identity, so that references stay stable across reloads.
type Card struct {
	ID uint `gorm:"primaryKey"`

	// Global card ID (`collection_id * 1000 + card_id`)
	CardID uint `gorm:"not null;index"`

	// When this copy was acquired
	Acquired time.Time `gorm:"not null"`

	// Special effects rolled for this copy; empty means a plain copy
	Effects CardEffects `gorm:"type:json;serializer:json"`

	OwnerID uint `gorm:"not null;index"`
	Owner   User `gorm:"constraint:OnDelete:CASCADE"`
}
