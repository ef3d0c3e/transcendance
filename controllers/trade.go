package controllers

import (
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
	"gorm.io/gorm/clause"
)

// Bounds of one side of an offer
const (
	tradeMaxItems     = 20
	tradeMaxCount     = 999
	tradeHistoryLimit = 20
	// Caps on pending offers, each of which notifies its receiver
	tradeMaxPendingPerPair = 5
	tradeMaxPendingPerUser = 50
)

// Redirection target of an action that cannot run on its offer anymore
const tradeGoneTarget = "/trades?error=settle"

type tradeOffer struct {
	Target    string             `json:"target"`
	Give      []models.TradeItem `json:"give"`
	Coins     uint               `json:"coins"`
	Want      []models.TradeItem `json:"want"`
	WantCoins uint               `json:"want_coins"`
	CounterID uint               `json:"counter_of"`
}

// tradeRejection is an offer refused with a message, rolling its transaction back
type tradeRejection struct {
	message string
	status  int
}

func (r *tradeRejection) Error() string {
	return r.message
}

// LockUsers locks the rows of the users until the transaction ends, in id
// order so two transactions cannot deadlock. Whatever moves cards or coins
// takes it first, so its checks still hold when it writes
func LockUsers(tx *gorm.DB, ids ...uint) error {
	var users []models.User
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id IN ?", ids).
		Order("id").
		Find(&users).Error
}

func TradeCommittedCoins(db *gorm.DB, userID uint) (uint, error) {
	var coins uint
	err := db.Model(&models.Trade{}).
		Select("COALESCE(SUM(coins), 0)").
		Where("from_id = ? AND state = ?", userID, models.TradePending).
		Scan(&coins).Error
	return coins, err
}

func FreeCoins(balance uint, committed uint) uint {
	if balance < committed {
		return 0
	}
	return balance - committed
}

// TradeBlocked reports whether either user has blocked the other
func TradeBlocked(db *gorm.DB, userID uint, otherID uint) (bool, error) {
	var count int64
	err := db.Model(&models.Relation{}).
		Where("type = ? AND ((user_id = ? AND target_id = ?) OR (user_id = ? AND target_id = ?))",
			models.RelationBlocked, userID, otherID, otherID, userID).
		Count(&count).Error
	return count > 0, err
}

// tradeStackKey identifies a stack of identical copies in an inventory
type tradeStackKey struct {
	CardID uint
	Foil   string
}

// TradeCommitments are the cards and coins a user put up in their pending offers
type TradeCommitments struct {
	stacks map[tradeStackKey]uint
	Coins  uint
}

func (tc *TradeCommitments) Copies(cardID uint, foil string) uint {
	return tc.stacks[tradeStackKey{CardID: cardID, Foil: foil}]
}

func (tc *TradeCommitments) Free(balance uint) uint {
	if balance < tc.Coins {
		return 0
	}
	return balance - tc.Coins
}

// TradeCommitmentsOf skips skipTradeID if set: a counter replaces that offer,
// freeing what it held
func TradeCommitmentsOf(db *gorm.DB, userID uint, skipTradeID uint) (*TradeCommitments, error) {
	pending := func(query *gorm.DB) *gorm.DB {
		query = query.Where("trades.from_id = ? AND trades.state = ?", userID, models.TradePending)
		if skipTradeID != 0 {
			query = query.Where("trades.id <> ?", skipTradeID)
		}
		return query
	}
	var stacks []struct {
		CardID uint
		Foil   string
		Count  uint
	}
	if err := pending(db.Table("trade_items").
		Select("trade_items.card_id, trade_items.foil, SUM(trade_items.count) AS count").
		Joins("JOIN trades ON trades.id = trade_items.trade_id").
		Where("trade_items.side = ?", models.TradeSideGive)).
		Group("trade_items.card_id, trade_items.foil").
		Scan(&stacks).Error; err != nil {
		return nil, err
	}
	commitments := TradeCommitments{stacks: make(map[tradeStackKey]uint, len(stacks))}
	for _, stack := range stacks {
		commitments.stacks[tradeStackKey{CardID: stack.CardID, Foil: stack.Foil}] = stack.Count
	}
	if err := pending(db.Model(&models.Trade{}).Select("COALESCE(SUM(trades.coins), 0)")).
		Scan(&commitments.Coins).Error; err != nil {
		return nil, err
	}
	return &commitments, nil
}

