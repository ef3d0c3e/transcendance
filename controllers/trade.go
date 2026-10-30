package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Bounds of one side of an offer
const (
	tradeMaxItems = 20
	tradeMaxCount = 999
	// Trades shown in the history of the trades page
	tradeHistoryLimit = 20
)

// Redirection target of an action that cannot run on its offer anymore
const tradeGoneTarget = "/trades?error=settle"

// tradeOffer is the JSON body of a new offer or a counter
type tradeOffer struct {
	Target    string             `json:"target"`
	Give      []models.TradeItem `json:"give"`
	Coins     uint               `json:"coins"`
	Want      []models.TradeItem `json:"want"`
	CounterID uint               `json:"counter_of"`
}

// One offer row of the trades page
type tradeRow struct {
	// The other side of the offer
	Partner string
	// What the viewer would give and get, one line per stack and coins
	GiveLines []string
	GetLines  []string
	// Actions of the row, empty when it is not offered anymore
	AcceptAction  string
	DeclineAction string
	CancelAction  string
	CounterUrl    string
	// Label of the state, on the history rows
	StateLabel string
}

// tradeRole restricts which side of an offer can act on it
type tradeRole int

const (
	// The user the offer is made to
	tradeRoleReceiver tradeRole = iota
	// The user who made the offer
	tradeRoleSender
	// Either side of the offer
	tradeRoleAny
)

// Get the offer of the `ID` param where the user is on the required side
func (tc *Controller) findTrade(c *gin.Context, userID uint, role tradeRole) (*models.Trade, bool) {
	query := tc.DB.Where("id = ?", c.Param("ID"))
	switch role {
	case tradeRoleReceiver:
		query = query.Where("to_id = ?", userID)
	case tradeRoleSender:
		query = query.Where("from_id = ?", userID)
	case tradeRoleAny:
		query = query.Where("from_id = ? OR to_id = ?", userID, userID)
	}
	var trade models.Trade
	if err := query.First(&trade).Error; err != nil {
		return nil, false
	}
	return &trade, true
}

// Check that the stacks of one side are well formed and distinct
func (tc *Controller) checkTradeItems(c *gin.Context, items []models.TradeItem) string {
	if len(items) > tradeMaxItems {
		return tc.Renderer.T(c, "trade-error-items")
	}
	seen := make(map[models.TradeItem]bool, len(items))
	for _, item := range items {
		if item.Count == 0 || item.Count > tradeMaxCount {
			return tc.Renderer.T(c, "trade-error-count")
		}
		if card, _ := tc.Data.GetCard(int(item.CardID)); card == nil {
			return tc.Renderer.T(c, "trade-error-card")
		}
		if seen[item] {
			return tc.Renderer.T(c, "trade-error-duplicate")
		}
		seen[item] = true
	}
	return ""
}

// Check that a user owns enough copies of every stack, returns the message and status when not
func (tc *Controller) checkTradeOwned(c *gin.Context, userID uint, items []models.TradeItem) (string, int) {
	for _, item := range items {
		var count int64
		if err := tc.DB.Model(&models.Card{}).
			Where("owner_id = ? AND card_id = ? AND foil = ?", userID, item.CardID, item.Foil).
			Count(&count).Error; err != nil {
			log.Printf("WARN failed to count cards of user %d: %s", userID, err)
			return tc.Renderer.T(c, "trade-error-internal"), http.StatusInternalServerError
		}
		if count < int64(item.Count) {
			return tc.Renderer.T(c, "trade-error-owned"), http.StatusBadRequest
		}
	}
	return "", 0
}

