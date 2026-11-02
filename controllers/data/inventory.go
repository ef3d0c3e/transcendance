package data

import (
	"cmp"
	"fmt"
	"log"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"transcendance/controllers"
	"transcendance/data"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

const (
	inventoryPageSize = 20
)

// /add POST: debug tool adding the card `id` to the admin's inventory
func (dc *DataController) InventoryAddPost(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)

	if !user.IsAdmin() {
		c.String(http.StatusForbidden, "Admins only")
		return
	}

	cardId, err := strconv.Atoi(c.PostForm("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}

	card, collection := dc.Data.GetCard(cardId)
	if card == nil || collection == nil {
		c.String(http.StatusBadRequest, "invalid card")
		return
	}

	row := models.Card{
		CardID:   uint(cardId),
		Acquired: time.Now(),
		OwnerID:  user.ID,
	}
	if err := dc.DB.Create(&row).Error; err != nil {
		c.String(http.StatusInternalServerError, "failed to insert card: %s", err)
		return
	}
	c.Status(http.StatusCreated)
}

const (
	inventoryTabCards    = "cards"
	inventoryTabBoosters = "boosters"
	// Every collection, with the user's progress
	inventoryTabCollections = "collections"
	// One's own cards, to trade up for the next rarity
	inventoryTabTradeup = "tradeup"
	// One's own cards, to roll one for a foil
	inventoryTabFoil = "foil"
)

// Effect filter values, "" shows every card
const (
	inventoryEffectNone = "none"
	inventoryEffectFoil = "foil"
)

var inventoryEffects = []string{inventoryEffectNone, inventoryEffectFoil}

// "show" value listing every card of the catalog, "" lists the owned ones
const inventoryShowAll = "all"

// Card sorts, relevance is only available when searching
const (
	inventorySortRelevance = "relevance"
	inventorySortRecent    = "recent"
	inventorySortOldest    = "oldest"
	inventorySortRarity    = "rarity"
	inventorySortCount     = "count"
	inventorySortName      = "name"
	inventorySortNameDesc  = "name-desc"
)

var inventorySorts = []string{
	inventorySortRelevance,
	inventorySortRecent,
	inventorySortOldest,
	inventorySortRarity,
	inventorySortCount,
	inventorySortName,
	inventorySortNameDesc,
}

var inventoryRarities = []string{"common", "rare", "legendary"}

func defaultInventorySort(query string) string {
	if strings.TrimSpace(query) != "" {
		return inventorySortRelevance
	}
	return inventorySortRecent
}

func (v *inventoryView) sorts() []string {
	if v.All {
		return append(slices.Clone(catalogSorts), catalogSortMissing)
	}
	return inventorySorts
}

func (v *inventoryView) defaultSort() string {
	if v.All {
		return defaultCatalogSort(v.Query)
	}
	return defaultInventorySort(v.Query)
}

type inventoryView struct {
	// Displayed tab, one of the inventoryTab constants
	Tab string
	// Viewed user
	Username string
	Query    string
	// Every card of the catalog instead of the owned ones
	All bool
	// Card filters, "" when unfiltered
	Rarity string
	Effect string
	// Card sort, always set on the cards tab
	Sort string
	// Error or empty state message
	Message string
	// Rendered items of the page
	Items []map[string]any
	Page  int
	// Number of items to paginate (card stacks, or booster stacks)
	PageItems int64
	// Count shown next to the title (cards, or boosters)
	Count int64
	// Count shown in each tab
	TotalCards       int64
	TotalCollections int64
	// Daily streak section, shown on one's own inventory
	Streak *streakTile
	// Pile of default boosters, shown on one's own boosters tab
	Pile *pileTile
	// Someone else, with neither side blocking the other
	Tradable bool
	// Pending offers made to the user, on their own inventory
	IncomingTrades int64
	// What selling the duplicates would take and pay, on one's own cards tab
	Duplicates *controllers.DuplicateSale
	// Collections the tradeup grid holds, for its filter
	Collections []controllers.SelectOption
	// Card the last tradeup awarded or foil roll was for, nil until one completes
	Result map[string]any
	// Whether the last foil roll was lost, destroying the card
	ResultLost bool
}

func (dc *DataController) renderInventory(c *gin.Context, user *controllers.AuthenticatedUser, v *inventoryView) {
	var title string
	if user.Username == v.Username {
		title = dc.Renderer.T(c, "inventory-title")
	} else {
		title = dc.Renderer.T(c, "inventory-title-other", "username", v.Username)
	}
	builder := views.PageBuilder("base", map[string]any{
		"Title": title,
		"User":  user,
	})
	if v.Items == nil {
		v.Items = make([]map[string]any, 0)
	}
	searching := v.Tab == inventoryTabCards && (strings.TrimSpace(v.Query) != "" || v.Rarity != "" || v.Effect != "")
	if v.Message == "" && len(v.Items) == 0 {
		switch {
		case searching && v.All:
			v.Message = dc.Renderer.T(c, "card-search-empty")
		case searching:
			v.Message = dc.Renderer.T(c, "inventory-message-no-match")
		case v.Tab == inventoryTabBoosters:
			// The pile of default boosters shows its own tile, even empty
			if v.Pile == nil || v.Pile.PileEmpty {
				v.Message = dc.Renderer.T(c, "inventory-message-no-boosters")
			}
		case v.Tab == inventoryTabTradeup:
			v.Message = dc.Renderer.T(c, "tradeup-empty")
		case v.Tab == inventoryTabFoil:
			v.Message = dc.Renderer.T(c, "foil-empty")
		default:
			v.Message = dc.Renderer.T(c, "inventory-message-empty")
		}
	}

	count := ""
	if v.Count > 0 {
		switch {
		case searching:
			count = dc.Renderer.T(c, "inventory-search-count", "count", int(v.Count))
		case v.Tab == inventoryTabCollections:
			count = dc.Renderer.T(c, "catalog-collection-count", "count", int(v.Count))
		default:
			count = dc.Renderer.T(c, "inventory-count", "count", int(v.Count))
		}
	}
	// Keep the viewed user in the tabs, search form and paginator links
	viewed := ""
	if user.Username != v.Username {
		viewed = v.Username
	}
	tabUrl := func(tab string) string {
		params := url.Values{}
		if viewed != "" {
			params.Set("u", viewed)
		}
		if tab != inventoryTabBoosters {
			params.Set("tab", tab)
		}
		if len(params) == 0 {
			return "/inventory"
		}
		return "/inventory?" + params.Encode()
	}

	cardsUrl := func(all bool) string {
		params := url.Values{"tab": {inventoryTabCards}}
		if viewed != "" {
			params.Set("u", viewed)
		}
		if all {
			params.Set("show", inventoryShowAll)
		}
		return "/inventory?" + params.Encode()
	}
	totalPages := v.PageItems / int64(inventoryPageSize)
	if v.PageItems%int64(inventoryPageSize) != 0 {
		totalPages = totalPages + 1
	}
	sellAction := ""
	if viewed == "" {
		sellAction = "/shop/sell"
	}

	tradeupUrl, foilUrl := "", ""
	if viewed == "" {
		tradeupUrl = tabUrl(inventoryTabTradeup)
		foilUrl = tabUrl(inventoryTabFoil)
	}
	fields := map[string]any{
		"Title":          title,
		"Message":        v.Message,
		"Count":          count,
		"Tab":            v.Tab,
		"TabCardsUrl":    tabUrl(inventoryTabCards),
		"TabBoostersUrl": tabUrl(inventoryTabBoosters),
		"TotalCards":     v.TotalCards,
		"Streak":         v.Streak,
		"Pile":           v.Pile,
		// Collections are part of the catalog: the same for every user
		"TabCollectionsUrl": tabUrl(inventoryTabCollections),
		"TotalCollections":  v.TotalCollections,
		"Items":             v.Items,
		"Query":             v.Query,
		"Filtered":          searching,
		"All":               v.All,
		"ShowOwnedUrl":      cardsUrl(false),
		"ShowAllUrl":        cardsUrl(true),
		// Clears the search and filters, keeping the toggle
		"ClearUrl": cardsUrl(v.All),
		"Viewed":   viewed,
		// One's own cards are sold from the card popup, and their duplicates at once
		"SellAction":         sellAction,
		"Duplicates":         v.Duplicates,
		"TabTradeupUrl":      tradeupUrl,
		"TradeupCost":        tradeupCost,
		"TradeupResult":      v.Result,
		"TradeupCollections": v.Collections,
		// The tradeup and foil grids are paged client-side, keeping the selection
		"PickerPageSize": inventoryPageSize,
		"TabFoilUrl":     foilUrl,
		"FoilResult":     v.Result,
		"FoilResultLost": v.ResultLost,
		"Tradable":       v.Tradable,
		"IncomingTrades": v.IncomingTrades,
		"Pages":          int(totalPages),
		"Locale":         dc.Renderer.Localizer.GetLocale(c),
		// Outcome of buying boosters or coins, which land here
		"Notice": controllers.GetShopNotice(c, dc.Renderer, dc.Data),
	}

	if v.Tab == inventoryTabBoosters && viewed == "" {
		// Coins up in pending trades are left out of the balance and said apart
		coins := user.FreeCoins()
		fields["CoinsLabel"] = dc.Renderer.T(c, "shop-price", "amount", coins)
		if locked := user.Coins - coins; locked > 0 {
			fields["CoinsLockedLabel"] = dc.Renderer.T(c, "shop-coins-locked", "amount", int(locked))
		}
		fields["Topups"] = controllers.TopupChoices(c, dc.Renderer)
	}
	if v.Tab == inventoryTabFoil {
		dc.foilFields(c, v, fields)
	}
	if v.Tab == inventoryTabCards {
		fields["RarityOptions"] = controllers.SelectOptions(append([]string{""}, inventoryRarities...), v.Rarity, func(value string) string {
			if value == "" {
				return dc.Renderer.T(c, "inventory-filter-rarity-all")
			}
			return dc.Renderer.T(c, "card-rarity-"+value)
		})
		fields["EffectOptions"] = controllers.SelectOptions(append([]string{""}, inventoryEffects...), v.Effect, func(value string) string {
			if value == "" {
				return dc.Renderer.T(c, "inventory-filter-effect-all")
			}
			return dc.Renderer.T(c, "inventory-filter-effect-"+value)
		})
		defaultSort := v.defaultSort()
		sorts := v.sorts()
		if defaultSort != inventorySortRelevance {
			sorts = sorts[1:]
		}
		sortOptions := controllers.SelectOptions(sorts, v.Sort, func(value string) string {
			switch value {
			case catalogSortNumber:
				return dc.Renderer.T(c, "catalog-sort-number")
			case catalogSortMissing:
				return dc.Renderer.T(c, "catalog-sort-missing")
			}
			return dc.Renderer.T(c, "inventory-sort-"+value)
		})
		// The default sort isn't submitted, so a new search ranks by relevance
		for i := range sortOptions {
			if sortOptions[i].Value == defaultSort {
				sortOptions[i].Value = ""
			}
		}
		fields["SortOptions"] = sortOptions
	}
	if v.Tab == inventoryTabTradeup {
		// Only the rarities a tradeup can hold
		rarities := make([]string, 0, len(inventoryRarities))
		for _, rarity := range inventoryRarities {
			if tierAbove(rarity) != "" {
				rarities = append(rarities, rarity)
			}
		}
		fields["RarityOptions"] = controllers.SelectOptions(append([]string{""}, rarities...), "", func(value string) string {
			if value == "" {
				return dc.Renderer.T(c, "inventory-filter-rarity-all")
			}
			return dc.Renderer.T(c, "card-rarity-"+value)
		})
		fields["SortOptions"] = controllers.SelectOptions([]string{"", tradeupSortGrouped}, "", func(value string) string {
			if value == "" {
				return dc.Renderer.T(c, "inventory-sort-count")
			}
			return dc.Renderer.T(c, "tradeup-sort-grouped")
		})
	}
	content := builder.Add("inventory", "Content", fields)
	params := url.Values{}
	if viewed != "" {
		params.Set("u", viewed)
	}
	if v.Tab != inventoryTabBoosters {
		params.Set("tab", v.Tab)
	}
	if v.All {
		params.Set("show", inventoryShowAll)
	}
	if v.Query != "" {
		params.Set("q", v.Query)
	}
	if v.Rarity != "" {
		params.Set("rarity", v.Rarity)
	}
	if v.Effect != "" {
		params.Set("effect", v.Effect)
	}
	if v.Tab == inventoryTabCards && v.Sort != v.defaultSort() {
		params.Set("sort", v.Sort)
	}
	paginatorUrl := "?p="
	if len(params) != 0 {
		paginatorUrl = "?" + params.Encode() + "&p="
	}
	content.AddPaginator(v.Page, int(totalPages), paginatorUrl)
	dc.Renderer.Render(c, &builder)
}

func (dc *DataController) InventoryGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)

	if user == nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	page := controllers.PageParam(c.Query("p"))
	var err error

	v := &inventoryView{
		Tab:      inventoryTabBoosters,
		Username: c.Query("u"),
		Page:     page,
	}
	// Boosters are shown unless the cards are asked for, or their search,
	// filters or sort are
	cardsAsked := c.Query("tab") == inventoryTabCards
	if c.Query("tab") == "" {
		for _, param := range []string{"show", "q", "rarity", "effect", "sort"} {
			cardsAsked = cardsAsked || c.Query(param) != ""
		}
	}
	if c.Query("tab") == inventoryTabCollections {
		v.Tab = inventoryTabCollections
	} else if c.Query("tab") == inventoryTabTradeup {
		v.Tab = inventoryTabTradeup
	} else if c.Query("tab") == inventoryTabFoil {
		v.Tab = inventoryTabFoil
	} else if cardsAsked {
		v.Tab = inventoryTabCards
		v.All = c.Query("show") == inventoryShowAll
		v.Query = c.Query("q")
		if rarity := c.Query("rarity"); slices.Contains(inventoryRarities, rarity) {
			v.Rarity = rarity
		}
		// Effects are those of owned copies
		if effect := c.Query("effect"); !v.All && slices.Contains(inventoryEffects, effect) {
			v.Effect = effect
		}
		v.Sort = v.defaultSort()
		if sort := c.Query("sort"); slices.Contains(v.sorts(), sort) &&
			(sort != inventorySortRelevance || v.Sort == inventorySortRelevance) {
			v.Sort = sort
		}
	}
	if v.Username == "" {
		v.Username = user.Username
	}
	userID := user.ID
	if strings.EqualFold(v.Username, user.Username) {
		v.Username = user.Username
	} else {
		// Someone else's inventory
		var otherUser models.User
		if err := dc.DB.Omit("avatar").Where("username = ?", v.Username).First(&otherUser).Error; err != nil {
			v.Message = dc.Renderer.T(c, "inventory-message-user-not-found")
			dc.renderInventory(c, user, v)
			return
		}
		userID = otherUser.ID
		// Trading up and rolling for foils spend the cards, so they only open
		// on one's own inventory
		if v.Tab == inventoryTabTradeup || v.Tab == inventoryTabFoil {
			v.Tab = inventoryTabBoosters
		}
		blocked, err := controllers.TradeBlocked(dc.DB, user.ID, userID)
		if err != nil {
			log.Printf("WARN failed to get the relation of users %d and %d: %s", user.ID, userID, err)
		}
		v.Tradable = err == nil && !blocked
	}

	fail := func(what string, err error) {
		log.Printf("WARN failed to get %s for user '%s': %s", what, v.Username, err)
		*v = inventoryView{Tab: v.Tab, Username: v.Username, All: v.All, Query: v.Query, Rarity: v.Rarity, Effect: v.Effect, Sort: v.Sort, Page: 1}
		v.Message = dc.Renderer.T(c, "inventory-message-fetch-fail")
		dc.renderInventory(c, user, v)
	}

<<<<<<< HEAD
	// Tab counts
	if v.TotalCards, err = dc.countCards(userID); err != nil {
=======
	if v.TotalCards, v.TotalBoosters, err = dc.inventoryCounts(userID); err != nil {
>>>>>>> master
		fail("counts", err)
		return
	}
	v.TotalCollections = int64(len(dc.Data.Collections))

	// The daily streak section shows on one's own inventory, on every tab
	own := v.Username == user.Username
	if own {
		// Counted on the link to the trades page
		if err := dc.DB.Model(&models.Trade{}).
			Where("to_id = ? AND state = ?", userID, models.TradePending).
			Count(&v.IncomingTrades).Error; err != nil {
			fail("incoming trades", err)
			return
		}

		state, err := controllers.DailyStreakState(dc.DB, userID, time.Now())
		if err != nil {
			fail("daily streak", err)
			return
		}
		v.Streak = dc.streakTileData(c, &state, v.Tab)
	}

<<<<<<< HEAD
	// The pile of default boosters tops itself up over time, and only belongs
	// to one's own inventory
=======
	// The pile of default boosters tops itself up over time. It counts in the
	// boosters tab from every tab
>>>>>>> master
	var pile *models.BoosterPile
	if own {
		accrued, err := controllers.AccrueBoosterPile(dc.DB, userID)
		if err != nil {
			fail("booster pile", err)
			return
		}
		pile = &accrued
	}

	if v.Tab == inventoryTabCollections {
		if err := dc.inventoryCollections(c, userID, v); err != nil {
			fail("collections", err)
			return
		}
	} else if v.Tab == inventoryTabTradeup {
		if err := dc.inventoryTradeup(c, userID, v); err != nil {
			fail("tradeup", err)
			return
		}
	} else if v.Tab == inventoryTabFoil {
		if err := dc.inventoryFoil(c, userID, v); err != nil {
			fail("foil", err)
			return
		}
	} else if v.Tab == inventoryTabBoosters {
		if pile != nil {
			v.Pile = dc.pileTileData(c, pile)
		}
		if err := dc.inventoryBoosters(c, userID, userID == user.ID, v); err != nil {
			fail("boosters", err)
			return
		}
	} else if v.All {
		if err := dc.inventoryAllCards(c, userID, v); err != nil {
			fail("cards", err)
			return
		}
	} else {
		// Copies up in pending trades are marked on one's own cards only
		var committed *controllers.TradeCommitments
		if own {
			if committed, err = controllers.TradeCommitmentsOf(dc.DB, userID, 0); err != nil {
				fail("trade commitments", err)
				return
			}
		}
		if err := dc.inventoryCards(c, userID, v, committed); err != nil {
			fail("cards", err)
			return
		}
	}
	if v.Tab == inventoryTabCards && own {
		sale, err := controllers.DuplicatesForSale(dc.DB, dc.Data, userID)
		if err != nil {
			fail("duplicates", err)
			return
		}
		v.Duplicates = &sale
	}

	dc.renderInventory(c, user, v)
}

// Copies of the same card with the same foil, displayed as one tile
type cardStack struct {
	CardID        uint
	Foil          string
	Count         int64
	FirstAcquired time.Time
	LastAcquired  time.Time

	card       *data.Card
	collection *data.Collection
}

// sortCardStacks breaks ties with the newest stacks first
func sortCardStacks(stacks []cardStack, sort string, loc string, relevance map[uint]int) {
	var titles map[uint]string
	var collator *collate.Collator
	if sort == inventorySortName || sort == inventorySortNameDesc {
		titles = make(map[uint]string, len(stacks))
		for _, stack := range stacks {
			title, err := stack.card.Translate(loc, "title")
			if err != nil {
				log.Printf("WARN card %d has no title: %s", stack.CardID, err)
			}
			titles[stack.CardID] = title
		}
		collator = collate.New(language.Make(loc), collate.IgnoreCase)
	}

	slices.SortStableFunc(stacks, func(a, b cardStack) int {
		var order int
		switch sort {
		case inventorySortRelevance:
			order = cmp.Compare(relevance[a.CardID], relevance[b.CardID])
		case inventorySortOldest:
			order = a.FirstAcquired.Compare(b.FirstAcquired)
		case inventorySortRarity:
			// Highest rarity first, unknown rarities last
			rank := func(s cardStack) int {
				if i := indexOfTier(s.card.Rarity); i >= 0 {
					return i
				}
				return len(rarityTiers)
			}
			order = cmp.Compare(rank(a), rank(b))
		case inventorySortCount:
			order = cmp.Compare(b.Count, a.Count)
		case inventorySortName:
			order = collator.CompareString(titles[a.CardID], titles[b.CardID])
		case inventorySortNameDesc:
			order = collator.CompareString(titles[b.CardID], titles[a.CardID])
		}
		return cmp.Or(
			order,
			b.LastAcquired.Compare(a.LastAcquired),
			cmp.Compare(a.CardID, b.CardID),
			cmp.Compare(a.Foil, b.Foil),
		)
	})
}

// committed marks the copies up in pending trades, nil on someone else's cards
func (dc *DataController) inventoryCards(c *gin.Context, userID uint, v *inventoryView, committed *controllers.TradeCommitments) error {
	loc := dc.Renderer.Localizer.GetLocale(c)

	cardsQuery := dc.DB.Model(&models.Card{}).Where("owner_id = ?", userID)
	var relevance map[uint]int
	if strings.TrimSpace(v.Query) != "" {
		matches, err := dc.searchInventory(c, userID, v.Query)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return nil
		}
		cardsQuery = cardsQuery.Where("card_id IN ?", matches)
		relevance = make(map[uint]int, len(matches))
		for i, id := range matches {
			relevance[id] = i
		}
	}
	switch v.Effect {
	case inventoryEffectNone:
		cardsQuery = cardsQuery.Where("foil = ''")
	case inventoryEffectFoil:
		cardsQuery = cardsQuery.Where("foil <> ''")
	}

	// Rarity and titles live in the card data, so stacks are filtered, sorted and paginated here
	var all []cardStack
	if err := cardsQuery.
		Select("card_id, foil, COUNT(*) AS count, MIN(acquired) AS first_acquired, MAX(acquired) AS last_acquired").
		Group("card_id, foil").
		Scan(&all).
		Error; err != nil {
		return err
	}
	stacks := all[:0]
	for _, stack := range all {
		stack.card, stack.collection = dc.Data.GetCard(int(stack.CardID))
		if stack.card == nil || stack.collection == nil {
			log.Printf("WARN user %d owns unknown card %d", userID, stack.CardID)
			continue
		}
		if v.Rarity != "" && stack.card.Rarity != v.Rarity {
			continue
		}
		stacks = append(stacks, stack)
		v.Count += stack.Count
	}
	v.PageItems = int64(len(stacks))
	sortCardStacks(stacks, v.Sort, loc, relevance)

	start := min((v.Page-1)*inventoryPageSize, len(stacks))
	end := min(start+inventoryPageSize, len(stacks))

	// Build card data
	v.Items = make([]map[string]any, 0, end-start)
	for _, stack := range stacks[start:end] {
		data, err := GetCardData(c, loc, dc.Renderer, stack.collection, stack.card)
		if err != nil {
			return fmt.Errorf("card %d: %w", stack.CardID, err)
		}
		// Shown in the card popup, formatted client-side
		data["CardCopies"] = stack.Count
		data["CardAcquiredFirst"] = stack.FirstAcquired.Format(time.RFC3339)
		data["CardAcquiredLast"] = stack.LastAcquired.Format(time.RFC3339)
		if committed != nil {
			locked := min(uint(stack.Count), committed.Copies(stack.CardID, stack.Foil))
			if locked > 0 {
				data["CardLocked"] = dc.Renderer.T(c, "inventory-locked-badge", "count", int(locked))
			}
			// The owner's popup sells copies of the stack, except those up in trades
			if price := controllers.SellPrice(stack.card.Rarity, stack.Foil); price > 0 {
				data["CardSellID"] = stack.CardID
				data["CardSellFoil"] = stack.Foil
				data["CardSellPrice"] = price
				data["CardSellOwned"] = uint(stack.Count) - locked
				data["CardSellLocked"] = locked
			}
		}
		setCardFoil(data, stack.Foil)
		v.Items = append(v.Items, map[string]any{
			"Data": data,
		})
	}
	return nil
}