type tradeRow struct {
	Partner string
	// From the viewer's side, one line per stack and one for the coins
	GiveLines []string
	GetLines  []string
	// Empty when the offer is not pending
	AcceptAction  string
	DeclineAction string
	CancelAction  string
	CounterUrl    string
	// Only set on the history rows
	StateLabel string
}

// tradeRole restricts which side of an offer can act on it
type tradeRole int

const (
	tradeRoleReceiver tradeRole = iota
	tradeRoleSender
	tradeRoleAny
)

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

func (tc *Controller) checkTradeItems(c *gin.Context, items []models.TradeItem) string {
	if len(items) > tradeMaxItems {
		return tc.Renderer.T(c, "trade-error-items")
	}
	seen := make(map[tradeStackKey]bool, len(items))
	for _, item := range items {
		if item.Count == 0 || item.Count > tradeMaxCount {
			return tc.Renderer.T(c, "trade-error-count")
		}
		if card, _ := tc.Data.GetCard(int(item.CardID)); card == nil || !models.ValidFoil(item.Foil) {
			return tc.Renderer.T(c, "trade-error-card")
		}
		key := tradeStackKey{CardID: item.CardID, Foil: item.Foil}
		if seen[key] {
			return tc.Renderer.T(c, "trade-error-duplicate")
		}
		seen[key] = true
	}
	return ""
}

// checkTradeOwned also checks, when committed is set, that the copies are not
// up in another pending offer
func (tc *Controller) checkTradeOwned(c *gin.Context, db *gorm.DB, userID uint, items []models.TradeItem, committed *TradeCommitments) (string, int) {
	if len(items) == 0 {
		return "", 0
	}
	cardIDs := make([]uint, 0, len(items))
	for _, item := range items {
		cardIDs = append(cardIDs, item.CardID)
	}
	var rows []struct {
		CardID uint
		Foil   string
		Count  int64
	}
	if err := db.Model(&models.Card{}).
		Select("card_id, foil, COUNT(*) AS count").
		Where("owner_id = ? AND card_id IN ?", userID, cardIDs).
		Group("card_id, foil").
		Scan(&rows).Error; err != nil {
		log.Printf("WARN failed to count cards of user %d: %s", userID, err)
		return tc.Renderer.T(c, "trade-error-internal"), http.StatusInternalServerError
	}
	owned := make(map[tradeStackKey]int64, len(rows))
	for _, row := range rows {
		owned[tradeStackKey{CardID: row.CardID, Foil: row.Foil}] = row.Count
	}

	for _, item := range items {
		count := owned[tradeStackKey{CardID: item.CardID, Foil: item.Foil}]
		if count < int64(item.Count) {
			return tc.Renderer.T(c, "trade-error-owned"), http.StatusBadRequest
		}
		if committed != nil {
			free := count - int64(committed.Copies(item.CardID, item.Foil))
			if free < int64(item.Count) {
				return tc.Renderer.T(c, "trade-error-locked"), http.StatusBadRequest
			}
		}
	}
	return "", 0
}

