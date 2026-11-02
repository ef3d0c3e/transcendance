package data

import (
	"errors"
	"log"
	"math/rand/v2"
	"net/http"
	"sort"
	"strconv"
	"strings"
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

// drawDefaultPack draws `size` cards among the pool of every collection. It
// rolls the same rarity drop table as a collection booster, then picks a card
// uniformly among the pool's cards of that tier. Duplicates are allowed.
// Returns nil when the pool holds no card.
func drawDefaultPack(rnd *rand.Rand, pool []poolCard, size int) []poolCard {
	return drawPack(rnd, groupPoolByRarity(pool), size)
}

// groupPoolByRarity indexes the cards of a pool by rarity
func groupPoolByRarity(pool []poolCard) map[string][]poolCard {
	byRarity := make(map[string][]poolCard, len(rarityTiers))
	for _, entry := range pool {
		byRarity[entry.card.Rarity] = append(byRarity[entry.card.Rarity], entry)
	}
	return byRarity
}

// Chance in percent for each card of a booster to be of a rarity, after the
// fallback of the tiers the collection lacks. Tiers that can't drop are left out.
func boosterOdds(byRarity map[string][]poolCard) map[string]int {
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

// packCardIDs returns the inventory ID of each drawn card: its collection,
// then its place in it
func packCardIDs(pack []poolCard) []uint {
	ids := make([]uint, len(pack))
	for i, drawn := range pack {
		ids[i] = uint(drawn.collection.ID*1000 + drawn.card.ID)
	}
	return ids
}

// ownedCardIDs returns, among the given card IDs, the ones the owner has at
// least one copy of
func ownedCardIDs(tx *gorm.DB, ownerID uint, ids []uint) (map[uint]bool, error) {
	var owned []uint
	if err := tx.Model(&models.Card{}).
		Where("owner_id = ? AND card_id IN ?", ownerID, ids).
		Pluck("card_id", &owned).
		Error; err != nil {
		return nil, err
	}
	have := make(map[uint]bool, len(owned))
	for _, id := range owned {
		have[id] = true
	}
	return have, nil
}

// firstCopies marks the drawn cards that are new to their owner: the ones
// they have no copy of, where the copies drawn before in the same pack count
// as theirs already
func firstCopies(owned map[uint]bool, drawn []uint) []bool {
	have := make(map[uint]bool, len(owned)+len(drawn))
	for id := range owned {
		have[id] = true
	}
	news := make([]bool, len(drawn))
	for i, id := range drawn {
		news[i] = !have[id]
		have[id] = true
	}
	return news
}

// newCards tells which cards of a drawn pack are new to their owner, foil or
// not. Must run before the cards are added to the inventory.
func newCards(tx *gorm.DB, ownerID uint, pack []poolCard) ([]bool, error) {
	drawn := packCardIDs(pack)
	owned, err := ownedCardIDs(tx, ownerID, drawn)
	if err != nil {
		return nil, err
	}
	return firstCopies(owned, drawn), nil
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
func (dc *DataController) packReels(c *gin.Context, rnd *rand.Rand, pool []poolCard, pack []poolCard, foils []string, news []bool) ([][]gin.H, bool) {
	reels := make([][]gin.H, 0, len(pack))
	for i, winner := range pack {
		strip := buildReel(rnd, pool, winner)

		slots := make([]gin.H, 0, len(strip))
		for _, entry := range strip {
			// Only the won card shows the foil it was obtained with, and
			// whether it is new to its owner
			foil := ""
			isNew := false
			if entry.Win {
				foil = foils[i]
				isNew = news[i]
			}
			html, ok := packCard(dc, c, entry.Collection, entry.Card, foil, isNew)
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

// /opening GET: open one of the user's boosters (?booster=ID), or boosters
// of a collection bought with coins (?collection=ID)
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

		// Drawn from every collection, so there is no collection progress
		// nor any to buy more of
		byRarity := groupPoolByRarity(dc.allCardsPool())
		coins := user.FreeCoins()
		// Without its countdown: reloading the page when it runs out would
		// open a booster on arrival, unasked
		tile := dc.pileTileData(c, &pile)
		tile.PileTimerLabel = ""
		builder := views.PageBuilder("base", map[string]any{
			"Title": dc.Renderer.T(c, "pack-opening-title"),
			"User":  user,
		})
		builder.Add("pack-opening", "Content", map[string]any{
			"BoosterID":       defaultBoosterRef,
			"Pile":            tile,
			"CollectionTitle": dc.Renderer.T(c, "booster-pile-title"),
			"ColorPrimary":    defaultBoosterColorPrimary,
			"ColorSecondary":  defaultBoosterColorSecondary,
			"BoosterSize":     dc.Renderer.T(c, "collection-booster", "count", defaultBoosterSize),
			"Remaining":       dc.Renderer.T(c, "pack-opening-remaining", "count", int(pile.Count)),
			"Rates":           dc.boosterRates(c, byRarity),
			"FoilChance":      foilChance,
			"Coins":           coins,
			"CoinsLabel":      dc.Renderer.T(c, "shop-price", "amount", coins),
			"Topups":          controllers.TopupChoices(c, dc.Renderer),
		})
		dc.Renderer.Render(c, &builder)
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

	coins := user.FreeCoins()
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
		"Rates":           dc.boosterRates(c, groupByRarity(dc.Data.Cards, collection)),
		"FoilChance":      foilChance,
		// Once out of boosters, more are bought with coins
		"CollectionID": collection.ID,
		"Coins":        coins,
		"CoinsLabel":   dc.Renderer.T(c, "shop-price", "amount", coins),
		"Price":        controllers.BoosterPrice(collection.ID),
		"PriceLabel":   dc.Renderer.T(c, "shop-price", "amount", controllers.BoosterPrice(collection.ID)),
		"Topups":       controllers.TopupChoices(c, dc.Renderer),
	})
	dc.Renderer.Render(c, &builder)
}

// boosterRates lists the drop rate of each rarity that can drop, rarest first
func (dc *DataController) boosterRates(c *gin.Context, byRarity map[string][]poolCard) []map[string]any {
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
	return rates
}

// freeCoins reads again the coins a user can spend, once a purchase changed
// them: the ones up in their pending trades are left out
func (dc *DataController) freeCoins(user *controllers.AuthenticatedUser) uint {
	balance := user.Coins
	if err := dc.DB.Model(&models.User{}).Where("id = ?", user.ID).Select("coins").Scan(&balance).Error; err != nil {
		log.Printf("[WARN] Failed to read the balance of user '%s': %s", user.Username, err)
	}
	locked, err := controllers.TradeCommittedCoins(dc.DB, user.ID)
	if err != nil {
		log.Printf("[WARN] Failed to get the coins user '%s' put up in trades: %s", user.Username, err)
	}
	return controllers.FreeCoins(balance, locked)
}

// packCard renders one drawn card with its foil ("" for none) and its new
// mark, as an HTML fragment, or writes the failure
func packCard(dc *DataController, c *gin.Context, collection *data.Collection, card *data.Card, foil string, isNew bool) (string, bool) {
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
	if isNew {
		cardData["CardNew"] = dc.Renderer.T(c, "pack-opening-new")
	}

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

// packSummary lists the drawn cards in a sentence, announced to screen
// readers once the reels stopped, or writes the failure
func packSummary(dc *DataController, c *gin.Context, pack []poolCard, foils []string, news []bool) (string, bool) {
	loc := dc.Renderer.Localizer.GetLocale(c)

	labels := make([]string, len(pack))
	for i, drawn := range pack {
		card := drawn.card
		title, err := card.Translate(loc, "title")
		if err != nil {
			log.Printf("[WARN] Failed to translate card '%s': %s", card.Name, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": dc.Renderer.T(c, "pack-opening-error-internal"),
			})
			return "", false
		}
		key := "pack-opening-card"
		if foils[i] != "" {
			key = "pack-opening-card-foil"
		}
		if news[i] {
			// The owner's first copy of that card
			key += "-new"
		}
		labels[i] = dc.Renderer.T(c, key, "title", title, "rarity", dc.Renderer.T(c, "card-rarity-"+card.Rarity))
	}
	return dc.Renderer.T(c, "pack-opening-summary", "cards", strings.Join(labels, ", ")), true
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

	// Debit only when the free balance covers the price, so concurrent
	// purchases cannot overdraw. Coins up in pending trades leave with those
	// trades, so they cannot pay here
	price := controllers.BoosterPrice(collection.ID)
	dc.openPack(c, user, collection, uint(collection.BoosterSize), func(tx *gorm.DB) error {
		if err := controllers.LockUsers(tx, user.ID); err != nil {
			return err
		}
		locked, err := controllers.TradeCommittedCoins(tx, user.ID)
		if err != nil {
			return err
		}
		result := tx.Model(&models.User{}).
			Where("id = ? AND coins >= ?", user.ID, price+locked).
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
	rnd := newRand()
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
	var news []bool
	err := dc.DB.Transaction(func(tx *gorm.DB) error {
		if err := pay(tx); err != nil {
			return err
		}

		// The inventory is untouched yet: the copies it already holds are
		// not new
		var err error
		news, err = newCards(tx, user.ID, pack)
		if err != nil {
			return err
		}

		ids := packCardIDs(pack)
		cards := make([]models.Card, len(pack))
		for i := range pack {
			cards[i] = models.Card{
				CardID:   ids[i],
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
	summary, ok := packSummary(dc, c, pack, foils, news)
	if !ok {
		return
	}
	reels, ok := dc.packReels(c, rnd, flattenByRarity(byRarity), pack, foils, news)
	if !ok {
		return
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
	coins := dc.freeCoins(user)

	c.JSON(http.StatusOK, gin.H{
		"reels":          reels,
		"next":           next,
		"remaining":      remaining,
		"remainingLabel": dc.Renderer.T(c, "pack-opening-remaining", "count", int(remaining)),
		"owned":          owned,
		"summary":        summary,
		"coins":          coins,
		"coinsLabel":     dc.Renderer.T(c, "shop-price", "amount", coins),
		"navBalance":     dc.Renderer.T(c, "header-balance", "amount", coins),
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

// openDefaultPost opens one default booster out of the user's pile, its
// cards are added to the inventory
func (dc *DataController) openDefaultPost(c *gin.Context, user *controllers.AuthenticatedUser) {
	// Draw before touching the pile, an empty catalog keeps the booster
	rnd := newRand()
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
	var news []bool
	err := dc.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		pile, err = controllers.ConsumePileBooster(tx, user.ID)
		if err != nil {
			return err
		}

		// The inventory is untouched yet: the copies it already holds are
		// not new
		news, err = newCards(tx, user.ID, pack)
		if err != nil {
			return err
		}

		ids := packCardIDs(pack)
		cards := make([]models.Card, len(pack))
		for i := range pack {
			cards[i] = models.Card{
				CardID:   ids[i],
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
	summary, ok := packSummary(dc, c, pack, foils, news)
	if !ok {
		return
	}
	reels, ok := dc.packReels(c, rnd, pool, pack, foils, news)
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
		"summary":        summary,
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
	if !user.IsAdmin() {
		c.String(http.StatusForbidden, "Admins only")
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
	if !user.IsAdmin() {
		c.String(http.StatusForbidden, "Admins only")
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
