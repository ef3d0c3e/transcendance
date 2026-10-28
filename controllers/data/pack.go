package data

import (
	"errors"
	"log"
	"math/rand/v2"
	"net/http"
	"sort"
	"strconv"
	"time"

	"transcendance/controllers"
	"transcendance/data"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Reel stack around the won card. The reel drops, so the cards stacked
// beneath the winner run past the window first — reelBelow is the spin
// length, in cards. The few above only give the overshoot something to
// settle against.
const (
	reelAbove = 2
	reelBelow = 24
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

// poolCard is one card of a draw pool, with the collection it belongs to
type poolCard struct {
	card       *data.Card
	collection *data.Collection
}

// groupByRarity indexes the cards of a collection by rarity
func groupByRarity(cards map[string]*data.Card, collection *data.Collection) map[string][]poolCard {
	byRarity := make(map[string][]poolCard, len(rarityTiers))
	for _, entry := range collection.Cards {
		card, ok := cards[entry.Name]
		if !ok {
			continue
		}
		byRarity[card.Rarity] = append(byRarity[card.Rarity], poolCard{card: card, collection: collection})
	}
	return byRarity
}

// stripEntry is one slot on an opening reel
type stripEntry struct {
	Card       *data.Card
	Collection *data.Collection
	Win        bool
}

// flattenByRarity returns every card of the collection in one slice
func flattenByRarity(byRarity map[string][]poolCard) []poolCard {
	pool := make([]poolCard, 0)
	for _, tier := range rarityTiers {
		pool = append(pool, byRarity[tier]...)
	}
	return pool
}

// buildReel returns the stack for one won card: cards above the winner at
// index reelAbove, the winner, then the cards that will run past the window
// as the reel drops. Returns nil when the pool holds no card to fill with.
func buildReel(rnd *rand.Rand, pool []poolCard, winner poolCard) []stripEntry {
	if len(pool) == 0 {
		return nil
	}

	strip := make([]stripEntry, 0, reelAbove+1+reelBelow)
	for i := 0; i < reelAbove; i++ {
		strip = append(strip, newStripEntry(pool[rnd.N(len(pool))], false))
	}
	strip = append(strip, newStripEntry(winner, true))
	for i := 0; i < reelBelow; i++ {
		strip = append(strip, newStripEntry(pool[rnd.N(len(pool))], false))
	}

	return strip
}

// newStripEntry places one pool card on an opening reel slot
func newStripEntry(pc poolCard, win bool) stripEntry {
	return stripEntry{Card: pc.card, Collection: pc.collection, Win: win}
}

// drawPack draws `size` cards from a collection. The rarity of each card is
// rolled first, then a card is picked uniformly among that tier's cards.
// Duplicates are allowed. Returns nil when the collection holds no card.
func drawPack(rnd *rand.Rand, byRarity map[string][]poolCard, size int) []poolCard {
	available := make(map[string]bool, len(rarityTiers))
	for _, tier := range rarityTiers {
		available[tier] = len(byRarity[tier]) > 0
	}

	pack := make([]poolCard, 0, size)
	for i := 0; i < size; i++ {
		tier := pickTier(rollRarity(rnd), available)
		if tier == "" {
			return nil
		}
		pack = append(pack, byRarity[tier][rnd.N(len(byRarity[tier]))])
	}
	return pack
}

// drawDefaultPack draws `size` cards among the pool, each of them uniformly
// across every collection. Duplicates are allowed. Returns nil when the pool
// holds no card.
func drawDefaultPack(rnd *rand.Rand, pool []poolCard, size int) []poolCard {
	if len(pool) == 0 {
		return nil
	}

	pack := make([]poolCard, 0, size)
	for i := 0; i < size; i++ {
		pack = append(pack, pool[rnd.N(len(pool))])
	}
	return pack
}

// Oldest booster of a stack (same collection and size) owned by a user, nil when none is left
func (dc *DataController) nextBooster(ownerID uint, collectionID uint, size uint) (*models.Booster, error) {
	var booster models.Booster
	err := dc.DB.
		Where("owner_id = ? AND collection_id = ? AND size = ?", ownerID, collectionID, size).
		Order("id").
		First(&booster).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &booster, nil
}

// allCardsPool returns every card of every collection in one pool, cards of
// the lower IDs first
func (dc *DataController) allCardsPool() []poolCard {
	sorted := make([]*data.Collection, 0, len(dc.Data.Collections))
	for _, collection := range dc.Data.Collections {
		sorted = append(sorted, collection)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	pool := make([]poolCard, 0)
	for _, collection := range sorted {
		pool = append(pool, flattenByRarity(groupByRarity(dc.Data.Cards, collection))...)
	}
	return pool
}

// packReels renders the opening reels of a drawn pack from a pool of
// possible cards, or writes the failure
func (dc *DataController) packReels(c *gin.Context, rnd *rand.Rand, pool []poolCard, pack []poolCard, foils []string) ([][]gin.H, bool) {
	reels := make([][]gin.H, 0, len(pack))
	for i, winner := range pack {
		strip := buildReel(rnd, pool, winner)

		slots := make([]gin.H, 0, len(strip))
		for _, entry := range strip {
			// Only the won card shows the foil it was obtained with
			foil := ""
			if entry.Win {
				foil = foils[i]
			}
			html, ok := packCard(dc, c, entry.Collection, entry.Card, foil)
			if !ok {
				return nil, false
			}
			slots = append(slots, gin.H{
				"rarity": entry.Card.Rarity,
				"html":   html,
				"win":    entry.Win,
			})
		}
		reels = append(reels, slots)
	}
	return reels, true
}

// /opening GET: open one of the user's boosters (?booster=ID)
func (dc *DataController) OpeningGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// A default booster comes out of the user's pile of default boosters
	if c.Query("booster") == defaultBoosterRef {
		pile, err := controllers.AccrueBoosterPile(dc.DB, user.ID)
		if err != nil {
			log.Printf("[WARN] Failed to get the booster pile of user '%s': %s", user.Username, err)
		}
		if err != nil || pile.Count == 0 {
			c.Redirect(http.StatusSeeOther, "/inventory?tab=boosters")
			return
		}

		builder := views.PageBuilder("base", map[string]any{
			"Title": dc.Renderer.T(c, "pack-opening-title"),
			"User":  user,
		})
		builder.Add("pack-opening", "Content", map[string]any{
			"BoosterID": defaultBoosterRef,
			"Pile":      dc.pileTileData(c, &pile),
		})
		dc.Renderer.Render(c, &builder)
		return
	}

	// Unknown, already opened or someone else's booster: back to the inventory
	var booster models.Booster
	if err := dc.DB.
		Where("id = ? AND owner_id = ?", c.Query("booster"), user.ID).
		First(&booster).
		Error; err != nil {
		c.Redirect(http.StatusSeeOther, "/inventory?tab=boosters")
		return
	}
	collection := dc.Data.GetCollection(int(booster.CollectionID))
	if collection == nil {
		c.Redirect(http.StatusSeeOther, "/inventory?tab=boosters")
		return
	}

	loc := dc.Renderer.Localizer.GetLocale(c)
	boosterData, err := getCollectionData(c, loc, dc.Renderer, collection)
	if err != nil {
		log.Printf("[WARN] Failed to build data for collection '%s': %s", collection.Name, err)
		c.Redirect(http.StatusSeeOther, "/inventory?tab=boosters")
		return
	}
	boosterData["CollectionDescription"] = dc.Renderer.T(c, "collection-booster", "count", int(booster.Size))

	builder := views.PageBuilder("base", map[string]any{
		"Title": dc.Renderer.T(c, "pack-opening-title"),
		"User":  user,
	})
	builder.Add("pack-opening", "Content", map[string]any{
		"BoosterID": booster.ID,
		"Booster":   map[string]any{"Data": boosterData},
	})
	dc.Renderer.Render(c, &builder)
}

// packCard renders one drawn card with its foil ("" for none) as an HTML fragment, or writes the failure
func packCard(dc *DataController, c *gin.Context, collection *data.Collection, card *data.Card, foil string) (string, bool) {
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
	// The reel scrolls cards into view while spinning, lazy pictures would show up blank
	cardData["CardEager"] = true
	setCardFoil(cardData, foil)

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

// /opening POST: open one of the user's boosters, its cards are added to the inventory
func (dc *DataController) OpeningPost(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"message": dc.Renderer.T(c, "inventory-unauthenticated"),
		})
		return
	}

	// A default booster comes out of the user's pile of default boosters
	if c.PostForm("booster") == defaultBoosterRef {
		dc.openDefaultPost(c, user)
		return
	}

	var booster models.Booster
	if err := dc.DB.
		Where("id = ? AND owner_id = ?", c.PostForm("booster"), user.ID).
		First(&booster).
		Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-booster"),
		})
		return
	}

	collection := dc.Data.GetCollection(int(booster.CollectionID))
	if collection == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-collection"),
		})
		return
	}

	// Draw before touching the inventory, an empty collection keeps the booster
	rnd := rand.New(rand.NewPCG(
		uint64(time.Now().UnixNano()),
		uint64(time.Now().UnixNano()),
	))
	byRarity := groupByRarity(dc.Data.Cards, collection)
	pack := drawPack(rnd, byRarity, int(booster.Size))
	if pack == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-empty"),
		})
		return
	}
	foils := make([]string, len(pack))
	for i := range pack {
		foils[i] = rollFoil(rnd)
	}

	// Consume the booster and award its cards at once. Deleting with the
	// owner check first means a booster can only be opened once, even when
	// the request is sent twice.
	now := time.Now()
	err := dc.DB.Transaction(func(tx *gorm.DB) error {
		deleted := tx.Where("id = ? AND owner_id = ?", booster.ID, user.ID).Delete(&models.Booster{})
		if deleted.Error != nil {
			return deleted.Error
		}
		if deleted.RowsAffected != 1 {
			return errBoosterOpened
		}

		cards := make([]models.Card, len(pack))
		for i, drawn := range pack {
			cards[i] = models.Card{
				CardID:   uint(drawn.collection.ID*1000 + drawn.card.ID),
				Acquired: now,
				Foil:     foils[i],
				OwnerID:  user.ID,
			}
		}
		return tx.Create(&cards).Error
	})
	if errors.Is(err, errBoosterOpened) {
		c.JSON(http.StatusConflict, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-booster"),
		})
		return
	}
	if err != nil {
		log.Printf("[WARN] Failed to open booster %d for user '%s': %s", booster.ID, user.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-internal"),
		})
		return
	}

	// The cards are in the inventory, what follows is only the animation
	reels, ok := dc.packReels(c, rnd, flattenByRarity(byRarity), pack, foils)
	if !ok {
		return
	}

	// Let the page chain into the next booster of the same stack
	next := uint(0)
	if nextBooster, err := dc.nextBooster(user.ID, booster.CollectionID, booster.Size); err != nil {
		log.Printf("[WARN] Failed to find next booster for user '%s': %s", user.Username, err)
	} else if nextBooster != nil {
		next = nextBooster.ID
	}
	var remaining int64
	if err := dc.DB.
		Model(&models.Booster{}).
		Where("owner_id = ? AND collection_id = ? AND size = ?", user.ID, booster.CollectionID, booster.Size).
		Count(&remaining).
		Error; err != nil {
		log.Printf("[WARN] Failed to count boosters for user '%s': %s", user.Username, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"reels":          reels,
		"next":           next,
		"remaining":      remaining,
		"remainingLabel": dc.Renderer.T(c, "pack-opening-remaining", "count", int(remaining)),
		"colors": gin.H{
			"primary":   collection.ColorPrimary,
			"secondary": collection.ColorSecondary,
		},
	})
}

