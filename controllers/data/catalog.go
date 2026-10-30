package data

import (
	"cmp"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"transcendance/controllers"
	"transcendance/data"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
)

// Catalog card sort keeping the collection order, used when not searching
const catalogSortNumber = "number"

// In a collection, cards the user is missing first, then the collection order
const catalogSortMissing = "missing"

// In a collection, filter on the cards the user collected or is missing
const (
	catalogStatusOwned   = "owned"
	catalogStatusMissing = "missing"
)

var catalogStatuses = []string{catalogStatusOwned, catalogStatusMissing}

// Card sorts of the catalog, relevance is only available when searching
var catalogSorts = []string{
	inventorySortRelevance,
	catalogSortNumber,
	inventorySortName,
	inventorySortNameDesc,
	inventorySortRarity,
}

// Sort used when none is picked: search relevance, or collection order
func defaultCatalogSort(query string) string {
	if strings.TrimSpace(query) != "" {
		return inventorySortRelevance
	}
	return catalogSortNumber
}

// Everything needed to render the catalog page
type catalogView struct {
	Query string
	// Card filters, "" or 0 when unfiltered
	Rarity     string
	Collection int
	// Collected or missing cards only, within a collection when logged in
	Status string
	// Card sort, always set
	Sort string
	// Error or empty state message
	Message string
	// Rendered cards
	Items []map[string]any
	Page  int
	// Number of matching cards
	Count int
	// Collection whose cards are listed, nil on the catalog
	Scope *data.Collection
	// In a collection, the cards of it the logged in user owns, nil otherwise
	Owned map[uint]bool
}

