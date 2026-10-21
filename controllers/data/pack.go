package data

import (
	"log"
	"math/rand/v2"
	"net/http"
	"sort"
	"strconv"
	"time"

	"transcendance/controllers"
	"transcendance/data"
	"transcendance/views"

	"github.com/gin-gonic/gin"
)

// Number of cards in a booster pack. Each one gets its own opening reel.
const PackSize = 5

// Reel stack around the won card. The reel drops, so the cards stacked
// beneath the winner run past the window first — reelBelow is the spin
// length, in cards. The few above only give the overshoot something to
// settle against.
const (
	reelAbove = 2
	reelBelow = 12
)

// Rarity tiers ordered from highest to lowest. Used to fall back to a lower
// tier when the rolled rarity has no card in the opened collection.
var rarityTiers = []string{"legendary", "rare", "common"}

// Drop weights out of 100 for the tiers above: common 80%, rare 15%,
// legendary 5%
const (
	weightCommon    = 80
	weightRare      = 15
	weightLegendary = 5
)

// rollRarity picks a rarity tier from the drop table
func rollRarity(rnd *rand.Rand) string {
	n := rnd.N(weightCommon + weightRare + weightLegendary)
	switch {
	case n < weightCommon:
		return "common"
	case n < weightCommon+weightRare:
		return "rare"
	default:
		return "legendary"
	}
}

// indexOfTier returns the position of a rarity in rarityTiers, or -1
func indexOfTier(rarity string) int {
	for i, tier := range rarityTiers {
		if tier == rarity {
			return i
		}
	}
	return -1
}

// pickTier returns the rarity to draw from. When the rolled tier holds no
// card, it falls back to the next lower tier (legendary -> rare -> common).
// Returns "" when no tier has any card.
func pickTier(rolled string, available map[string]bool) string {
	start := indexOfTier(rolled)
	if start < 0 {
		start = 0
	}
	for i := start; i < len(rarityTiers); i++ {
		if available[rarityTiers[i]] {
			return rarityTiers[i]
		}
	}
	// Collection only holds higher tiers than the one rolled: use those
	for i := start - 1; i >= 0; i-- {
		if available[rarityTiers[i]] {
			return rarityTiers[i]
		}
	}
	return ""
}

// groupByRarity indexes the cards of a collection by rarity
func groupByRarity(cards map[string]*data.Card, collection *data.Collection) map[string][]*data.Card {
	byRarity := make(map[string][]*data.Card, len(rarityTiers))
	for _, entry := range collection.Cards {
		card, ok := cards[entry.Name]
		if !ok {
			continue
		}
		byRarity[card.Rarity] = append(byRarity[card.Rarity], card)
	}
	return byRarity
}

// stripEntry is one slot on an opening reel
type stripEntry struct {
	Card *data.Card
	Win  bool
}

// flattenByRarity returns every card of the collection in one slice
func flattenByRarity(byRarity map[string][]*data.Card) []*data.Card {
	pool := make([]*data.Card, 0)
	for _, tier := range rarityTiers {
		pool = append(pool, byRarity[tier]...)
	}
	return pool
}

// buildReel returns the stack for one won card: cards above the winner at
// index reelAbove, the winner, then the cards that will run past the window
// as the reel drops. Returns nil when the collection holds no card to fill
// with.
func buildReel(rnd *rand.Rand, pool []*data.Card, winner *data.Card) []stripEntry {
	if len(pool) == 0 {
		return nil
	}

	strip := make([]stripEntry, 0, reelAbove+1+reelBelow)
	for i := 0; i < reelAbove; i++ {
		strip = append(strip, stripEntry{Card: pool[rnd.N(len(pool))], Win: false})
	}
	strip = append(strip, stripEntry{Card: winner, Win: true})
	for i := 0; i < reelBelow; i++ {
		strip = append(strip, stripEntry{Card: pool[rnd.N(len(pool))], Win: false})
	}

	return strip
}