// Check that an offer is well formed and that both sides own what they promise
func (tc *Controller) validateOffer(c *gin.Context, fromID uint, toID uint, offer *tradeOffer) (string, int) {
	if len(offer.Want) == 0 {
		return tc.Renderer.T(c, "trade-error-empty-want"), http.StatusBadRequest
	}
	if len(offer.Give) == 0 && offer.Coins == 0 {
		return tc.Renderer.T(c, "trade-error-empty-give"), http.StatusBadRequest
	}
	if msg := tc.checkTradeItems(c, offer.Give); msg != "" {
		return msg, http.StatusBadRequest
	}
	if msg := tc.checkTradeItems(c, offer.Want); msg != "" {
		return msg, http.StatusBadRequest
	}
	// Both sides own the cards they put up, the sender the coins too
	if msg, status := tc.checkTradeOwned(c, fromID, offer.Give); msg != "" {
		return msg, status
	}
	if msg, status := tc.checkTradeOwned(c, toID, offer.Want); msg != "" {
		return msg, status
	}
	if offer.Coins > 0 {
		var from models.User
		if err := tc.DB.Select("coins").First(&from, fromID).Error; err != nil {
			log.Printf("WARN failed to get balance of user %d: %s", fromID, err)
			return tc.Renderer.T(c, "trade-error-internal"), http.StatusInternalServerError
		}
		if offer.Coins > from.Coins {
			return tc.Renderer.T(c, "trade-error-coins"), http.StatusBadRequest
		}
	}
	return "", 0
}

// /trades POST: make a trade offer to a user, or counter one with new contents
func (tc *Controller) TradePost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": tc.Renderer.T(c, "error-unauthorized")})
		return
	}

	var offer tradeOffer
	if err := c.ShouldBindJSON(&offer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": tc.Renderer.T(c, "trade-error-invalid")})
		return
	}

	trade := models.Trade{FromID: user.ID}
	if offer.CounterID != 0 {
		// Countering replaces the contents of a pending offer received by
		// the user, and flips its direction
		var old models.Trade
		if err := tc.DB.First(&old, offer.CounterID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": tc.Renderer.T(c, "trade-error-not-found")})
			return
		}
		if old.State != models.TradePending {
			c.JSON(http.StatusBadRequest, gin.H{"message": tc.Renderer.T(c, "trade-error-state")})
			return
		}
		if old.ToID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"message": tc.Renderer.T(c, "trade-error-forbidden")})
			return
		}
		trade.ID = old.ID
		trade.ToID = old.FromID
	} else {
		if strings.EqualFold(offer.Target, user.Username) {
			c.JSON(http.StatusBadRequest, gin.H{"message": tc.Renderer.T(c, "trade-error-self")})
			return
		}
		var target models.User
		if err := tc.DB.Omit("avatar").Where("username = ?", offer.Target).First(&target).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": tc.Renderer.T(c, "trade-error-user")})
			return
		}
		trade.ToID = target.ID
	}

	if msg, status := tc.validateOffer(c, trade.FromID, trade.ToID, &offer); msg != "" {
		c.JSON(status, gin.H{"message": msg})
		return
	}

	give, err := json.Marshal(offer.Give)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": tc.Renderer.T(c, "trade-error-internal")})
		return
	}
	want, err := json.Marshal(offer.Want)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": tc.Renderer.T(c, "trade-error-internal")})
		return
	}
	trade.Give = give
	trade.Coins = offer.Coins
	trade.Want = want
	trade.State = models.TradePending

	if offer.CounterID != 0 {
		// Only a still pending offer can be overwritten by its counter
		result := tc.DB.Model(&models.Trade{}).
			Where("id = ? AND state = ?", trade.ID, models.TradePending).
			Updates(map[string]any{
				"from_id": trade.FromID,
				"to_id":   trade.ToID,
				"give":    trade.Give,
				"coins":   trade.Coins,
				"want":    trade.Want,
			})
		if result.Error != nil {
			log.Printf("WARN failed to counter trade %d: %s", trade.ID, result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"message": tc.Renderer.T(c, "trade-error-internal")})
			return
		}
		if result.RowsAffected != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"message": tc.Renderer.T(c, "trade-error-state")})
			return
		}
	} else {
		if err := tc.DB.Create(&trade).Error; err != nil {
			log.Printf("WARN failed to create trade: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": tc.Renderer.T(c, "trade-error-internal")})
			return
		}
	}

	// The receiver learns about the offer, or about its counter
	kind := models.NotificationTrade
	if offer.CounterID != 0 {
		kind = models.NotificationTradeCounter
	}
	tc.notifyTrade(kind, trade.ToID, user.Username)

	c.JSON(http.StatusCreated, gin.H{"redirect": "/trades"})
}