// Every collection, in ID order
func (dc *DataController) sortedCollections() []*data.Collection {
	collections := make([]*data.Collection, 0, len(dc.Data.Collections))
	for _, collection := range dc.Data.Collections {
		collections = append(collections, collection)
	}
	slices.SortFunc(collections, func(a, b *data.Collection) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return collections
}

func (dc *DataController) renderCatalog(c *gin.Context, v *catalogView) {
	title := dc.Renderer.T(c, "catalog-title")
	if v.Scope != nil {
		title = dc.Renderer.T(c, "collection-title", "collection", dc.collectionTitle(c, v.Scope))
	}
	builder := views.PageBuilder("base", map[string]any{
		"Title": title,
		"User":  controllers.GetAuthenticatedUser(c),
	})
	if v.Items == nil {
		v.Items = make([]map[string]any, 0)
	}
	searching := strings.TrimSpace(v.Query) != "" || v.Rarity != "" || v.Status != "" ||
		(v.Collection != 0 && v.Scope == nil)
	if v.Message == "" && len(v.Items) == 0 {
		v.Message = dc.Renderer.T(c, "card-search-empty")
	}
	count := ""
	if v.Count > 0 {
		if searching {
			count = dc.Renderer.T(c, "inventory-search-count", "count", v.Count)
		} else {
			count = dc.Renderer.T(c, "inventory-count", "count", v.Count)
		}
	}
	// The page lists every card, or a collection's
	action := "/cards/search/"
	if v.Scope != nil {
		action = fmt.Sprint("/collections/", v.Scope.ID, "/")
	}
	totalPages := (v.Count + pageSize - 1) / pageSize
	fields := map[string]any{
		"Title":    title,
		"Message":  v.Message,
		"Count":    count,
		"Action":   action,
		"Items":    v.Items,
		"Query":    v.Query,
		"Filtered": searching,
		"ClearUrl": action,
		"Pages":    totalPages,
		"Scoped":   v.Scope != nil,
	}
	if v.Scope != nil && v.Owned != nil && len(v.Scope.Cards) > 0 {
		fields["Progress"] = map[string]any{
			"Owned":   len(v.Owned),
			"Total":   len(v.Scope.Cards),
			"Percent": len(v.Owned) * 100 / len(v.Scope.Cards),
		}
	}
	{
		fields["RarityOptions"] = controllers.SelectOptions(append([]string{""}, inventoryRarities...), v.Rarity, func(value string) string {
			if value == "" {
				return dc.Renderer.T(c, "inventory-filter-rarity-all")
			}
			return dc.Renderer.T(c, "card-rarity-"+value)
		})
		collections := []string{""}
		titles := map[string]string{"": dc.Renderer.T(c, "catalog-filter-collection-all")}
		for _, collection := range dc.sortedCollections() {
			id := strconv.Itoa(collection.ID)
			collections = append(collections, id)
			titles[id] = dc.collectionTitle(c, collection)
		}
		current := ""
		if v.Collection != 0 {
			current = strconv.Itoa(v.Collection)
		}
		fields["CollectionOptions"] = controllers.SelectOptions(collections, current, func(value string) string {
			return titles[value]
		})
		if v.Owned != nil {
			fields["StatusOptions"] = controllers.SelectOptions(append([]string{""}, catalogStatuses...), v.Status, func(value string) string {
				if value == "" {
					return dc.Renderer.T(c, "catalog-filter-status-all")
				}
				return dc.Renderer.T(c, "catalog-filter-status-"+value)
			})
		}
		defaultSort := defaultCatalogSort(v.Query)
		sorts := slices.Clone(catalogSorts)
		if defaultSort != inventorySortRelevance {
			sorts = sorts[1:]
		}
		if v.Owned != nil {
			sorts = append(sorts, catalogSortMissing)
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
	content := builder.Add("catalog", "Content", fields)
	params := url.Values{}
	if v.Query != "" {
		params.Set("q", v.Query)
	}
	if v.Rarity != "" {
		params.Set("rarity", v.Rarity)
	}
	if v.Collection != 0 && v.Scope == nil {
		params.Set("collection", strconv.Itoa(v.Collection))
	}
	if v.Status != "" {
		params.Set("status", v.Status)
	}
	if v.Sort != defaultCatalogSort(v.Query) {
		params.Set("sort", v.Sort)
	}
	paginatorUrl := "?p="
	if len(params) != 0 {
		paginatorUrl = "?" + params.Encode() + "&p="
	}
	content.Add("paginator", "Paginator", map[string]any{
		"Current": v.Page,
		"Total":   totalPages,
		"Url":     paginatorUrl,
	})
	dc.Renderer.Render(c, &builder)
}

// Global IDs of the cards of a collection owned by a user
func (dc *DataController) ownedCards(userID uint, collection *data.Collection) (map[uint]bool, error) {
	var ids []uint
	if err := dc.DB.
		Model(&models.Card{}).
		Where("owner_id = ? AND card_id > ? AND card_id < ?", userID, collection.ID*1000, (collection.ID+1)*1000).
		Distinct("card_id").
		Pluck("card_id", &ids).
		Error; err != nil {
		return nil, err
	}
	owned := make(map[uint]bool, len(ids))
	for _, id := range ids {
		owned[id] = true
	}
	return owned, nil
}

// Translated title of a collection, its name when missing
func (dc *DataController) collectionTitle(c *gin.Context, collection *data.Collection) string {
	title, err := collection.Translate(dc.Renderer.Localizer.GetLocale(c), "title")
	if err != nil {
		log.Printf("WARN collection %d has no title: %s", collection.ID, err)
		return collection.Name
	}
	return title
}

// Page number from the query, 1 when missing or invalid
func catalogPage(c *gin.Context) int {
	page, err := strconv.Atoi(c.Query("p"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

// cards/search: every card, or the ones matching the search and filters
func (dc *DataController) catalogCards(c *gin.Context) {
	dc.searchCards(c, nil)
}

// collections/:id: the collection's cards, with the catalog search and filters
func (dc *DataController) collectionCards(c *gin.Context, collection *data.Collection) {
	dc.searchCards(c, collection)
}

// Card search page, within scope when it isn't nil
func (dc *DataController) searchCards(c *gin.Context, scope *data.Collection) {
	v := &catalogView{
		Query: c.Query("q"),
		Page:  catalogPage(c),
		Scope: scope,
	}
	if rarity := c.Query("rarity"); slices.Contains(inventoryRarities, rarity) {
		v.Rarity = rarity
	}
	if scope != nil {
		v.Collection = scope.ID
		// Logged in, the cards not collected yet are grayed out
		if user := controllers.GetAuthenticatedUser(c); user != nil {
			owned, err := dc.ownedCards(user.ID, scope)
			if err != nil {
				log.Printf("WARN failed to get the cards of collection %d owned by '%s': %s", scope.ID, user.Username, err)
			} else {
				v.Owned = owned
			}
		}
	} else if id, err := strconv.Atoi(c.Query("collection")); err == nil && dc.Data.GetCollection(id) != nil {
		v.Collection = id
	}
	// What the user owns is only known inside a collection, logged in
	if status := c.Query("status"); v.Owned != nil && slices.Contains(catalogStatuses, status) {
		v.Status = status
	}
	v.Sort = defaultCatalogSort(v.Query)
	if sort := c.Query("sort"); (slices.Contains(catalogSorts, sort) || (sort == catalogSortMissing && v.Owned != nil)) &&
		(sort != inventorySortRelevance || v.Sort == inventorySortRelevance) {
		v.Sort = sort
	}

	if err := dc.catalogCardItems(c, v); err != nil {
		log.Printf("WARN card catalog failed for term '%s': %s", v.Query, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": dc.Renderer.T(c, "card-search-error-internal"),
		})
		return
	}
	dc.renderCatalog(c, v)
}

// Fill the view with the matching cards, sorted and paginated
func (dc *DataController) catalogCardItems(c *gin.Context, v *catalogView) error {
	loc := dc.Renderer.Localizer.GetLocale(c)

	// Cards are folded like inventory stacks to share the inventory sorts
	var stacks []cardStack
	var relevance map[uint]int
	if strings.TrimSpace(v.Query) != "" {
		ss, err := dc.GetCardSearchSet(loc)
		if err != nil {
			return err
		}
		pairs, _, err := ss.Search(v.Query, len(ss.byID), 0)
		if err != nil {
			return err
		}
		relevance = make(map[uint]int, len(pairs))
		for i, pair := range pairs {
			id := uint(pair.collection.ID*1000 + pair.card.ID)
			relevance[id] = i
			stacks = append(stacks, cardStack{CardID: id, card: pair.card, collection: pair.collection})
		}
	} else {
		for _, collection := range dc.sortedCollections() {
			for _, colCard := range collection.Cards {
				card := dc.Data.Cards[colCard.Name]
				if card == nil {
					continue
				}
				id := uint(collection.ID*1000 + card.ID)
				stacks = append(stacks, cardStack{CardID: id, card: card, collection: collection})
			}
		}
	}

	stacks = slices.DeleteFunc(stacks, func(stack cardStack) bool {
		return (v.Rarity != "" && stack.card.Rarity != v.Rarity) ||
			(v.Collection != 0 && stack.collection.ID != v.Collection) ||
			(v.Status == catalogStatusOwned && !v.Owned[stack.CardID]) ||
			(v.Status == catalogStatusMissing && v.Owned[stack.CardID])
	})
	v.Count = len(stacks)
	// The collection order is the ID tie-break
	if v.Sort == catalogSortMissing {
		sortCardStacks(stacks, catalogSortNumber, loc, relevance)
		slices.SortStableFunc(stacks, func(a, b cardStack) int {
			// Missing (false) before owned
			return cmp.Compare(boolRank(v.Owned[a.CardID]), boolRank(v.Owned[b.CardID]))
		})
	} else {
		sortCardStacks(stacks, v.Sort, loc, relevance)
	}

	start := min((v.Page-1)*pageSize, len(stacks))
	end := min(start+pageSize, len(stacks))
	v.Items = make([]map[string]any, 0, end-start)
	for _, stack := range stacks[start:end] {
		data, err := GetCardData(c, loc, dc.Renderer, stack.collection, stack.card)
		if err != nil {
			return fmt.Errorf("card %d: %w", stack.CardID, err)
		}
		if v.Owned != nil {
			data["CardMissing"] = !v.Owned[stack.CardID]
		}
		v.Items = append(v.Items, map[string]any{
			"Data": data,
		})
	}
	return nil
}

// 0 for false, 1 for true, to sort on a flag
func boolRank(b bool) int {
	if b {
		return 1
	}
	return 0
}