// validateOffer runs in the transaction holding the locks of both sides.
// counterID is the countered offer, made by the other side
func (tc *Controller) validateOffer(c *gin.Context, tx *gorm.DB, fromID uint, toID uint, offer *tradeOffer, counterID uint) (string, int) {
	if len(offer.Want) == 0 && offer.WantCoins == 0 {
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
	// What is up in another pending offer is locked. The countered offer is the
	// other side's, so nothing is skipped here
	committed, err := TradeCommitmentsOf(tx, fromID, 0)
	if err != nil {
		log.Printf("WARN failed to get commitments of user %d: %s", fromID, err)
		return tc.Renderer.T(c, "trade-error-internal"), http.StatusInternalServerError
	}
	// The other side can only give what their own pending offers leave free, as
	// when accepting. The countered offer is theirs and gets replaced, so it is skipped
	theirs, err := TradeCommitmentsOf(tx, toID, counterID)
	if err != nil {
		log.Printf("WARN failed to get commitments of user %d: %s", toID, err)
		return tc.Renderer.T(c, "trade-error-internal"), http.StatusInternalServerError
	}
	if msg, status := tc.checkTradeOwned(c, tx, fromID, offer.Give, committed); msg != "" {
		return msg, status
	}
	if msg, status := tc.checkTradeOwned(c, tx, toID, offer.Want, theirs); msg != "" {
		return msg, status
	}
	if offer.Coins > 0 {
		var from models.User
		if err := tx.Select("coins").First(&from, fromID).Error; err != nil {
			log.Printf("WARN failed to get balance of user %d: %s", fromID, err)
			return tc.Renderer.T(c, "trade-error-internal"), http.StatusInternalServerError
		}
		if offer.Coins > from.Coins {
			return tc.Renderer.T(c, "trade-error-coins"), http.StatusBadRequest
		}
		if int64(offer.Coins) > int64(from.Coins)-int64(committed.Coins) {
			return tc.Renderer.T(c, "trade-error-locked-coins"), http.StatusBadRequest
		}
	}
	if offer.WantCoins > 0 {
		var to models.User
		if err := tx.Select("coins").First(&to, toID).Error; err != nil {
			log.Printf("WARN failed to get balance of user %d: %s", toID, err)
			return tc.Renderer.T(c, "trade-error-internal"), http.StatusInternalServerError
		}
		if offer.WantCoins > theirs.Free(to.Coins) {
			return tc.Renderer.T(c, "trade-error-want-coins"), http.StatusBadRequest
		}
	}
	return "", 0
}

// /trades POST: make or counter an offer
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
		// A counter replaces a received offer and flips its direction. Checked again
		// under the locks: a concurrent counter may flip it first
		var old models.Trade
		if err := tc.DB.First(&old, offer.CounterID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": tc.Renderer.T(c, "trade-error-not-found")})
			return
		}
		trade.ID = old.ID
		trade.ToID = old.FromID
	} else {
		var target models.User
		if err := tc.DB.Omit("avatar").Where("username = ?", offer.Target).First(&target).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": tc.Renderer.T(c, "trade-error-user")})
			return
		}
		// Compared by ID: usernames compare ignoring case and trailing spaces in the database
		if target.ID == user.ID {
			c.JSON(http.StatusBadRequest, gin.H{"message": tc.Renderer.T(c, "trade-error-self")})
			return
		}
		trade.ToID = target.ID
	}

	trade.Coins = offer.Coins
	trade.WantCoins = offer.WantCoins
	trade.State = models.TradePending

	err := tc.DB.Transaction(func(tx *gorm.DB) error {
		// Locked until the offer is saved, so nothing concurrent can take what it checks
		if err := LockUsers(tx, trade.FromID, trade.ToID); err != nil {
			return err
		}

		if offer.CounterID != 0 {
			var old models.Trade
			if err := tx.First(&old, trade.ID).Error; err != nil {
				return &tradeRejection{tc.Renderer.T(c, "trade-error-not-found"), http.StatusNotFound}
			}
			if old.State != models.TradePending {
				return &tradeRejection{tc.Renderer.T(c, "trade-error-state"), http.StatusBadRequest}
			}
			if old.ToID != user.ID || old.FromID != trade.ToID {
				return &tradeRejection{tc.Renderer.T(c, "trade-error-forbidden"), http.StatusForbidden}
			}
		}

		if blocked, err := TradeBlocked(tx, trade.FromID, trade.ToID); err != nil {
			return err
		} else if blocked {
			return &tradeRejection{tc.Renderer.T(c, "trade-error-blocked"), http.StatusForbidden}
		}

		if msg, status := tc.validateOffer(c, tx, trade.FromID, trade.ToID, &offer, offer.CounterID); msg != "" {
			return &tradeRejection{msg, status}
		}

		if offer.CounterID == 0 {
			var toTarget, toAnyone int64
			if err := tx.Model(&models.Trade{}).Where("from_id = ? AND to_id = ? AND state = ?",
				trade.FromID, trade.ToID, models.TradePending).Count(&toTarget).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Trade{}).Where("from_id = ? AND state = ?",
				trade.FromID, models.TradePending).Count(&toAnyone).Error; err != nil {
				return err
			}
			if toTarget >= tradeMaxPendingPerPair || toAnyone >= tradeMaxPendingPerUser {
				return &tradeRejection{tc.Renderer.T(c, "trade-error-too-many"), http.StatusConflict}
			}
			if err := tx.Create(&trade).Error; err != nil {
				return err
			}
			return models.SaveTradeItems(tx, trade.ID, offer.Give, offer.Want)
		}
		result := tx.Model(&models.Trade{}).
			Where("id = ? AND state = ? AND from_id = ? AND to_id = ?",
				trade.ID, models.TradePending, trade.ToID, trade.FromID).
			Updates(map[string]any{
				"from_id":    trade.FromID,
				"to_id":      trade.ToID,
				"coins":      trade.Coins,
				"want_coins": trade.WantCoins,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return &tradeRejection{tc.Renderer.T(c, "trade-error-state"), http.StatusBadRequest}
		}
		return models.SaveTradeItems(tx, trade.ID, offer.Give, offer.Want)
	})
	var rejection *tradeRejection
	if errors.As(err, &rejection) {
		c.JSON(rejection.status, gin.H{"message": rejection.message})
		return
	}
	if err != nil {
		log.Printf("WARN failed to save trade offer of user %d: %s", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": tc.Renderer.T(c, "trade-error-internal")})
		return
	}

	kind := models.NotificationTrade
	if offer.CounterID != 0 {
		kind = models.NotificationTradeCounter
	}
	tc.notifyTrade(kind, trade.ToID, user.Username)

	c.JSON(http.StatusCreated, gin.H{"redirect": "/trades"})
}

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
		// The owner is checked again, the copies may have left since they were read
		result := tx.Model(&models.Card{}).Where("id IN ? AND owner_id = ?", ids, fromID).
			Update("owner_id", toID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(ids)) {
			return fmt.Errorf("copies of card %d left their owner during the trade", item.CardID)
		}
	}
	return nil
}

