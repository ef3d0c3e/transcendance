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
	"transcendance/views"

	"github.com/gin-gonic/gin"
)

const (
	catalogTabCards       = "cards"
	catalogTabCollections = "collections"
)

// Catalog card sort keeping the collection order, used when not searching
const catalogSortNumber = "number"

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
	// Displayed tab, cards or collections
	Tab   string
	Query string
	// Card filters, "" or 0 when unfiltered
	Rarity     string
	Collection int
	// Card sort, always set on the cards tab
	Sort string
	// Error or empty state message
	Message string
	// Rendered cards or collections
	Items []map[string]any
	Page  int
	// Number of matching cards or collections
	Count int
	// Collection whose cards are listed, nil on the catalog
	Scope *data.Collection
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
	searching := strings.TrimSpace(v.Query) != "" ||
		(v.Tab == catalogTabCards && (v.Rarity != "" || (v.Collection != 0 && v.Scope == nil)))
	if v.Message == "" && len(v.Items) == 0 {
		if v.Tab == catalogTabCollections {
			v.Message = dc.Renderer.T(c, "collection-search-empty")
		} else {
			v.Message = dc.Renderer.T(c, "card-search-empty")
		}
	}
	count := ""
	if v.Count > 0 {
		switch {
		case v.Tab == catalogTabCollections:
			count = dc.Renderer.T(c, "catalog-collection-count", "count", v.Count)
		case searching:
			count = dc.Renderer.T(c, "inventory-search-count", "count", v.Count)
		default:
			count = dc.Renderer.T(c, "inventory-count", "count", v.Count)
		}
	}
	tabBase := map[string]string{
		catalogTabCards:       "/cards/search/",
		catalogTabCollections: "/collections/search/",
	}
	// The search is kept when switching tabs
	tabUrl := func(tab string) string {
		if v.Query == "" {
			return tabBase[tab]
		}
		return tabBase[tab] + "?" + url.Values{"q": {v.Query}}.Encode()
	}
	if v.Scope != nil {
		tabBase[catalogTabCards] = fmt.Sprint("/collections/", v.Scope.ID, "/")
	}
	totalPages := (v.Count + pageSize - 1) / pageSize
	fields := map[string]any{
		"Title":             title,
		"Message":           v.Message,
		"Count":             count,
		"Tab":               v.Tab,
		"Action":            tabBase[v.Tab],
		"TabCardsUrl":       tabUrl(catalogTabCards),
		"TabCollectionsUrl": tabUrl(catalogTabCollections),
		"Items":             v.Items,
		"Query":             v.Query,
		"Filtered":          searching,
		"ClearUrl":          tabBase[v.Tab],
		"Pages":             totalPages,
		"Scoped":            v.Scope != nil,
	}
	if v.Tab == catalogTabCards {
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
		defaultSort := defaultCatalogSort(v.Query)
		sorts := catalogSorts
		if defaultSort != inventorySortRelevance {
			sorts = sorts[1:]
		}
		sortOptions := controllers.SelectOptions(sorts, v.Sort, func(value string) string {
			if value == catalogSortNumber {
				return dc.Renderer.T(c, "catalog-sort-number")
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
	if v.Tab == catalogTabCards && v.Sort != defaultCatalogSort(v.Query) {
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
		Tab:   catalogTabCards,
		Query: c.Query("q"),
		Page:  catalogPage(c),
		Scope: scope,
	}
	if rarity := c.Query("rarity"); slices.Contains(inventoryRarities, rarity) {
		v.Rarity = rarity
	}
	if scope != nil {
		v.Collection = scope.ID
	} else if id, err := strconv.Atoi(c.Query("collection")); err == nil && dc.Data.GetCollection(id) != nil {
		v.Collection = id
	}
	v.Sort = defaultCatalogSort(v.Query)
	if sort := c.Query("sort"); slices.Contains(catalogSorts, sort) &&
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
			(v.Collection != 0 && stack.collection.ID != v.Collection)
	})
	v.Count = len(stacks)
	// The collection order is the ID tie-break
	sortCardStacks(stacks, v.Sort, loc, relevance)

	start := min((v.Page-1)*pageSize, len(stacks))
	end := min(start+pageSize, len(stacks))
	v.Items = make([]map[string]any, 0, end-start)
	for _, stack := range stacks[start:end] {
		data, err := GetCardData(c, loc, dc.Renderer, stack.collection, stack.card)
		if err != nil {
			return fmt.Errorf("card %d: %w", stack.CardID, err)
		}
		v.Items = append(v.Items, map[string]any{
			"Data": data,
		})
	}
	return nil
}

// collections/search: every collection, or the ones matching the search
func (dc *DataController) catalogCollections(c *gin.Context) {
	v := &catalogView{
		Tab:   catalogTabCollections,
		Query: c.Query("q"),
		Page:  catalogPage(c),
	}
	loc := dc.Renderer.Localizer.GetLocale(c)

	var collections []*data.Collection
	if strings.TrimSpace(v.Query) != "" {
		ss, err := dc.GetCollectionSearchSet(loc)
		if err != nil {
			log.Printf("WARN failed to get collection search set for locale '%s': %s", loc, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": dc.Renderer.T(c, "collection-search-error-internal"),
			})
			return
		}
		results, total, err := ss.Search(v.Query, pageSize, (v.Page-1)*pageSize)
		if err != nil {
			log.Printf("WARN search failed for locale: '%s', term '%s': %s", loc, v.Query, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": dc.Renderer.T(c, "collection-search-error-internal"),
			})
			return
		}
		collections = results
		v.Count = int(total)
	} else {
		all := dc.sortedCollections()
		v.Count = len(all)
		start := min((v.Page-1)*pageSize, len(all))
		collections = all[start:min(start+pageSize, len(all))]
	}

	v.Items = make([]map[string]any, 0, len(collections))
	for _, collection := range collections {
		data, err := getCollectionData(c, loc, dc.Renderer, collection)
		if err != nil {
			log.Printf("WARN failed to build data for collection '%s': %s", collection.Name, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": dc.Renderer.T(c, "collection-search-error-internal"),
			})
			return
		}
		v.Items = append(v.Items, map[string]any{
			"Data": data,
			// Lists the collection's cards
			"Url": fmt.Sprint("/collections/", collection.ID, "/"),
		})
	}
	dc.renderCatalog(c, v)
}
