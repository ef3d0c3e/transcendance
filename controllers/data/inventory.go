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

func (dc *DataController) InventoryAddGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)

	if user == nil {
		c.String(http.StatusForbidden, "You must be authenticated")
		return
	}

	cardId, err := strconv.Atoi(c.Query("id"))
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
}

const (
	inventoryTabCards    = "cards"
	inventoryTabBoosters = "boosters"
)

// Effect filter values, "" shows every card
const (
	inventoryEffectNone = "none"
	inventoryEffectFoil = "foil"
)

var inventoryEffects = []string{inventoryEffectNone, inventoryEffectFoil}

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

// Rarities offered by the rarity filter, lowest first
var inventoryRarities = []string{"common", "rare", "legendary"}

// Sort used when none is picked: search relevance, or newest cards
func defaultInventorySort(query string) string {
	if strings.TrimSpace(query) != "" {
		return inventorySortRelevance
	}
	return inventorySortRecent
}

// Everything needed to render the inventory page
type inventoryView struct {
	// Displayed tab, cards or boosters
	Tab string
	// Viewed user
	Username string
	Query    string
	// Card filters, "" when unfiltered
	Rarity string
	Effect string
	// Card sort, always set on the cards tab
	Sort string
	// Error or empty state message
	Message string
	// Rendered cards or boosters
	Items []map[string]any
	Page  int
	// Number of items to paginate (card stacks, or booster stacks)
	PageItems int64
	// Count shown next to the title (cards, or boosters)
	Count int64
	// Count shown in each tab
	TotalCards    int64
	TotalBoosters int64
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
		case searching:
			v.Message = dc.Renderer.T(c, "inventory-message-no-match")
		case v.Tab == inventoryTabBoosters:
			v.Message = dc.Renderer.T(c, "inventory-message-no-boosters")
		default:
			v.Message = dc.Renderer.T(c, "inventory-message-empty")
		}
	}
	// Number of items in the tab, or matching the search
	count := ""
	if v.Count > 0 {
		switch {
		case searching:
			count = dc.Renderer.T(c, "inventory-search-count", "count", int(v.Count))
		case v.Tab == inventoryTabBoosters:
			count = dc.Renderer.T(c, "inventory-booster-count", "count", int(v.Count))
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
	totalPages := v.PageItems / int64(inventoryPageSize)
	if v.PageItems%int64(inventoryPageSize) != 0 {
		totalPages = totalPages + 1
	}
	fields := map[string]any{
		"Title":          title,
		"Message":        v.Message,
		"Count":          count,
		"Tab":            v.Tab,
		"TabCardsUrl":    tabUrl(inventoryTabCards),
		"TabBoostersUrl": tabUrl(inventoryTabBoosters),
		"TotalCards":     v.TotalCards,
		"TotalBoosters":  v.TotalBoosters,
		"Items":          v.Items,
		"Query":          v.Query,
		"Filtered":       searching,
		"Viewed":         viewed,
		"Pages":          int(totalPages),
		"Locale":         dc.Renderer.Localizer.GetLocale(c),
		// Outcome of buying boosters or coins, which land here
		"Notice": controllers.GetShopNotice(c, dc.Renderer, dc.Data),
	}
	// Boosters and coins are bought from the user's own boosters tab
	if v.Tab == inventoryTabBoosters && viewed == "" {
		fields["Coins"] = user.Coins
		fields["CoinsLabel"] = dc.Renderer.T(c, "shop-price", "amount", user.Coins)
		fields["MaxQuantity"] = controllers.MaxBuyQuantity
		fields["Topups"] = controllers.TopupChoices(c, dc.Renderer)
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
		defaultSort := defaultInventorySort(v.Query)
		sorts := inventorySorts
		if defaultSort != inventorySortRelevance {
			sorts = sorts[1:]
		}
		sortOptions := controllers.SelectOptions(sorts, v.Sort, func(value string) string {
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
	content := builder.Add("inventory", "Content", fields)
	params := url.Values{}
	if viewed != "" {
		params.Set("u", viewed)
	}
	if v.Tab != inventoryTabBoosters {
		params.Set("tab", v.Tab)
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
	if v.Tab == inventoryTabCards && v.Sort != defaultInventorySort(v.Query) {
		params.Set("sort", v.Sort)
	}
	paginatorUrl := "?p="
	if len(params) != 0 {
		paginatorUrl = "?" + params.Encode() + "&p="
	}
	content.Add("paginator", "Paginator", map[string]any{
		"Current": v.Page,
		"Total":   int(totalPages),
		"Url":     paginatorUrl,
	})
	dc.Renderer.Render(c, &builder)
}

func (dc *DataController) InventoryGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)

	if user == nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	page, err := strconv.Atoi(c.Query("p"))
	if err != nil || page <= 0 {
		page = 1
	}

	v := &inventoryView{
		Tab:      inventoryTabBoosters,
		Username: c.Query("u"),
		Page:     page,
	}
	// Boosters are shown unless the cards are asked for, or their search,
	// filters or sort are
	cardsAsked := c.Query("tab") == inventoryTabCards
	if c.Query("tab") == "" {
		for _, param := range []string{"q", "rarity", "effect", "sort"} {
			cardsAsked = cardsAsked || c.Query(param) != ""
		}
	}
	if cardsAsked {
		v.Tab = inventoryTabCards
		v.Query = c.Query("q")
		if rarity := c.Query("rarity"); slices.Contains(inventoryRarities, rarity) {
			v.Rarity = rarity
		}
		if effect := c.Query("effect"); slices.Contains(inventoryEffects, effect) {
			v.Effect = effect
		}
		v.Sort = defaultInventorySort(v.Query)
		if sort := c.Query("sort"); slices.Contains(inventorySorts, sort) &&
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
		// Fetch
		var otherUser models.User
		if err := dc.DB.Omit("avatar").Where("username = ?", v.Username).First(&otherUser).Error; err != nil {
			v.Message = dc.Renderer.T(c, "inventory-message-user-not-found")
			dc.renderInventory(c, user, v)
			return
		}
		userID = otherUser.ID
	}

	fail := func(what string, err error) {
		log.Printf("WARN failed to get %s for user '%s': %s", what, v.Username, err)
		*v = inventoryView{Tab: v.Tab, Username: v.Username, Query: v.Query, Rarity: v.Rarity, Effect: v.Effect, Sort: v.Sort, Page: 1}
		v.Message = dc.Renderer.T(c, "inventory-message-fetch-fail")
		dc.renderInventory(c, user, v)
	}

	// Tab counts
	if err := dc.DB.Model(&models.Card{}).Where("owner_id = ?", userID).Count(&v.TotalCards).Error; err != nil {
		fail("cards count", err)
		return
	}
	if err := dc.DB.Model(&models.Booster{}).Where("owner_id = ?", userID).Count(&v.TotalBoosters).Error; err != nil {
		fail("boosters count", err)
		return
	}

	if v.Tab == inventoryTabBoosters {
		if err := dc.inventoryBoosters(c, userID, userID == user.ID, v); err != nil {
			fail("boosters", err)
			return
		}
	} else if err := dc.inventoryCards(c, userID, v); err != nil {
		fail("cards", err)
		return
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

// Order card stacks with the given sort, newest stacks first on ties
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

// Fill the view with the user's cards folded by card and foil, restricted to the search and filters if any
func (dc *DataController) inventoryCards(c *gin.Context, userID uint, v *inventoryView) error {
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
		setCardFoil(data, stack.Foil)
		v.Items = append(v.Items, map[string]any{
			"Data": data,
		})
	}
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

// Fill the view with the user's boosters, stacked by collection and size.
// On the user's own inventory, collections without boosters are listed too,
// to buy some.
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
	v.Count = v.TotalBoosters

	byCollection := make(map[uint][]boosterStack)
	for _, stack := range stacks {
		if dc.Data.GetCollection(int(stack.CollectionID)) == nil {
			log.Printf("WARN user %d owns boosters of unknown collection %d", userID, stack.CollectionID)
			continue
		}
		byCollection[stack.CollectionID] = append(byCollection[stack.CollectionID], stack)
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

		// None left: a grayed tile selling the shop's booster
		if len(collectionStacks) == 0 {
			data["CollectionEmpty"] = true
			price := controllers.BoosterPrice(collection.ID)
			items = append(items, map[string]any{
				"Data":         data,
				"Empty":        true,
				"CollectionID": collection.ID,
				"Title":        data["CollectionTitle"],
				"Price":        price,
				"PriceLabel":   dc.Renderer.T(c, "shop-price", "amount", price),
			})
			continue
		}

		for _, stack := range collectionStacks {
			stackData := maps.Clone(data)
			// The booster keeps the size it was acquired with
			stackData["CollectionDescription"] = dc.Renderer.T(c, "collection-booster", "count", int(stack.Size))
			stackData["CollectionStack"] = stack.Count
			items = append(items, map[string]any{
				"Data":      stackData,
				"BoosterID": stack.BoosterID,
				"Label":     dc.Renderer.T(c, "inventory-booster-open", "name", data["CollectionTitle"]),
			})
		}
	}

	v.PageItems = int64(len(items))
	start := min((v.Page-1)*inventoryPageSize, len(items))
	v.Items = items[start:min(start+inventoryPageSize, len(items))]
	return nil
}

// Get the global IDs of cards owned by a user that match a search
func (dc *DataController) searchInventory(c *gin.Context, userID uint, query string) ([]uint, error) {
	var owned []uint
	if err := dc.DB.
		Model(&models.Card{}).
		Where("owner_id = ?", userID).
		Distinct("card_id").
		Pluck("card_id", &owned).
		Error; err != nil {
		return nil, err
	}

	ss, err := dc.GetCardSearchSet(dc.Renderer.Localizer.GetLocale(c))
	if err != nil {
		return nil, err
	}
	return ss.SearchWithin(query, owned)
}