func moveCoins(tx *gorm.DB, fromID uint, toID uint, amount uint) error {
	if amount == 0 {
		return nil
	}
	result := tx.Model(&models.User{}).
		Where("id = ? AND coins >= ?", fromID, amount).
		Update("coins", gorm.Expr("coins - ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("only %d coins left", amount)
	}
	return tx.Model(&models.User{}).Where("id = ?", toID).
		Update("coins", gorm.Expr("coins + ?", amount)).Error
}

// settleTrade runs in the transaction that claimed the offer
func settleTrade(tx *gorm.DB, trade *models.Trade) error {
	items, err := trade.Items(tx)
	if err != nil {
		return err
	}
	give, want := items.Give, items.Want

	if err := moveCards(tx, trade.FromID, trade.ToID, give); err != nil {
		return err
	}
	if err := moveCoins(tx, trade.FromID, trade.ToID, trade.Coins); err != nil {
		return err
	}

	if err := moveCards(tx, trade.ToID, trade.FromID, want); err != nil {
		return err
	}
	return moveCoins(tx, trade.ToID, trade.FromID, trade.WantCoins)
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
		if err := LockUsers(tx, trade.FromID, trade.ToID); err != nil {
			return err
		}
		// Read again under the locks: a counter may have replaced the offer since
		var current models.Trade
		if err := tx.Where("id = ? AND from_id = ? AND to_id = ? AND state = ?",
			trade.ID, trade.FromID, user.ID, models.TradePending).First(&current).Error; err != nil {
			return fmt.Errorf("the offer is not pending anymore: %w", err)
		}

		if blocked, err := TradeBlocked(tx, current.FromID, current.ToID); err != nil {
			return err
		} else if blocked {
			return errors.New("one side of the offer blocked the other")
		}

		// What the receiver gives must not be locked in their own pending offers
		committed, err := TradeCommitmentsOf(tx, user.ID, 0)
		if err != nil {
			return err
		}
		items, err := current.Items(tx)
		if err != nil {
			return err
		}
		if msg, _ := tc.checkTradeOwned(c, tx, user.ID, items.Want, committed); msg != "" {
			return errors.New(msg)
		}
		if current.WantCoins > 0 {
			var receiver models.User
			if err := tx.Select("coins").First(&receiver, user.ID).Error; err != nil {
				return err
			}
			if current.WantCoins > committed.Free(receiver.Coins) {
				return errors.New("the requested coins are up in another trade")
			}
		}

		// Claim the offer, so a concurrent accept cannot settle it twice
		result := tx.Model(&models.Trade{}).
			Where("id = ? AND state = ?", current.ID, models.TradePending).
			Update("state", models.TradeAccepted)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("the offer is not pending anymore")
		}
		return settleTrade(tx, &current)
	})
	if err != nil {
		log.Printf("WARN failed to settle trade %d: %s", trade.ID, err)
		c.Redirect(http.StatusSeeOther, tradeGoneTarget)
		return
	}
	c.Redirect(http.StatusSeeOther, "/trades")
}

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