// drawPack draws PackSize cards from a collection. The rarity of each card is
// rolled first, then a card is picked uniformly among that tier's cards.
// Duplicates are allowed. Returns nil when the collection holds no card.
func drawPack(rnd *rand.Rand, byRarity map[string][]*data.Card) []*data.Card {
	available := make(map[string]bool, len(rarityTiers))
	for _, tier := range rarityTiers {
		available[tier] = len(byRarity[tier]) > 0
	}

	pack := make([]*data.Card, 0, PackSize)
	for i := 0; i < PackSize; i++ {
		tier := pickTier(rollRarity(rnd), available)
		if tier == "" {
			return nil
		}
		pack = append(pack, byRarity[tier][rnd.N(len(byRarity[tier]))])
	}
	return pack
}

// /opening GET: booster picker
func (dc *DataController) OpeningGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	loc := dc.Renderer.Localizer.GetLocale(c)

	sorted := make([]*data.Collection, 0, len(dc.Data.Collections))
	for _, collection := range dc.Data.Collections {
		sorted = append(sorted, collection)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	collections := make([]map[string]any, 0, len(sorted))
	for _, collection := range sorted {
		title, err := collection.Translate(loc, "title")
		if err != nil {
			log.Printf("[WARN] Failed to translate collection '%s': %s", collection.Name, err)
			continue
		}
		collections = append(collections, map[string]any{
			"ID":    collection.ID,
			"Title": title,
		})
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": dc.Renderer.T(c, "pack-opening-title"),
		"User":  user,
	})
	builder.Add("pack-opening", "Content", map[string]any{
		"Collections": collections,
	})
	dc.Renderer.Render(c, &builder)
}

// packCard renders one drawn card as an HTML fragment, or writes the failure
func packCard(dc *DataController, c *gin.Context, collection *data.Collection, card *data.Card) (string, bool) {
	loc := dc.Renderer.Localizer.GetLocale(c)

	cardData, err := GetCardData(c, loc, dc.Renderer, collection, card)
	if err != nil {
		log.Printf("[WARN] Failed to build data for card '%s': %s", card.Name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-internal"),
		})
		return "", false
	}
	cardData["CardRarity"] = card.Rarity

	html, err := dc.Renderer.RenderToString("pack-card", map[string]any{"Data": cardData})
	if err != nil {
		log.Printf("[WARN] Failed to render pack card '%s': %s", card.Name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-internal"),
		})
		return "", false
	}
	return html, true
}

// /opening POST: draw a pack of cards from a collection
func (dc *DataController) OpeningPost(c *gin.Context) {
	id, err := strconv.Atoi(c.PostForm("collection"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-collection"),
		})
		return
	}

	collection := dc.Data.GetCollection(id)
	if collection == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-collection"),
		})
		return
	}

	rnd := rand.New(rand.NewPCG(
		uint64(time.Now().UnixNano()),
		uint64(time.Now().UnixNano()),
	))
	byRarity := groupByRarity(dc.Data.Cards, collection)
	pack := drawPack(rnd, byRarity)
	if pack == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-empty"),
		})
		return
	}

	pool := flattenByRarity(byRarity)
	reels := make([][]gin.H, 0, PackSize)
	for _, winner := range pack {
		strip := buildReel(rnd, pool, winner)
		if strip == nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": dc.Renderer.T(c, "pack-opening-error-empty"),
			})
			return
		}

		slots := make([]gin.H, 0, len(strip))
		for _, entry := range strip {
			html, ok := packCard(dc, c, collection, entry.Card)
			if !ok {
				return
			}
			slots = append(slots, gin.H{
				"rarity": entry.Card.Rarity,
				"html":   html,
				"win":    entry.Win,
			})
		}
		reels = append(reels, slots)
	}

	c.JSON(http.StatusOK, gin.H{
		"reels": reels,
		"colors": gin.H{
			"primary":   collection.ColorPrimary,
			"secondary": collection.ColorSecondary,
		},
	})
}