var errBoosterOpened = errors.New("booster already opened")

// openDefaultPost opens one default booster out of the user's pile, its
// cards are added to the inventory
func (dc *DataController) openDefaultPost(c *gin.Context, user *controllers.AuthenticatedUser) {
	// Draw before touching the pile, an empty catalog keeps the booster
	rnd := rand.New(rand.NewPCG(
		uint64(time.Now().UnixNano()),
		uint64(time.Now().UnixNano()),
	))
	pool := dc.allCardsPool()
	pack := drawDefaultPack(rnd, pool, defaultBoosterSize)
	if pack == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-empty"),
		})
		return
	}
	foils := make([]string, len(pack))
	for i := range pack {
		foils[i] = rollFoil(rnd)
	}

	// Take the booster out of the pile and award its cards at once. The
	// conditional consume keeps concurrent openings from digging into an
	// empty pile.
	now := time.Now()
	var pile models.BoosterPile
	err := dc.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		pile, err = controllers.ConsumePileBooster(tx, user.ID)
		if err != nil {
			return err
		}

		cards := make([]models.Card, len(pack))
		for i, drawn := range pack {
			cards[i] = models.Card{
				CardID:   uint(drawn.collection.ID*1000 + drawn.card.ID),
				Acquired: now,
				Foil:     foils[i],
				OwnerID:  user.ID,
			}
		}
		return tx.Create(&cards).Error
	})
	if errors.Is(err, controllers.ErrNoPileBooster) {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-pile"),
		})
		return
	}
	if err != nil {
		log.Printf("[WARN] Failed to open a default booster for user '%s': %s", user.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-internal"),
		})
		return
	}

	// The cards are in the inventory, what follows is only the animation
	reels, ok := dc.packReels(c, rnd, pool, pack, foils)
	if !ok {
		return
	}

	// Let the page chain into the next default booster of the pile
	next := ""
	if pile.Count > 0 {
		next = defaultBoosterRef
	}
	c.JSON(http.StatusOK, gin.H{
		"reels":          reels,
		"next":           next,
		"remaining":      pile.Count,
		"remainingLabel": dc.Renderer.T(c, "pack-opening-remaining", "count", int(pile.Count)),
		"colors": gin.H{
			"primary":   defaultBoosterColorPrimary,
			"secondary": defaultBoosterColorSecondary,
		},
	})
}