func (tc *Controller) tradeRows(c *gin.Context, user *AuthenticatedUser, trades []models.Trade, names map[uint]string) ([]tradeRow, error) {
	ids := make([]uint, len(trades))
	for i := range trades {
		ids[i] = trades[i].ID
	}
	items, err := models.LoadTradeItems(tc.DB, ids...)
	if err != nil {
		return nil, err
	}
	rows := make([]tradeRow, 0, len(trades))
	for i := range trades {
		trade := &trades[i]
		give, want := items[trade.ID].Give, items[trade.ID].Want

		// Seen from the viewer's side: swap the sides when they received the offer
		var giveCoins, wantCoins uint
		partnerID := trade.ToID
		if trade.FromID == user.ID {
			giveCoins = trade.Coins
			wantCoins = trade.WantCoins
		} else {
			give, want = want, give
			giveCoins = trade.WantCoins
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
	return rows, nil
}

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

// /trades GET: pending offers both ways and the trade history
func (tc *Controller) TradesGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	var incoming, outgoing, history []models.Trade
	if err := tc.DB.Where("to_id = ? AND state = ?", user.ID, models.TradePending).
		Order("updated_at DESC").Find(&incoming).Error; err != nil {
		InternalError(c, tc.Renderer, err)
		return
	}
	if err := tc.DB.Where("from_id = ? AND state = ?", user.ID, models.TradePending).
		Order("updated_at DESC").Find(&outgoing).Error; err != nil {
		InternalError(c, tc.Renderer, err)
		return
	}
	if err := tc.DB.Where("(from_id = ? OR to_id = ?) AND state != ?", user.ID, user.ID, models.TradePending).
		Order("updated_at DESC").Limit(tradeHistoryLimit).Find(&history).Error; err != nil {
		InternalError(c, tc.Renderer, err)
		return
	}

	all := make([]models.Trade, 0, len(incoming)+len(outgoing)+len(history))
	all = append(append(append(all, incoming...), outgoing...), history...)
	names := mustPartners(tc, all, user.ID)
	fields := map[string]any{}
	for key, trades := range map[string][]models.Trade{"Incoming": incoming, "Outgoing": outgoing, "History": history} {
		rows, err := tc.tradeRows(c, user, trades, names)
		if err != nil {
			InternalError(c, tc.Renderer, err)
			return
		}
		fields[key] = rows
	}
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

func mustPartners(tc *Controller, trades []models.Trade, userID uint) map[uint]string {
	names, err := tc.tradePartners(trades, userID)
	if err != nil {
		log.Printf("WARN failed to get trade partners: %s", err)
		return map[uint]string{}
	}
	return names
}

// /trades/new GET: build an offer to ?u=, or counter the offer ?counter=
func (tc *Controller) TradeMenuGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	targetName := c.Query("u")
	counterID := 0
	giveCoins := 0
	wantCoins := 0

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
		// The counter starts mirrored, as the stacks API does for the cards
		giveCoins = int(old.WantCoins)
		wantCoins = int(old.Coins)
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
	// Compared by ID too: usernames compare ignoring case and trailing spaces in the database
	if target.ID == user.ID {
		tc.Renderer.RenderError(c, views.ErrorPage{
			User:       user,
			StatusCode: http.StatusBadRequest,
			Message:    tc.Renderer.T(c, "trade-error-self"),
		})
		return
	}
	if blocked, err := TradeBlocked(tc.DB, user.ID, target.ID); err != nil || blocked {
		tc.Renderer.RenderError(c, views.ErrorPage{
			User:       user,
			StatusCode: http.StatusForbidden,
			Message:    tc.Renderer.T(c, "trade-error-blocked"),
		})
		return
	}

	// Without the countered offer, so its cards show as free to mirror
	stacksUrl := "/api/trade/stacks?u=" + url.QueryEscape(target.Username)
	if counterID != 0 {
		stacksUrl += "&counter=" + strconv.Itoa(counterID)
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": tc.Renderer.T(c, "trade-menu-title", "username", target.Username),
		"User":  user,
	})
	builder.Add("trade-menu", "Content", map[string]any{
		"Target":           target.Username,
		"StacksUrl":        stacksUrl,
		"SubmitUrl":        "/trades",
		"CounterID":        counterID,
		"GiveCoinsPrefill": giveCoins,
		"WantCoinsPrefill": wantCoins,
	})
	tc.Renderer.Render(c, &builder)
}