// Tell the receiver of an offer or of a counter, opening the trades page
func (tc *Controller) notifyTrade(kind string, toID uint, from string) {
	notif := models.Notification{
		UserID: toID,
		Type:   kind,
		Data:   datatypes.JSONMap{"username": from},
		Action: "/trades",
	}
	if err := tc.DB.Create(&notif).Error; err != nil {
		log.Printf("WARN failed to notify trade: %s", err)
	}
}

// Move count copies of each stack between two owners, oldest copies first
func moveCards(tx *gorm.DB, fromID uint, toID uint, items []models.TradeItem) error {
	for _, item := range items {
		var ids []uint
		if err := tx.Model(&models.Card{}).
			Where("owner_id = ? AND card_id = ? AND foil = ?", fromID, item.CardID, item.Foil).
			Order("acquired").
			Limit(int(item.Count)).
			Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) != int(item.Count) {
			return fmt.Errorf("only %d copies left of card %d", len(ids), item.CardID)
		}
		if err := tx.Model(&models.Card{}).Where("id IN ?", ids).
			Update("owner_id", toID).Error; err != nil {
			return err
		}
	}
	return nil
}

// Settle an accepted offer: move the cards and coins between its two sides,
// in the transaction that claimed the offer
func settleTrade(tx *gorm.DB, trade *models.Trade) error {
	give, err := trade.GiveItems()
	if err != nil {
		return err
	}
	want, err := trade.WantItems()
	if err != nil {
		return err
	}

	// The offered cards and coins leave the sender
	if err := moveCards(tx, trade.FromID, trade.ToID, give); err != nil {
		return err
	}
	if trade.Coins > 0 {
		result := tx.Model(&models.User{}).
			Where("id = ? AND coins >= ?", trade.FromID, trade.Coins).
			Update("coins", gorm.Expr("coins - ?", trade.Coins))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("only %d coins left", trade.Coins)
		}
		if err := tx.Model(&models.User{}).Where("id = ?", trade.ToID).
			Update("coins", gorm.Expr("coins + ?", trade.Coins)).Error; err != nil {
			return err
		}
	}

	// The requested cards leave the receiver
	return moveCards(tx, trade.ToID, trade.FromID, want)
}

// /trades/:ID/accept POST: settle a pending offer made to the user
func (tc *Controller) TradeAcceptPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	trade, ok := tc.findTrade(c, user.ID, tradeRoleReceiver)
	if !ok {
		c.Redirect(http.StatusSeeOther, tradeGoneTarget)
		return
	}

	err := tc.DB.Transaction(func(tx *gorm.DB) error {
		// Claim the offer first, so a concurrent accept cannot settle it twice
		result := tx.Model(&models.Trade{}).
			Where("id = ? AND state = ?", trade.ID, models.TradePending).
			Update("state", models.TradeAccepted)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("the offer is not pending anymore")
		}
		return settleTrade(tx, trade)
	})
	if err != nil {
		log.Printf("WARN failed to settle trade %d: %s", trade.ID, err)
		c.Redirect(http.StatusSeeOther, tradeGoneTarget)
		return
	}
	c.Redirect(http.StatusSeeOther, "/trades")
}

// Set the state of a pending offer, when the user is on the side allowed to
func (tc *Controller) closeTrade(c *gin.Context, role tradeRole, state string) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	trade, ok := tc.findTrade(c, user.ID, role)
	if !ok {
		c.Redirect(http.StatusSeeOther, tradeGoneTarget)
		return
	}
	result := tc.DB.Model(&models.Trade{}).
		Where("id = ? AND state = ?", trade.ID, models.TradePending).
		Update("state", state)
	if result.Error != nil {
		log.Printf("WARN failed to close trade %d: %s", trade.ID, result.Error)
	}
	if result.Error != nil || result.RowsAffected != 1 {
		c.Redirect(http.StatusSeeOther, tradeGoneTarget)
		return
	}
	c.Redirect(http.StatusSeeOther, "/trades")
}

