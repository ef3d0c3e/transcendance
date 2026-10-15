package data

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"transcendance/controllers"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
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

func (dc *DataController) renderInventory(c *gin.Context, user *controllers.AuthenticatedUser, cards []map[string]any, page int, total int64, username string, message string) {
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
		message = dc.Renderer.T(c, "inventory-message-empty")
	}
	content := builder.Add("inventory", "Content", map[string]any{
		"Title": title,
		"Message": message,
		"Cards":   cards,
	})
	totalPages := total / int64(inventoryPageSize)
	if total%int64(inventoryPageSize) != 0 {
		totalPages = totalPages + 1
	}
	content.Add("paginator", "Paginator", map[string]any{
		"Current": page,
		"Total":   int(totalPages),
		"Url":     "?p=",
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
			dc.renderInventory(c, user, nil, 1, 0, username, dc.Renderer.T(c, "inventory-message-user-not-found"))
			return
		}
		userID = otherUser.ID
	}

	var total int64
	if err := dc.DB.
		Model(&models.Card{}).
		Where("owner_id = ?", userID).
		Count(&total).
		Error; err != nil {
		log.Printf("WARN failed to get cards count for user '%s': %s", username, err)
		dc.renderInventory(c, user, nil, 1, 0, username, dc.Renderer.T(c, "inventory-message-fetch-fail"))
		return
	}

	var cards []models.Card
	if err := dc.DB.
		Where("owner_id = ?", userID).
		Offset((page - 1) * inventoryPageSize).
		Limit(inventoryPageSize).
		Find(&cards).
		Error; err != nil {
		log.Printf("WARN failed to get cards for user '%s': %s", username, err)
		dc.renderInventory(c, user, nil, 1, 0, username, dc.Renderer.T(c, "inventory-message-fetch-fail"))
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
			dc.renderInventory(c, user, nil, 1, 0, username, dc.Renderer.T(c, "inventory-message-fetch-fail"))
			return
		}
		cardData[i] = map[string]any{
			"Data": data,
		}
	}

	dc.renderInventory(c, user, cardData, page, total, username, "")
}
