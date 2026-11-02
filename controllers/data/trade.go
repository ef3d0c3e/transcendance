package data

import (
	"fmt"
	"net/http"
	"strconv"
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

// Card stacks of a user, with the fields and the render shown by the trade
// menu. The counts are the copies free to be put up: what the user has
// committed to other pending offers is left out
func (dc *DataController) tradeStacks(c *gin.Context, loc string, userID uint, committed *controllers.TradeCommitments) ([]tradeStack, error) {
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
		free := row.Count
		if locked := committed.Copies(row.CardID, row.Foil); free > locked {
			free -= locked
		} else {
			free = 0
		}
		stacks = append(stacks, tradeStack{
			CardID: row.CardID,
			Foil:   row.Foil,
			Count:  free,
			Title:  title,
			Rarity: card.Rarity,
			Image:  fmt.Sprint("/cards/", collection.ID*1000+card.ID, "/thumbnail"),
			HTML:   html,
		})
	}
	return stacks, nil
}

// /api/trade/stacks GET: the card stacks of the user and of the trade partner
//
// Query: `u` is the trade partner, `counter` is the offer being countered if
// any, whose contents are left unlocked on the partner's side since it will be
// replaced
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
	if blocked, err := controllers.TradeBlocked(dc.DB, user.ID, target.ID); err != nil || blocked {
		c.JSON(http.StatusForbidden, gin.H{"message": dc.Renderer.T(c, "trade-error-blocked")})
		return
	}

	// The countered offer must be one made by the partner to the user: it is
	// the only one its counter replaces and unlocks
	var skipTradeID uint
	if counter := c.Query("counter"); counter != "" {
		id, err := strconv.ParseUint(counter, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": dc.Renderer.T(c, "trade-error-invalid")})
			return
		}
		var old models.Trade
		if err := dc.DB.Where("id = ? AND from_id = ? AND to_id = ? AND state = ?",
			uint(id), target.ID, user.ID, models.TradePending).First(&old).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": dc.Renderer.T(c, "trade-error-state")})
			return
		}
		skipTradeID = old.ID
	}

	// What each side put up in other pending offers is locked there: the
	// stacks only show the copies free to be put up again. The countered
	// offer is the partner's, so only their side skips it
	mine, err := controllers.TradeCommitmentsOf(dc.DB, user.ID, 0)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	theirs, err := controllers.TradeCommitmentsOf(dc.DB, target.ID, skipTradeID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	loc := dc.Renderer.Localizer.GetLocale(c)
	mineStacks, err := dc.tradeStacks(c, loc, user.ID, mine)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	theirStacks, err := dc.tradeStacks(c, loc, target.ID, theirs)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"coins":       mine.Free(user.Coins),
		"their_coins": theirs.Free(target.Coins),
		"mine":        mineStacks,
		"theirs":      theirStacks,
	})
}