func (dc *DataController) inventoryAllCards(c *gin.Context, userID uint, v *inventoryView) error {
	var owned []cardStack
	if err := dc.DB.
		Model(&models.Card{}).
		Select("card_id, COUNT(*) AS count, MIN(acquired) AS first_acquired, MAX(acquired) AS last_acquired").
		Where("owner_id = ?", userID).
		Group("card_id").
		Scan(&owned).
		Error; err != nil {
		return err
	}
	catalog := &catalogView{
		Query:  v.Query,
		Rarity: v.Rarity,
		Sort:   v.Sort,
		Page:   v.Page,
		Owned:  make(map[uint]bool, len(owned)),
		Copies: make(map[uint]cardStack, len(owned)),
	}
	for _, stack := range owned {
		catalog.Owned[stack.CardID] = true
		catalog.Copies[stack.CardID] = stack
	}
	if err := dc.catalogCardItems(c, catalog); err != nil {
		return err
	}
	v.Items = catalog.Items
	v.Count = int64(catalog.Count)
	v.PageItems = v.Count
	return nil
}

// Boosters of the same collection and size, displayed as one tile
type boosterStack struct {
	CollectionID uint
	Size         uint
	Count        int64
	// Oldest booster of the stack, opened first
	BoosterID uint
}

<<<<<<< HEAD
// Fill the view with the user's boosters, stacked by collection and size.
// On the user's own inventory, collections without boosters are listed too,
// their tile leading to the pack opening page where some are bought.
=======
// On one's own inventory, collections without boosters are listed too, to
// buy some
>>>>>>> master
func (dc *DataController) inventoryBoosters(c *gin.Context, userID uint, own bool, v *inventoryView) error {
	var stacks []boosterStack
	if err := dc.DB.
		Model(&models.Booster{}).
		Select("collection_id, size, COUNT(*) AS count, MIN(id) AS booster_id").
		Where("owner_id = ?", userID).
		Group("collection_id, size").
		Order("collection_id, size").
		Scan(&stacks).
		Error; err != nil {
		return err
	}

	byCollection := make(map[uint][]boosterStack)
	for _, stack := range stacks {
		if dc.Data.GetCollection(int(stack.CollectionID)) == nil {
			log.Printf("WARN user %d owns boosters of unknown collection %d", userID, stack.CollectionID)
			continue
		}
		byCollection[stack.CollectionID] = append(byCollection[stack.CollectionID], stack)
	}

	// The collection's progress shown under each tile
	collected, err := dc.collectedCards(userID)
	if err != nil {
		return err
	}

	loc := dc.Renderer.Localizer.GetLocale(c)
	items := make([]map[string]any, 0, len(stacks))
	for _, collection := range dc.sortedCollections() {
		collectionStacks := byCollection[uint(collection.ID)]
		if len(collectionStacks) == 0 && !own {
			continue
		}
		data, err := getCollectionData(c, loc, dc.Renderer, collection)
		if err != nil {
			return fmt.Errorf("collection %d: %w", collection.ID, err)
		}
		total := len(collection.Cards)
		percent := 0
		if total > 0 {
			percent = collected[collection.ID] * 100 / total
		}
		// None left: the tile leads to the pack opening page, where some
		// are bought
		if len(collectionStacks) == 0 {
			items = append(items, map[string]any{
				"Data":    data,
				"Open":    fmt.Sprintf("/opening?collection=%d", collection.ID),
				"Label":   dc.Renderer.T(c, "inventory-booster-buy", "name", data["CollectionTitle"]),
				"Owned":   collected[collection.ID],
				"Total":   total,
				"Percent": percent,
			})
			continue
		}

		for _, stack := range collectionStacks {
			stackData := maps.Clone(data)
			// The booster keeps the size it was acquired with
			stackData["CollectionDescription"] = dc.Renderer.T(c, "collection-booster", "count", int(stack.Size))
			// Said in the tile's accessible name, no longer displayed
			stackData["CollectionStack"] = stack.Count
			items = append(items, map[string]any{
				"Data":    stackData,
				"Open":    fmt.Sprintf("/opening?booster=%d", stack.BoosterID),
				"Label":   dc.Renderer.T(c, "inventory-booster-open", "name", data["CollectionTitle"]),
				"Owned":   collected[collection.ID],
				"Total":   total,
				"Percent": percent,
			})
		}
	}

	v.PageItems = int64(len(items))
	start := min((v.Page-1)*inventoryPageSize, len(items))
	v.Items = items[start:min(start+inventoryPageSize, len(items))]
	return nil
}

