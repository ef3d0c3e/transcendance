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
)

const (
	inventoryPageSize  = 20
	acquiredDateFormat = "2006-01-02"
)

// InventoryAddGet seeds a card into the authenticated user's inventory.
// DEV STOPGAP: this is a plain GET on purpose while TRANS-3 (pack opening)
// doesn't exist yet — it is the only way to populate an inventory.
// `?effects=foil,sparkle` rolls copies with the given special effects.
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

	// Only cards known from the data files can be granted, so the global ID
	// (`collection_id * 1000 + card_id`) always resolves back to a real card.
	card, collection := dc.Data.GetCard(cardId)
	if card == nil || collection == nil {
		c.String(http.StatusBadRequest, "invalid card")
		return
	}

	effects, err := ParseEffects(c.Query("effects"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid effects: %s", err)
		return
	}

	row := models.Card{
		CardID:   uint(cardId),
		Acquired: time.Now(),
		Effects:  effects,
		OwnerID:  user.ID,
	}
	if err := dc.DB.Create(&row).Error; err != nil {
		c.String(http.StatusInternalServerError, "failed to insert card: %s", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       row.ID,
		"card_id":  row.CardID,
		"acquired": row.Acquired,
		"effects":  row.Effects,
	})
}

func (dc *DataController) renderInventory(c *gin.Context, user *controllers.AuthenticatedUser, cards []map[string]any, page int, total int64, username string, message string) {
	title := dc.Renderer.T(c, "inventory-title")
	if user != nil && username != "" && !strings.EqualFold(user.Username, username) {
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
		"Title":   title,
		"Message": message,
		"Cards":   cards,
	})
	totalPages := total / int64(inventoryPageSize)
	if total%int64(inventoryPageSize) != 0 {
		totalPages = totalPages + 1
	}
	// Keep paging inside the inventory being viewed (`?u=` when it's not ours)
	pageUrl := "?p="
	if username != "" && (user == nil || !strings.EqualFold(user.Username, username)) {
		pageUrl = "?u=" + url.QueryEscape(username) + "&p="
	}
	content.Add("paginator", "Paginator", map[string]any{
		"Current": page,
		"Total":   int(totalPages),
		"Url":     pageUrl,
	})
	dc.Renderer.Render(c, &builder)
}

func (dc *DataController) InventoryGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)

	if user == nil {
		c.Status(http.StatusForbidden)
		dc.renderInventory(c, nil, nil, 1, 0, "", dc.Renderer.T(c, "inventory-unauthenticated"))
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
		// Newest copies first, with the row id as a stable tiebreak so that
		// pagination stays consistent between pages.
		Order("acquired DESC, id DESC").
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
	cardData := make([]map[string]any, 0, len(cards))
	for _, invCard := range cards {
		card, collection := dc.Data.GetCard(int(invCard.CardID))
		if card == nil || collection == nil {
			// Stale row: the card it points at no longer exists in the data
			// files. Skip it instead of failing the whole inventory.
			log.Printf("WARN inventory of '%s' references unknown card id %d", username, invCard.CardID)
			continue
		}
		data, err := GetCardData(c, loc, dc.Renderer, collection, card)
		if err != nil {
			log.Printf("WARN failed to get card data for card %d: %s", invCard.CardID, err)
			dc.renderInventory(c, user, nil, 1, 0, username, dc.Renderer.T(c, "inventory-message-fetch-fail"))
			return
		}
		data["CardEffects"] = InventoryCardEffects(invCard.Effects)
		data["CardAcquired"] = dc.Renderer.T(c, "card-acquired", "date", invCard.Acquired.Format(acquiredDateFormat))
		cardData = append(cardData, map[string]any{
			"Data": data,
		})
	}

	dc.renderInventory(c, user, cardData, page, total, username, "")
}
