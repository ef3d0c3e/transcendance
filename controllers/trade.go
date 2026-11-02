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
	"gorm.io/gorm/clause"
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

// LockUsers locks the rows of the given users until the end of the
// transaction, in id order so two transactions locking the same users cannot
// deadlock. What commits or moves the cards and coins of a user takes this
// lock first, so its checks still hold when it writes
func LockUsers(tx *gorm.DB, ids ...uint) error {
	var users []models.User
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id IN ?", ids).
		Order("id").
		Find(&users).Error
}

// TradeCommittedCoins sums the coins a user put up in their pending offers
func TradeCommittedCoins(db *gorm.DB, userID uint) (uint, error) {
	var coins uint
	err := db.Model(&models.Trade{}).
		Select("COALESCE(SUM(coins), 0)").
		Where("from_id = ? AND state = ?", userID, models.TradePending).
		Scan(&coins).Error
	return coins, err
}

// FreeCoins is the part of a balance that pending offers do not hold
func FreeCoins(balance uint, committed uint) uint {
	if balance < committed {
		return 0
	}
	return balance - committed
}

// TradeBlocked reports whether either user has blocked the other: they
// cannot trade, nor reach each other with offers
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

// TradeCommitments are the assets a user has put up in the give side of
// their pending offers
type TradeCommitments struct {
	stacks map[tradeStackKey]uint
	Coins  uint
}

// Copies of a stack put up in the pending offers
func (tc *TradeCommitments) Copies(cardID uint, foil string) uint {
	return tc.stacks[tradeStackKey{CardID: cardID, Foil: foil}]
}

// Free part of a balance: what the pending offers do not hold
func (tc *TradeCommitments) Free(balance uint) uint {
	if balance < tc.Coins {
		return 0
	}
	return balance - tc.Coins
}

// TradeCommitmentsOf sums what a user put up in their pending offers,
// skipping the given trade if any: a counter replaces its contents, freeing
// what its previous version held
func TradeCommitmentsOf(db *gorm.DB, userID uint, skipTradeID uint) (*TradeCommitments, error) {
	query := db.Where("from_id = ? AND state = ?", userID, models.TradePending)
	if skipTradeID != 0 {
		query = query.Where("id <> ?", skipTradeID)
	}
	var trades []models.Trade
	if err := query.Find(&trades).Error; err != nil {
		return nil, err
	}
	commitments := TradeCommitments{stacks: make(map[tradeStackKey]uint)}
	for i := range trades {
		give, err := trades[i].GiveItems()
		if err != nil {
			return nil, err
		}
		for _, item := range give {
			commitments.stacks[tradeStackKey{CardID: item.CardID, Foil: item.Foil}] += item.Count
		}
		commitments.Coins += trades[i].Coins
	}
	return &commitments, nil
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
	// A stack is one card with one foil, whatever its count
	seen := make(map[tradeStackKey]bool, len(items))
	for _, item := range items {
		if item.Count == 0 || item.Count > tradeMaxCount {
			return tc.Renderer.T(c, "trade-error-count")
		}
		if card, _ := tc.Data.GetCard(int(item.CardID)); card == nil {
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

// Check that a user owns enough copies of every stack, and when the stacks
// are put up, that enough copies are not committed to another trade already,
// returns the message and status when not
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

// Check that an offer is well formed and that both sides own what they promise,
// in the transaction holding the locks of both sides. counterID is the offer
// being countered if any, made by the other side
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
	// What a side puts up must still be theirs to give: cards and coins up in
	// another pending offer are locked there. The countered offer is made by
	// the other side, so it is never among these
	committed, err := TradeCommitmentsOf(tx, fromID, 0)
	if err != nil {
		log.Printf("WARN failed to get commitments of user %d: %s", fromID, err)
		return tc.Renderer.T(c, "trade-error-internal"), http.StatusInternalServerError
	}
	// The wanted cards are only requested, so they are not locked by this
	// offer, but the other side can only give what their own offers leave
	// free, as when accepting. The countered offer is theirs and is replaced
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
	// The coins are checked the same way, against the free balance of each side
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
		// the user, and flips its direction. It is checked again under the
		// locks, a concurrent counter may flip it first
		var old models.Trade
		if err := tc.DB.First(&old, offer.CounterID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": tc.Renderer.T(c, "trade-error-not-found")})
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
	trade.WantCoins = offer.WantCoins
	trade.State = models.TradePending

	err = tc.DB.Transaction(func(tx *gorm.DB) error {
		// Both sides stay locked until the offer is saved, so no concurrent
		// offer, trade or purchase can take what it checks
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
			return tx.Create(&trade).Error
		}
		// Only the pending offer the user received can be overwritten by its counter
		result := tx.Model(&models.Trade{}).
			Where("id = ? AND state = ? AND from_id = ? AND to_id = ?",
				trade.ID, models.TradePending, trade.ToID, trade.FromID).
			Updates(map[string]any{
				"from_id":    trade.FromID,
				"to_id":      trade.ToID,
				"give":       trade.Give,
				"coins":      trade.Coins,
				"want":       trade.Want,
				"want_coins": trade.WantCoins,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return &tradeRejection{tc.Renderer.T(c, "trade-error-state"), http.StatusBadRequest}
		}
		return nil
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

// Move coins between two owners, keeping them from going negative
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
	if err := moveCoins(tx, trade.FromID, trade.ToID, trade.Coins); err != nil {
		return err
	}

	// The requested cards and coins leave the receiver
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
		// Read again under the locks, so it settles the offer as it is now and
		// not a version a counter has replaced since
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
		want, err := current.WantItems()
		if err != nil {
			return err
		}
		if msg, _ := tc.checkTradeOwned(c, tx, user.ID, want, committed); msg != "" {
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

		// The cards are read from the point of view of the viewer, and the
		// coins follow them: each side with the coins it put up or asks for
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

	// The partners of the three lists are loaded at once
	all := make([]models.Trade, 0, len(incoming)+len(outgoing)+len(history))
	all = append(append(append(all, incoming...), outgoing...), history...)
	names := mustPartners(tc, all, user.ID)
	fields := map[string]any{
		"Incoming": tc.tradeRows(c, user, incoming, names),
		"Outgoing": tc.tradeRows(c, user, outgoing, names),
		"History":  tc.tradeRows(c, user, history, names),
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
		// The counter starts mirrored: the cards and coins the sender offered
		// become what the user wants, and what they asked for what the user gives
		giveItems, _ := old.WantItems()
		wantItems, _ := old.GiveItems()
		give = marshalTradeItems(giveItems)
		want = marshalTradeItems(wantItems)
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
	if blocked, err := TradeBlocked(tc.DB, user.ID, target.ID); err != nil || blocked {
		tc.Renderer.RenderError(c, views.ErrorPage{
			User:       user,
			StatusCode: http.StatusForbidden,
			Message:    tc.Renderer.T(c, "trade-error-blocked"),
		})
		return
	}

	// A counter asks the stacks of the other side without the offer it
	// replaces, so its cards show as free to mirror
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
		"GivePrefill":      give,
		"WantPrefill":      want,
		"GiveCoinsPrefill": giveCoins,
		"WantCoinsPrefill": wantCoins,
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