// /trades/:ID/decline POST: turn down a pending offer made to the user
func (tc *Controller) TradeDeclinePost(c *gin.Context) {
	tc.closeTrade(c, tradeRoleReceiver, models.TradeDeclined)
}

// /trades/:ID/cancel POST: withdraw a pending offer made by the user
func (tc *Controller) TradeCancelPost(c *gin.Context) {
	tc.closeTrade(c, tradeRoleSender, models.TradeCancelled)
}

// Display lines of one side of an offer: its card stacks and coins
func (tc *Controller) tradeLines(c *gin.Context, items []models.TradeItem, coins uint) []string {
	loc := tc.Renderer.Localizer.GetLocale(c)
	lines := make([]string, 0, len(items)+1)
	for _, item := range items {
		title := "?"
		if card, _ := tc.Data.GetCard(int(item.CardID)); card != nil {
			if name, err := card.Translate(loc, "title"); err == nil {
				title = name
			}
		}
		if item.Foil != "" {
			title += " · " + tc.Renderer.T(c, "inventory-filter-effect-foil")
		}
		lines = append(lines, tc.Renderer.T(c, "trade-stack", "count", int(item.Count), "title", title))
	}
	if coins > 0 {
		lines = append(lines, tc.Renderer.T(c, "shop-price", "amount", int(coins)))
	}
	return lines
}