<<<<<<< HEAD
// Number of cards a user owns, shown in the inventory tabs
func (dc *DataController) countCards(userID uint) (int64, error) {
	var count int64
	err := dc.DB.Model(&models.Card{}).Where("owner_id = ?", userID).Count(&count).Error
	return count, err
=======
func (dc *DataController) inventoryCounts(userID uint) (cards int64, boosters int64, err error) {
	if err = dc.DB.Model(&models.Card{}).Where("owner_id = ?", userID).Count(&cards).Error; err != nil {
		return
	}
	err = dc.DB.Model(&models.Booster{}).Where("owner_id = ?", userID).Count(&boosters).Error
	return
>>>>>>> master
}

func (dc *DataController) inventoryCollections(c *gin.Context, userID uint, v *inventoryView) error {
	owned, err := dc.collectedCards(userID)
	if err != nil {
		return err
	}

	loc := dc.Renderer.Localizer.GetLocale(c)
	collections := dc.sortedCollections()
	v.Count = int64(len(collections))
	v.PageItems = v.Count
	start := min((v.Page-1)*inventoryPageSize, len(collections))
	collections = collections[start:min(start+inventoryPageSize, len(collections))]

	v.Items = make([]map[string]any, 0, len(collections))
	for _, collection := range collections {
		data, err := getCollectionData(c, loc, dc.Renderer, collection)
		if err != nil {
			return fmt.Errorf("collection %d: %w", collection.ID, err)
		}
		total := len(collection.Cards)
		percent := 0
		if total > 0 {
			percent = owned[collection.ID] * 100 / total
		}
		collectionURL := fmt.Sprint("/collections/", collection.ID, "/")
		if userID != controllers.GetAuthenticatedUser(c).ID {
			collectionURL += "?" + url.Values{"u": {v.Username}}.Encode()
		}
		v.Items = append(v.Items, map[string]any{
			"Data": data,
			// Lists the collection's cards for the viewed user
			"Url":     collectionURL,
			"Owned":   owned[collection.ID],
			"Total":   total,
			"Percent": percent,
		})
	}
	return nil
}

<<<<<<< HEAD
// Distinct cards owned, by collection id
func (dc *DataController) collectedCards(userID uint) (map[int]int, error) {
	var rows []struct {
		CollectionID int
		Owned        int
	}
	if err := dc.DB.
		Model(&models.Card{}).
		Select("FLOOR(card_id / 1000) AS collection_id, COUNT(DISTINCT card_id) AS owned").
		Where("owner_id = ?", userID).
		Group("FLOOR(card_id / 1000)").
		Scan(&rows).
		Error; err != nil {
		return nil, err
	}
	collected := make(map[int]int, len(rows))
	for _, row := range rows {
		collected[row.CollectionID] = row.Owned
	}
	return collected, nil
}

// Global IDs of the cards a user owns, once each
=======
>>>>>>> master
func (dc *DataController) ownedCardIDs(userID uint) ([]uint, error) {
	var owned []uint
	err := dc.DB.
		Model(&models.Card{}).
		Where("owner_id = ?", userID).
		Distinct("card_id").
		Pluck("card_id", &owned).
		Error
	return owned, err
}

func (dc *DataController) searchInventory(c *gin.Context, userID uint, query string) ([]uint, error) {
	owned, err := dc.ownedCardIDs(userID)
	if err != nil {
		return nil, err
	}

	ss, err := dc.GetCardSearchSet(dc.Renderer.Localizer.GetLocale(c))
	if err != nil {
		return nil, err
	}
	return ss.SearchWithin(query, owned)
}
