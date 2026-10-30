package data

import (
	"fmt"
	"net/http"
	"strings"

	"transcendance/controllers"
	"transcendance/models"

	"github.com/gin-gonic/gin"
)

// One (card, foil) stack of a user, shown by the trade menu
type tradeStack struct {
	CardID uint   `json:"card_id"`
	Foil   string `json:"foil"`
	Count  uint   `json:"count"`
	Title  string `json:"title"`
	Rarity string `json:"rarity"`
	Image  string `json:"image"`
	// The stack's card, rendered as on the inventory page
	HTML string `json:"html"`
}

// One (card, foil) stack as grouped by the database
type tradeStackRow struct {
	CardID uint
	Foil   string
	Count  uint
}

// Card stacks of a user, with the fields and the render shown by the trade menu
func (dc *DataController) tradeStacks(c *gin.Context, loc string, userID uint) ([]tradeStack, error) {
	var rows []tradeStackRow
	if err := dc.DB.Model(&models.Card{}).
		Select("card_id, foil, COUNT(*) AS count").
		Where("owner_id = ?", userID).
		Group("card_id, foil").
		Order("card_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	stacks := make([]tradeStack, 0, len(rows))
	for _, row := range rows {
		card, collection := dc.Data.GetCard(int(row.CardID))
		if card == nil {
			continue
		}
		title, err := card.Translate(loc, "title")
		if err != nil {
			return nil, err
		}
		cardData, err := GetCardData(c, loc, dc.Renderer, collection, card)
		if err != nil {
			return nil, err
		}
		cardData["CardCopies"] = row.Count
		setCardFoil(cardData, row.Foil)
		html, err := dc.Renderer.RenderToString("card", map[string]any{"Data": cardData})
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, tradeStack{
			CardID: row.CardID,
			Foil:   row.Foil,
			Count:  row.Count,
			Title:  title,
			Rarity: card.Rarity,
			Image:  fmt.Sprint("/cards/", collection.ID*1000+card.ID, "/thumbnail"),
			HTML:   html,
		})
	}
	return stacks, nil
}

// /api/trade/stacks GET: the card stacks of the user and of the trade partner
func (dc *DataController) TradeStacksGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": dc.Renderer.T(c, "error-unauthorized")})
		return
	}

	other := c.Query("u")
	if strings.EqualFold(other, user.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"message": dc.Renderer.T(c, "trade-error-self")})
		return
	}
	var target models.User
	if err := dc.DB.Omit("avatar").Where("username = ?", other).First(&target).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": dc.Renderer.T(c, "trade-error-user")})
		return
	}

	loc := dc.Renderer.Localizer.GetLocale(c)
	mine, err := dc.tradeStacks(c, loc, user.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	theirs, err := dc.tradeStacks(c, loc, target.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"coins":  user.Coins,
		"mine":   mine,
		"theirs": theirs,
	})
}
