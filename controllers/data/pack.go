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

// drawPack draws `size` cards from a collection. The rarity of each card is
// rolled first, then a card is picked uniformly among that tier's cards.
// Duplicates are allowed. Returns nil when the collection holds no card.
func drawPack(rnd *rand.Rand, byRarity map[string][]*data.Card, size int) []*data.Card {
	available := make(map[string]bool, len(rarityTiers))
	for _, tier := range rarityTiers {
		available[tier] = len(byRarity[tier]) > 0
	}

	pack := make([]*data.Card, 0, size)
	for i := 0; i < size; i++ {
		tier := pickTier(rollRarity(rnd), available)
		if tier == "" {
			return nil
		}
		pack = append(pack, byRarity[tier][rnd.N(len(byRarity[tier]))])
	}
	return pack
}

// Chance in percent for each card of a booster to be of a rarity, after the
// fallback of the tiers the collection lacks. Tiers that can't drop are left out.
func boosterOdds(byRarity map[string][]*data.Card) map[string]int {
	available := make(map[string]bool, len(rarityTiers))
	for _, tier := range rarityTiers {
		available[tier] = len(byRarity[tier]) > 0
	}
	weights := map[string]int{
		"common":    weightCommon,
		"rare":      weightRare,
		"legendary": weightLegendary,
	}
	total := weightCommon + weightRare + weightLegendary
	odds := make(map[string]int, len(rarityTiers))
	for _, rolled := range rarityTiers {
		if tier := pickTier(rolled, available); tier != "" {
			odds[tier] += weights[rolled] * 100 / total
		}
	}
	return odds
}

// Number of boosters left in a stack (same collection and size) of a user
func (dc *DataController) countBoosters(ownerID uint, collectionID uint, size uint) (int64, error) {
	var count int64
	err := dc.DB.
		Model(&models.Booster{}).
		Where("owner_id = ? AND collection_id = ? AND size = ?", ownerID, collectionID, size).
		Count(&count).
		Error
	return count, err
}

// Number of distinct cards of a collection owned by a user
func (dc *DataController) countOwnedCards(ownerID uint, collection *data.Collection) (int64, error) {
	var count int64
	err := dc.DB.
		Model(&models.Card{}).
		Where("owner_id = ? AND card_id > ? AND card_id < ?", ownerID, collection.ID*1000, (collection.ID+1)*1000).
		Distinct("card_id").
		Count(&count).
		Error
	return count, err
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

// /opening GET: open one of the user's boosters (?booster=ID), or boosters
// of a collection bought with coins (?collection=ID)
func (dc *DataController) OpeningGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Unknown, already opened or someone else's booster: back to the inventory
	var booster models.Booster
	var collection *data.Collection
	var size uint
	if id, err := strconv.Atoi(c.Query("collection")); err == nil && c.Query("booster") == "" {
		// Shop boosters: the size the shop sells
		collection = dc.Data.GetCollection(id)
		if collection != nil {
			size = uint(collection.BoosterSize)
		}
	} else if err := dc.DB.
		Where("id = ? AND owner_id = ?", c.Query("booster"), user.ID).
		First(&booster).
		Error; err == nil {
		collection = dc.Data.GetCollection(int(booster.CollectionID))
		size = booster.Size
	}
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
	boosterData["CollectionDescription"] = dc.Renderer.T(c, "collection-booster", "count", int(size))
	collectionTitle, err := collection.Translate(loc, "title")
	if err != nil {
		log.Printf("[WARN] Failed to translate collection '%s': %s", collection.Name, err)
		collectionTitle = collection.Name
	}

	// Side information, the page still works without it
	remaining, err := dc.countBoosters(user.ID, uint(collection.ID), size)
	if err != nil {
		log.Printf("[WARN] Failed to count boosters for user '%s': %s", user.Username, err)
	}
	owned, err := dc.countOwnedCards(user.ID, collection)
	if err != nil {
		log.Printf("[WARN] Failed to count cards for user '%s': %s", user.Username, err)
	}
	total := len(collection.Cards)
	progress := 0
	if total > 0 {
		progress = int(owned) * 100 / total
	}

	byRarity := groupByRarity(dc.Data.Cards, collection)
	odds := boosterOdds(byRarity)
	rates := make([]map[string]any, 0, len(rarityTiers))
	for i := len(rarityTiers) - 1; i >= 0; i-- {
		tier := rarityTiers[i]
		if odds[tier] == 0 {
			continue
		}
		rates = append(rates, map[string]any{
			"Rarity":  tier,
			"Label":   dc.Renderer.T(c, "card-rarity-"+tier),
			"Percent": odds[tier],
		})
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": dc.Renderer.T(c, "pack-opening-title"),
		"User":  user,
	})
	builder.Add("pack-opening", "Content", map[string]any{
		"BoosterID":       booster.ID,
		"Booster":         map[string]any{"Data": boosterData},
		"CollectionTitle": collectionTitle,
		"ColorPrimary":    collection.ColorPrimary,
		"ColorSecondary":  collection.ColorSecondary,
		"BoosterSize":     dc.Renderer.T(c, "collection-booster", "count", int(size)),
		"Remaining":       dc.Renderer.T(c, "pack-opening-remaining", "count", int(remaining)),
		"Owned":           owned,
		"Total":           total,
		"Progress":        progress,
		"Rates":           rates,
		"FoilChance":      foilChance,
		// Once out of boosters, more are bought with coins
		"CollectionID": collection.ID,
		"Coins":        user.Coins,
		"CoinsLabel":   dc.Renderer.T(c, "shop-price", "amount", user.Coins),
		"Price":        controllers.BoosterPrice(collection.ID),
		"PriceLabel":   dc.Renderer.T(c, "shop-price", "amount", controllers.BoosterPrice(collection.ID)),
		"Topups":       controllers.TopupChoices(c, dc.Renderer),
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

	// Deleting with the owner check first means a booster can only be
	// opened once, even when the request is sent twice
	dc.openPack(c, user, collection, booster.Size, func(tx *gorm.DB) error {
		deleted := tx.Where("id = ? AND owner_id = ?", booster.ID, user.ID).Delete(&models.Booster{})
		if deleted.Error != nil {
			return deleted.Error
		}
		if deleted.RowsAffected != 1 {
			return errBoosterOpened
		}
		return nil
	})
}

// /opening/buy POST: buy a booster of a collection with coins and open it at once
func (dc *DataController) OpeningBuyPost(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"message": dc.Renderer.T(c, "inventory-unauthenticated"),
		})
		return
	}

	id, err := strconv.Atoi(c.PostForm("collection"))
	collection := dc.Data.GetCollection(id)
	if err != nil || collection == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-collection"),
		})
		return
	}

	// Debit only when the balance covers the price, so concurrent
	// purchases cannot overdraw
	price := controllers.BoosterPrice(collection.ID)
	dc.openPack(c, user, collection, uint(collection.BoosterSize), func(tx *gorm.DB) error {
		result := tx.Model(&models.User{}).
			Where("id = ? AND coins >= ?", user.ID, price).
			Update("coins", gorm.Expr("coins - ?", price))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errNotEnoughCoins
		}
		return nil
	})
}

