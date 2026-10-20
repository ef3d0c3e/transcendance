package data

import (
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

func (dc *DataController) renderInventory(c *gin.Context, user *controllers.AuthenticatedUser, cards []map[string]any, page int, total int64, username string, query string, message string) {
	var title string
	if user.Username == username {
		title = dc.Renderer.T(c, "inventory-title")
	} else {
		title = dc.Renderer.T(c, "inventory-title-other", "username", username)
	}
	builder := views.PageBuilder("base", map[string]any{
		"Title": title,
		"User":  user,
	})
	if cards == nil {
		cards = make([]map[string]any, 0)
	}
	if message == "" && len(cards) == 0 {
		if strings.TrimSpace(query) != "" {
			message = dc.Renderer.T(c, "inventory-message-no-match")
		} else {
			message = dc.Renderer.T(c, "inventory-message-empty")
		}
	}
	// Number of cards in the inventory, or matching the search
	count := ""
	if total > 0 {
		if strings.TrimSpace(query) != "" {
			count = dc.Renderer.T(c, "inventory-search-count", "count", int(total))
		} else {
			count = dc.Renderer.T(c, "inventory-count", "count", int(total))
		}
	}
	// Keep the viewed user in the search form and paginator links
	viewed := ""
	if user.Username != username {
		viewed = username
	}
	content := builder.Add("inventory", "Content", map[string]any{
		"Title":   title,
		"Message": message,
		"Count":   count,
		"Cards":   cards,
		"Query":   query,
		"Viewed":  viewed,
	})
	params := url.Values{}
	if viewed != "" {
		params.Set("u", viewed)
	}
	if query != "" {
		params.Set("q", query)
	}
	paginatorUrl := "?p="
	if len(params) != 0 {
		paginatorUrl = "?" + params.Encode() + "&p="
	}
	totalPages := total / int64(inventoryPageSize)
	if total%int64(inventoryPageSize) != 0 {
		totalPages = totalPages + 1
	}
	content.Add("paginator", "Paginator", map[string]any{
		"Current": page,
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

	query := c.Query("q")
	username := c.Query("u")
	if username == "" {
		username = user.Username
	}
	userID := user.ID
	if strings.EqualFold(username, user.Username) {
		username = user.Username
	} else {
		// Fetch
		var otherUser models.User
		if err := dc.DB.Where("username = ?", username).First(&otherUser).Error; err != nil {
			dc.renderInventory(c, user, nil, 1, 0, username, query, dc.Renderer.T(c, "inventory-message-user-not-found"))
			return
		}
		userID = otherUser.ID
	}

	cardsQuery := dc.DB.Model(&models.Card{}).Where("owner_id = ?", userID)
	if strings.TrimSpace(query) != "" {
		matches, err := dc.searchInventory(c, userID, query)
		if err != nil {
			log.Printf("WARN failed to search cards for user '%s': %s", username, err)
			dc.renderInventory(c, user, nil, 1, 0, username, query, dc.Renderer.T(c, "inventory-message-fetch-fail"))
			return
		}
		if len(matches) == 0 {
			dc.renderInventory(c, user, nil, 1, 0, username, query, "")
			return
		}
		cardsQuery = cardsQuery.
			Where("card_id IN ?", matches).
			Order(clause.Expr{SQL: "FIELD(card_id, ?)", Vars: []any{matches}, WithoutParentheses: true}).
			Order("acquired DESC")
	}

	var total int64
	if err := cardsQuery.
		Session(&gorm.Session{}).
		Count(&total).
		Error; err != nil {
		log.Printf("WARN failed to get cards count for user '%s': %s", username, err)
		dc.renderInventory(c, user, nil, 1, 0, username, query, dc.Renderer.T(c, "inventory-message-fetch-fail"))
		return
	}

	var cards []models.Card
	if err := cardsQuery.
		Session(&gorm.Session{}).
		Offset((page - 1) * inventoryPageSize).
		Limit(inventoryPageSize).
		Find(&cards).
		Error; err != nil {
		log.Printf("WARN failed to get cards for user '%s': %s", username, err)
		dc.renderInventory(c, user, nil, 1, 0, username, query, dc.Renderer.T(c, "inventory-message-fetch-fail"))
		return
	}

	// Build card data
	loc := dc.Renderer.Localizer.GetLocale(c)
	cardData := make([]map[string]any, len(cards))
	for i, invCard := range cards {
		card, collection := dc.Data.GetCard(int(invCard.CardID))
		data, err := GetCardData(c, loc, dc.Renderer, collection, card)
		if err != nil {
			log.Printf("WARN failed to get card data for card %d: %s", invCard.CardID, err)
			dc.renderInventory(c, user, nil, 1, 0, username, query, dc.Renderer.T(c, "inventory-message-fetch-fail"))
			return
		}
		cardData[i] = map[string]any{
			"Data": data,
		}
	}

	dc.renderInventory(c, user, cardData, page, total, username, query, "")
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