// Collections sorted by ID, with their translated title
func (dc *DataController) collectionChoices(c *gin.Context) []map[string]any {
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
	return collections
}

// /add_booster GET: debug page to add a booster to the user's inventory
func (dc *DataController) AddBoosterGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	builder := views.PageBuilder("base", map[string]any{
		// Debug page, not localized
		"Title": "Add a booster (debug)",
		"User":  user,
	})
	builder.Add("add-booster", "Content", map[string]any{
		"Collections": dc.collectionChoices(c),
	})
	dc.Renderer.Render(c, &builder)
}

// /add_booster POST: add a booster of a collection, then show the boosters
func (dc *DataController) AddBoosterPost(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	if user == nil {
		c.String(http.StatusForbidden, "You must be authenticated")
		return
	}

	id, err := strconv.Atoi(c.PostForm("collection"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid collection")
		return
	}
	collection := dc.Data.GetCollection(id)
	if collection == nil {
		c.String(http.StatusBadRequest, "invalid collection")
		return
	}

	booster := models.Booster{
		CollectionID: uint(collection.ID),
		Size:         uint(collection.BoosterSize),
		Acquired:     time.Now(),
		OwnerID:      user.ID,
	}
	if err := dc.DB.Create(&booster).Error; err != nil {
		c.String(http.StatusInternalServerError, "failed to insert booster: %s", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/inventory?tab=boosters")
}