// openPack draws a pack of a collection and adds its cards to the user's
// inventory, in the same transaction as pay, which takes what the pack costs.
// Replies with the reels to animate.
func (dc *DataController) openPack(c *gin.Context, user *controllers.AuthenticatedUser, collection *data.Collection, size uint, pay func(tx *gorm.DB) error) {
	// Draw before paying, an empty collection costs nothing
	rnd := rand.New(rand.NewPCG(
		uint64(time.Now().UnixNano()),
		uint64(time.Now().UnixNano()),
	))
	byRarity := groupByRarity(dc.Data.Cards, collection)
	pack := drawPack(rnd, byRarity, int(size))
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

	// Pay and award the cards at once
	now := time.Now()
	err := dc.DB.Transaction(func(tx *gorm.DB) error {
		if err := pay(tx); err != nil {
			return err
		}

		cards := make([]models.Card, len(pack))
		for i, card := range pack {
			cards[i] = models.Card{
				CardID:   uint(collection.ID*1000 + card.ID),
				Acquired: now,
				Foil:     foils[i],
				OwnerID:  user.ID,
			}
		}
		return tx.Create(&cards).Error
	})
	switch {
	case errors.Is(err, errBoosterOpened):
		c.JSON(http.StatusConflict, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-booster"),
		})
		return
	case errors.Is(err, errNotEnoughCoins):
		c.JSON(http.StatusPaymentRequired, gin.H{
			"message": dc.Renderer.T(c, "shop-error-funds"),
		})
		return
	case err != nil:
		log.Printf("[WARN] Failed to open a booster of collection %d for user '%s': %s", collection.ID, user.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": dc.Renderer.T(c, "pack-opening-error-internal"),
		})
		return
	}

	// The cards are in the inventory, what follows is only the animation
	pool := flattenByRarity(byRarity)
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
			html, ok := packCard(dc, c, collection, entry.Card, foil)
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

	// Let the page chain into the next booster of the same stack
	next := uint(0)
	if nextBooster, err := dc.nextBooster(user.ID, uint(collection.ID), size); err != nil {
		log.Printf("[WARN] Failed to find next booster for user '%s': %s", user.Username, err)
	} else if nextBooster != nil {
		next = nextBooster.ID
	}
	remaining, err := dc.countBoosters(user.ID, uint(collection.ID), size)
	if err != nil {
		log.Printf("[WARN] Failed to count boosters for user '%s': %s", user.Username, err)
	}
	// Refreshes the collection progress of the page
	owned, err := dc.countOwnedCards(user.ID, collection)
	if err != nil {
		log.Printf("[WARN] Failed to count cards for user '%s': %s", user.Username, err)
	}
	// Tells the page whether the next booster can be bought
	coins := user.Coins
	if err := dc.DB.Model(&models.User{}).Where("id = ?", user.ID).Select("coins").Scan(&coins).Error; err != nil {
		log.Printf("[WARN] Failed to read the balance of user '%s': %s", user.Username, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"reels":          reels,
		"next":           next,
		"remaining":      remaining,
		"remainingLabel": dc.Renderer.T(c, "pack-opening-remaining", "count", int(remaining)),
		"owned":          owned,
		"coins":          coins,
		"coinsLabel":     dc.Renderer.T(c, "shop-price", "amount", coins),
		"colors": gin.H{
			"primary":   collection.ColorPrimary,
			"secondary": collection.ColorSecondary,
		},
	})
}

var (
	errBoosterOpened  = errors.New("booster already opened")
	errNotEnoughCoins = errors.New("not enough coins")
)

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
