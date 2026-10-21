package data

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"transcendance/controllers"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// Everything needed to render the inventory page
type inventoryView struct {
	// Displayed tab, cards or boosters
	Tab string
	// Viewed user
	Username string
	Query    string
	// Error or empty state message
	Message string
	// Rendered cards or boosters
	Items []map[string]any
	Page  int
	// Number of items to paginate (cards, or booster stacks)
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
	searching := v.Tab == inventoryTabCards && strings.TrimSpace(v.Query) != ""
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
		if tab != inventoryTabCards {
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
	content := builder.Add("inventory", "Content", map[string]any{
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
		"Viewed":         viewed,
		"Pages":          int(totalPages),
		"Locale":         dc.Renderer.Localizer.GetLocale(c),
	})
	params := url.Values{}
	if viewed != "" {
		params.Set("u", viewed)
	}
	if v.Tab != inventoryTabCards {
		params.Set("tab", v.Tab)
	}
	if v.Query != "" {
		params.Set("q", v.Query)
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
		Tab:      inventoryTabCards,
		Username: c.Query("u"),
		Query:    c.Query("q"),
		Page:     page,
	}
	if c.Query("tab") == inventoryTabBoosters {
		v.Tab = inventoryTabBoosters
		// Boosters can't be searched
		v.Query = ""
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
		if err := dc.DB.Where("username = ?", v.Username).First(&otherUser).Error; err != nil {
			v.Message = dc.Renderer.T(c, "inventory-message-user-not-found")
			dc.renderInventory(c, user, v)
			return
		}
		userID = otherUser.ID
	}

	fail := func(what string, err error) {
		log.Printf("WARN failed to get %s for user '%s': %s", what, v.Username, err)
		*v = inventoryView{Tab: v.Tab, Username: v.Username, Query: v.Query, Page: 1}
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
		if err := dc.inventoryBoosters(c, userID, v); err != nil {
			fail("boosters", err)
			return
		}
	} else if err := dc.inventoryCards(c, userID, v); err != nil {
		fail("cards", err)
		return
	}

	dc.renderInventory(c, user, v)
}

// Fill the view with the user's cards, restricted to the search if any
func (dc *DataController) inventoryCards(c *gin.Context, userID uint, v *inventoryView) error {
	cardsQuery := dc.DB.Model(&models.Card{}).Where("owner_id = ?", userID)
	if strings.TrimSpace(v.Query) != "" {
		matches, err := dc.searchInventory(c, userID, v.Query)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return nil
		}
		cardsQuery = cardsQuery.
			Where("card_id IN ?", matches).
			Order(clause.Expr{SQL: "FIELD(card_id, ?)", Vars: []any{matches}, WithoutParentheses: true}).
			Order("acquired DESC")
	}

	if err := cardsQuery.
		Session(&gorm.Session{}).
		Count(&v.PageItems).
		Error; err != nil {
		return err
	}
	v.Count = v.PageItems

	var cards []models.Card
	if err := cardsQuery.
		Session(&gorm.Session{}).
		Offset((v.Page - 1) * inventoryPageSize).
		Limit(inventoryPageSize).
		Find(&cards).
		Error; err != nil {
		return err
	}

	// Build card data
	loc := dc.Renderer.Localizer.GetLocale(c)
	v.Items = make([]map[string]any, len(cards))
	for i, invCard := range cards {
		card, collection := dc.Data.GetCard(int(invCard.CardID))
		data, err := GetCardData(c, loc, dc.Renderer, collection, card)
		if err != nil {
			return fmt.Errorf("card %d: %w", invCard.CardID, err)
		}
		// Shown in the card popup, formatted client-side
		data["CardAcquired"] = invCard.Acquired.Format(time.RFC3339)
		v.Items[i] = map[string]any{
			"Data": data,
		}
	}
	return nil
}

// Boosters of the same collection and size, displayed as one tile
type boosterStack struct {
	CollectionID uint
	Size         uint
	Count        int64
}

// Fill the view with the user's boosters, stacked by collection and size
func (dc *DataController) inventoryBoosters(c *gin.Context, userID uint, v *inventoryView) error {
	stacksQuery := dc.DB.
		Model(&models.Booster{}).
		Select("collection_id, size, COUNT(*) AS count").
		Where("owner_id = ?", userID).
		Group("collection_id, size")

	if err := dc.DB.
		Table("(?) AS stacks", stacksQuery).
		Count(&v.PageItems).
		Error; err != nil {
		return err
	}
	v.Count = v.TotalBoosters

	var stacks []boosterStack
	if err := stacksQuery.
		Order("collection_id, size").
		Offset((v.Page - 1) * inventoryPageSize).
		Limit(inventoryPageSize).
		Scan(&stacks).
		Error; err != nil {
		return err
	}

	loc := dc.Renderer.Localizer.GetLocale(c)
	v.Items = make([]map[string]any, 0, len(stacks))
	for _, stack := range stacks {
		collection := dc.Data.GetCollection(int(stack.CollectionID))
		if collection == nil {
			log.Printf("WARN user %d owns boosters of unknown collection %d", userID, stack.CollectionID)
			continue
		}
		data, err := getCollectionData(c, loc, dc.Renderer, collection)
		if err != nil {
			return fmt.Errorf("collection %d: %w", collection.ID, err)
		}
		// The booster keeps the size it was acquired with
		data["CollectionDescription"] = dc.Renderer.T(c, "collection-booster", "count", int(stack.Size))
		data["CollectionStack"] = stack.Count
		v.Items = append(v.Items, map[string]any{
			"Data": data,
		})
	}
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