// Build the rows of a list of offers, as seen by the viewer
func (tc *Controller) tradeRows(c *gin.Context, user *AuthenticatedUser, trades []models.Trade, names map[uint]string) []tradeRow {
	rows := make([]tradeRow, 0, len(trades))
	for i := range trades {
		trade := &trades[i]
		give, _ := trade.GiveItems()
		want, _ := trade.WantItems()

		// Coins belong to the side of the sender, the cards are read from
		// the point of view of the viewer
		var giveCoins, wantCoins uint
		partnerID := trade.ToID
		if trade.FromID == user.ID {
			giveCoins = trade.Coins
		} else {
			give, want = want, give
			wantCoins = trade.Coins
			partnerID = trade.FromID
		}

		row := tradeRow{
			Partner:    names[partnerID],
			GiveLines:  tc.tradeLines(c, give, giveCoins),
			GetLines:   tc.tradeLines(c, want, wantCoins),
			StateLabel: tc.Renderer.T(c, "trade-state-"+trade.State),
		}
		if trade.State == models.TradePending {
			row.StateLabel = ""
			action := "/trades/" + strconv.FormatUint(uint64(trade.ID), 10)
			switch user.ID {
			case trade.ToID:
				row.AcceptAction = action + "/accept"
				row.DeclineAction = action + "/decline"
				row.CounterUrl = "/trades/new?counter=" + strconv.FormatUint(uint64(trade.ID), 10)
			case trade.FromID:
				row.CancelAction = action + "/cancel"
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// Usernames of the other sides of the given offers
func (tc *Controller) tradePartners(trades []models.Trade, userID uint) (map[uint]string, error) {
	ids := make([]uint, 0, len(trades))
	for i := range trades {
		if trades[i].FromID == userID {
			ids = append(ids, trades[i].ToID)
		} else {
			ids = append(ids, trades[i].FromID)
		}
	}
	names := make(map[uint]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}
	var users []models.User
	if err := tc.DB.Omit("avatar").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		names[users[i].ID] = users[i].Username
	}
	return names, nil
}

// /trades GET: the incoming and outgoing offers of the user, and their past trades
func (tc *Controller) TradesGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	var incoming, outgoing, history []models.Trade
	if err := tc.DB.Where("to_id = ? AND state = ?", user.ID, models.TradePending).
		Order("updated_at DESC").Find(&incoming).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if err := tc.DB.Where("from_id = ? AND state = ?", user.ID, models.TradePending).
		Order("updated_at DESC").Find(&outgoing).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if err := tc.DB.Where("(from_id = ? OR to_id = ?) AND state != ?", user.ID, user.ID, models.TradePending).
		Order("updated_at DESC").Limit(tradeHistoryLimit).Find(&history).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	fields := map[string]any{
		"Incoming": tc.tradeRows(c, user, incoming, mustPartners(tc, incoming, user.ID)),
		"Outgoing": tc.tradeRows(c, user, outgoing, mustPartners(tc, outgoing, user.ID)),
		"History":  tc.tradeRows(c, user, history, mustPartners(tc, history, user.ID)),
	}
	// An action that could not run points the message out on the page
	if c.Query("error") == "settle" {
		fields["Error"] = tc.Renderer.T(c, "trade-error-settle")
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": tc.Renderer.T(c, "trades-title"),
		"User":  user,
	})
	builder.Add("trades", "Content", fields)
	tc.Renderer.Render(c, &builder)
}

// Usernames of the other sides of the offers, empty names when they cannot be loaded
func mustPartners(tc *Controller, trades []models.Trade, userID uint) map[uint]string {
	names, err := tc.tradePartners(trades, userID)
	if err != nil {
		log.Printf("WARN failed to get trade partners: %s", err)
		return map[uint]string{}
	}
	return names
}

// /trades/new GET: the side by side panel to build an offer
//
// Query: `u` to make an offer to a user, or `counter` to counter one received
func (tc *Controller) TradeMenuGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	targetName := c.Query("u")
	counterID := 0
	give := "[]"
	want := "[]"

	if counter := c.Query("counter"); counter != "" {
		id, err := strconv.Atoi(counter)
		if err != nil {
			tc.Renderer.RenderError(c, views.ErrorPage{
				User:       user,
				StatusCode: http.StatusBadRequest,
				Message:    tc.Renderer.T(c, "trade-error-not-found"),
			})
			return
		}
		var old models.Trade
		if err := tc.DB.First(&old, id).Error; err != nil {
			tc.Renderer.RenderError(c, views.ErrorPage{
				User:       user,
				StatusCode: http.StatusNotFound,
				Message:    tc.Renderer.T(c, "trade-error-not-found"),
			})
			return
		}
		if old.State != models.TradePending || old.ToID != user.ID {
			tc.Renderer.RenderError(c, views.ErrorPage{
				User:       user,
				StatusCode: http.StatusBadRequest,
				Message:    tc.Renderer.T(c, "trade-error-state"),
			})
			return
		}
		var other models.User
		if err := tc.DB.Omit("avatar").First(&other, old.FromID).Error; err != nil {
			tc.Renderer.RenderError(c, views.ErrorPage{
				User:       user,
				StatusCode: http.StatusNotFound,
				Message:    tc.Renderer.T(c, "trade-error-not-found"),
			})
			return
		}
		targetName = other.Username
		counterID = int(old.ID)
		// The counter starts mirrored: the cards the sender offered become
		// what the user wants, and the cards they asked for what the user gives
		giveItems, _ := old.WantItems()
		wantItems, _ := old.GiveItems()
		give = marshalTradeItems(giveItems)
		want = marshalTradeItems(wantItems)
	}

	if strings.EqualFold(targetName, user.Username) {
		tc.Renderer.RenderError(c, views.ErrorPage{
			User:       user,
			StatusCode: http.StatusBadRequest,
			Message:    tc.Renderer.T(c, "trade-error-self"),
		})
		return
	}
	var target models.User
	if err := tc.DB.Omit("avatar").Where("username = ?", targetName).First(&target).Error; err != nil {
		tc.Renderer.RenderError(c, views.ErrorPage{
			User:       user,
			StatusCode: http.StatusNotFound,
			Message:    tc.Renderer.T(c, "trade-error-user"),
		})
		return
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": tc.Renderer.T(c, "trade-menu-title", "username", target.Username),
		"User":  user,
	})
	builder.Add("trade-menu", "Content", map[string]any{
		"Target":      target.Username,
		"StacksUrl":   "/api/trade/stacks?u=" + url.QueryEscape(target.Username),
		"SubmitUrl":   "/trades",
		"CounterID":   counterID,
		"GivePrefill": give,
		"WantPrefill": want,
	})
	tc.Renderer.Render(c, &builder)
}

// JSON list of the stacks prefilled into one side of the menu
func marshalTradeItems(items []models.TradeItem) string {
	if items == nil {
		items = []models.TradeItem{}
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}
