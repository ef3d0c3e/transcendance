package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"transcendance/models"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// POST a JSON body, optionally authenticated
func postJSON(t *testing.T, router *gin.Engine, cookie *http.Cookie, target string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// One stack of an offer
func item(cardID uint, foil string, count uint) models.TradeItem {
	return models.TradeItem{CardID: cardID, Foil: foil, Count: count}
}

// JSON body of a trade offer
func tradeBody(t *testing.T, target string, give []models.TradeItem, coins uint, want []models.TradeItem, counterOf uint) string {
	t.Helper()
	return tradeBodyWantCoins(t, target, give, coins, want, 0, counterOf)
}

// JSON body of a trade offer, coins asked along the wanted cards
func tradeBodyWantCoins(t *testing.T, target string, give []models.TradeItem, coins uint, want []models.TradeItem, wantCoins uint, counterOf uint) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"target":     target,
		"give":       give,
		"coins":      coins,
		"want":       want,
		"want_coins": wantCoins,
		"counter_of": counterOf,
	})
	require.NoError(t, err)
	return string(body)
}

// Every trade of a user, from or to them
func tradesOf(t *testing.T, username string) []models.Trade {
	t.Helper()
	var trades []models.Trade
	require.NoError(t, tests.DB.
		Where("from_id = ? OR to_id = ?", userID(t, username), userID(t, username)).
		Order("id").
		Find(&trades).
		Error)
	return trades
}

// Copies of a card stack owned by a user
func ownedCardCount(t *testing.T, username string, cardID uint, foil string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, tests.DB.Model(&models.Card{}).
		Where("owner_id = ? AND card_id = ? AND foil = ?", userID(t, username), cardID, foil).
		Count(&count).
		Error)
	return count
}

// Set the coins of a user
func setCoins(t *testing.T, username string, coins uint) {
	t.Helper()
	require.NoError(t, tests.DB.Model(&models.User{}).
		Where("username = ?", username).
		Update("coins", coins).
		Error)
}

// Run an action on a trade, e.g /trades/3/accept
func tradeAction(t *testing.T, router *gin.Engine, cookie *http.Cookie, id uint, action string) string {
	t.Helper()
	rec := postForm(t, router, cookie, "/trades/"+strconv.FormatUint(uint64(id), 10)+"/"+action, url.Values{})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	return rec.Header().Get("Location")
}

// An offer settles when its receiver accepts it
func TestTradeOfferSettlesOnAccept(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade2", "foo-bar-baz-qux", 0)
	giveCards(t, "trade1", cardFordT, 1)
	giveCards(t, "trade2", cardTungsten, 1)
	setCoins(t, "trade1", 300)

	cookie1 := login(t, router, "trade1", "foo-bar-baz-qux")
	cookie2 := login(t, router, "trade2", "foo-bar-baz-qux")

	// trade1 offers one Ford Model T and 300 coins for one Tungsten
	rec := postJSON(t, router, cookie1, "/trades", tradeBody(t, "trade2",
		[]models.TradeItem{item(cardFordT, "", 1)}, 300,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	trades := tradesOf(t, "trade1")
	require.Len(t, trades, 1)
	require.Equal(t, models.TradePending, trades[0].State)
	require.Equal(t, uint(300), trades[0].Coins)
	require.Equal(t, userID(t, "trade1"), trades[0].FromID)
	require.Equal(t, userID(t, "trade2"), trades[0].ToID)

	// The offer shows up on the trades page of its receiver
	body := getInventory(t, router, cookie2, "/trades")
	require.Contains(t, body, "trade1")
	require.Contains(t, body, "1× Ford Model T")
	require.Contains(t, body, "300 coins")
	require.Contains(t, body, "1× Tungsten")
	require.Contains(t, body, "/accept")
	require.Contains(t, body, "/decline")
	require.NotContains(t, body, "/cancel")

	// The sender only sees it as outgoing, with a cancel button
	body = getInventory(t, router, cookie1, "/trades")
	require.Contains(t, body, "trade2")
	require.Contains(t, body, "/cancel")
	require.NotContains(t, body, "/accept")

	// Accepting settles the offer
	target := tradeAction(t, router, cookie2, trades[0].ID, "accept")
	require.Equal(t, "/trades", target)

	require.Zero(t, ownedCardCount(t, "trade1", cardFordT, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "trade1", cardTungsten, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "trade2", cardFordT, ""))
	require.Zero(t, ownedCardCount(t, "trade2", cardTungsten, ""))
	require.Zero(t, shopBalance(t, "trade1"))
	require.Equal(t, uint(300), shopBalance(t, "trade2"))
	require.Equal(t, models.TradeAccepted, tradesOf(t, "trade1")[0].State)

	// The settled offer moved to the history of both sides
	body = getInventory(t, router, cookie1, "/trades")
	require.Contains(t, body, "Accepted")
	require.NotContains(t, body, "/accept")
}

// Countering an offer replaces its contents and flips its direction
func TestTradeCounterFlipsTheOffer(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade3", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade4", "foo-bar-baz-qux", 0)
	giveCards(t, "trade3", cardFordT, 1)
	giveCards(t, "trade4", cardTungsten, 1)
	setCoins(t, "trade4", 100)

	cookie3 := login(t, router, "trade3", "foo-bar-baz-qux")
	cookie4 := login(t, router, "trade4", "foo-bar-baz-qux")

	rec := postJSON(t, router, cookie3, "/trades", tradeBody(t, "trade4",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	original := tradesOf(t, "trade3")[0]

	// The counter starts from the offer, mirrored
	body := getInventory(t, router, cookie4, "/trades/new?counter="+strconv.FormatUint(uint64(original.ID), 10))
	require.Contains(t, body, "Trade with trade3")
	require.Contains(t, body, `data-counter-id="`+strconv.FormatUint(uint64(original.ID), 10)+`"`)
	require.Contains(t, body, "&#34;card_id&#34;:1005")
	require.Contains(t, body, "&#34;card_id&#34;:1004")

	// trade4 counters with one Tungsten and 100 coins for the Ford Model T
	rec = postJSON(t, router, cookie4, "/trades", tradeBody(t, "trade3",
		[]models.TradeItem{item(cardTungsten, "", 1)}, 100,
		[]models.TradeItem{item(cardFordT, "", 1)}, original.ID))
	require.Equal(t, http.StatusCreated, rec.Code)

	// The same offer now comes from the other side, with the new contents
	trades := tradesOf(t, "trade4")
	require.Len(t, trades, 1)
	require.Equal(t, original.ID, trades[0].ID)
	require.Equal(t, userID(t, "trade4"), trades[0].FromID)
	require.Equal(t, userID(t, "trade3"), trades[0].ToID)
	require.Equal(t, uint(100), trades[0].Coins)
	require.Equal(t, models.TradePending, trades[0].State)
	give, err := trades[0].GiveItems()
	require.NoError(t, err)
	require.Equal(t, []models.TradeItem{item(cardTungsten, "", 1)}, give)

	// trade3 accepts its own card back, with the coins on top
	target := tradeAction(t, router, cookie3, trades[0].ID, "accept")
	require.Equal(t, "/trades", target)
	require.Equal(t, int64(1), ownedCardCount(t, "trade3", cardTungsten, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "trade4", cardFordT, ""))
	require.Zero(t, ownedCardCount(t, "trade3", cardFordT, ""))
	require.Equal(t, uint(100), shopBalance(t, "trade3"))
	require.Zero(t, shopBalance(t, "trade4"))
}

// Declined and cancelled offers cannot be acted on anymore
func TestTradeDeclineAndCancel(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade5", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade6", "foo-bar-baz-qux", 0)
	giveCards(t, "trade5", cardFordT, 1)
	giveCards(t, "trade6", cardTungsten, 1)

	cookie5 := login(t, router, "trade5", "foo-bar-baz-qux")
	cookie6 := login(t, router, "trade6", "foo-bar-baz-qux")

	offer := tradeBody(t, "trade6",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0)
	rec := postJSON(t, router, cookie5, "/trades", offer)
	require.Equal(t, http.StatusCreated, rec.Code)
	first := tradesOf(t, "trade5")[0]

	// Declining leaves everything where it is
	target := tradeAction(t, router, cookie6, first.ID, "decline")
	require.Equal(t, "/trades", target)
	require.Equal(t, models.TradeDeclined, tradesOf(t, "trade5")[0].State)
	require.Equal(t, int64(1), ownedCardCount(t, "trade5", cardFordT, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "trade6", cardTungsten, ""))

	// A declined offer cannot be accepted anymore
	target = tradeAction(t, router, cookie6, first.ID, "accept")
	require.Equal(t, "/trades?error=settle", target)
	require.Equal(t, models.TradeDeclined, tradesOf(t, "trade5")[0].State)

	// Cancelling withdraws an offer of one's own
	rec = postJSON(t, router, cookie5, "/trades", offer)
	require.Equal(t, http.StatusCreated, rec.Code)
	second := tradesOf(t, "trade5")[1]
	target = tradeAction(t, router, cookie5, second.ID, "cancel")
	require.Equal(t, "/trades", target)
	require.Equal(t, models.TradeCancelled, tradesOf(t, "trade5")[1].State)

	target = tradeAction(t, router, cookie6, second.ID, "accept")
	require.Equal(t, "/trades?error=settle", target)
	require.Equal(t, models.TradeCancelled, tradesOf(t, "trade5")[1].State)
}

// Malformed offers are refused with the reason why
func TestTradeOfferValidation(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade7", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade8", "foo-bar-baz-qux", 0)
	giveCards(t, "trade7", cardFordT, 1)
	giveCards(t, "trade8", cardTungsten, 1)
	setCoins(t, "trade7", 100)
	cookie7 := login(t, router, "trade7", "foo-bar-baz-qux")

	cases := []struct {
		name   string
		body   string
		status int
		error  string
	}{
		{
			name:   "with oneself",
			body:   tradeBody(t, "trade7", []models.TradeItem{item(cardFordT, "", 1)}, 0, []models.TradeItem{item(cardTungsten, "", 1)}, 0),
			status: http.StatusBadRequest,
			error:  "You cannot trade with yourself",
		},
		{
			name:   "unknown user",
			body:   tradeBody(t, "nobody", []models.TradeItem{item(cardFordT, "", 1)}, 0, []models.TradeItem{item(cardTungsten, "", 1)}, 0),
			status: http.StatusNotFound,
			error:  "User not found",
		},
		{
			name:   "no wanted cards",
			body:   tradeBody(t, "trade8", []models.TradeItem{item(cardFordT, "", 1)}, 0, nil, 0),
			status: http.StatusBadRequest,
			error:  "Request at least one card or coins",
		},
		{
			name:   "nothing given",
			body:   tradeBody(t, "trade8", nil, 0, []models.TradeItem{item(cardTungsten, "", 1)}, 0),
			status: http.StatusBadRequest,
			error:  "Offer at least one card or coins",
		},
		{
			name:   "zero count",
			body:   tradeBody(t, "trade8", []models.TradeItem{item(cardFordT, "", 1)}, 0, []models.TradeItem{item(cardTungsten, "", 0)}, 0),
			status: http.StatusBadRequest,
			error:  "Invalid card count",
		},
		{
			name:   "unknown card",
			body:   tradeBody(t, "trade8", []models.TradeItem{item(cardFordT, "", 1)}, 0, []models.TradeItem{item(9999, "", 1)}, 0),
			status: http.StatusBadRequest,
			error:  "Unknown card",
		},
		{
			name: "duplicate stack",
			body: tradeBody(t, "trade8",
				[]models.TradeItem{item(cardFordT, "", 1), item(cardFordT, "", 1)}, 0,
				[]models.TradeItem{item(cardTungsten, "", 1)}, 0),
			status: http.StatusBadRequest,
			error:  "The same card stack appears twice",
		},
		{
			name:   "too many copies",
			body:   tradeBody(t, "trade8", []models.TradeItem{item(cardFordT, "", 5)}, 0, []models.TradeItem{item(cardTungsten, "", 1)}, 0),
			status: http.StatusBadRequest,
			error:  "Not enough copies of a card",
		},
		{
			name:   "too many coins",
			body:   tradeBody(t, "trade8", []models.TradeItem{item(cardFordT, "", 1)}, 1000, []models.TradeItem{item(cardTungsten, "", 1)}, 0),
			status: http.StatusBadRequest,
			error:  "Not enough coins",
		},
		{
			name:   "too many wanted coins",
			body:   tradeBodyWantCoins(t, "trade8", []models.TradeItem{item(cardFordT, "", 1)}, 0, nil, 1, 0),
			status: http.StatusBadRequest,
			error:  "The other user does not have that many coins",
		},
	}

	for _, test := range cases {
		rec := postJSON(t, router, cookie7, "/trades", test.body)
		require.Equal(t, test.status, rec.Code, test.name)
		require.Contains(t, rec.Body.String(), test.error, test.name)
	}

	// Refused offers leave nothing behind
	require.Empty(t, tradesOf(t, "trade7"))
}

// Only the two sides of an offer can act on it
func TestTradeActionsRespectSides(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade9", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade10", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade11", "foo-bar-baz-qux", 0)
	giveCards(t, "trade9", cardFordT, 1)
	giveCards(t, "trade10", cardTungsten, 1)

	cookie9 := login(t, router, "trade9", "foo-bar-baz-qux")
	cookie10 := login(t, router, "trade10", "foo-bar-baz-qux")
	cookie11 := login(t, router, "trade11", "foo-bar-baz-qux")

	rec := postJSON(t, router, cookie9, "/trades", tradeBody(t, "trade10",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	offer := tradesOf(t, "trade9")[0]

	// The sender cannot accept its own offer
	target := tradeAction(t, router, cookie9, offer.ID, "accept")
	require.Equal(t, "/trades?error=settle", target)
	// The receiver cannot withdraw it
	target = tradeAction(t, router, cookie10, offer.ID, "cancel")
	require.Equal(t, "/trades?error=settle", target)
	// A stranger can neither accept nor close it
	target = tradeAction(t, router, cookie11, offer.ID, "accept")
	require.Equal(t, "/trades?error=settle", target)
	target = tradeAction(t, router, cookie11, offer.ID, "decline")
	require.Equal(t, "/trades?error=settle", target)
	require.Equal(t, models.TradePending, tradesOf(t, "trade9")[0].State)

	// The sender cannot counter their own offer either
	rec = postJSON(t, router, cookie9, "/trades", tradeBody(t, "trade10",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		[]models.TradeItem{item(cardTungsten, "", 1)}, offer.ID))
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "This trade offer is not yours")
}

// An offer cannot settle when its assets are gone, and rolls back whole
func TestTradeAcceptFailsWithoutTheAssets(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade12", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade13", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade14", "foo-bar-baz-qux", 0)
	giveCards(t, "trade12", cardFordT, 1)
	giveCards(t, "trade13", cardTungsten, 1)
	setCoins(t, "trade12", 300)

	cookie12 := login(t, router, "trade12", "foo-bar-baz-qux")
	cookie13 := login(t, router, "trade13", "foo-bar-baz-qux")

	// The offered coins get spent while the offer is pending
	rec := postJSON(t, router, cookie12, "/trades", tradeBody(t, "trade13",
		[]models.TradeItem{item(cardFordT, "", 1)}, 300,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	first := tradesOf(t, "trade12")[0]
	setCoins(t, "trade12", 0)

	target := tradeAction(t, router, cookie13, first.ID, "accept")
	require.Equal(t, "/trades?error=settle", target)
	require.Equal(t, models.TradePending, tradesOf(t, "trade12")[0].State)
	// Nothing moved, not even the cards offered along the missing coins
	require.Equal(t, int64(1), ownedCardCount(t, "trade12", cardFordT, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "trade13", cardTungsten, ""))
	require.Zero(t, shopBalance(t, "trade13"))

	// Withdrawing the offer gives its assets back: the Ford Model T can be
	// put up again, and the requested cards get traded away while the new
	// offer is pending
	target = tradeAction(t, router, cookie12, first.ID, "cancel")
	require.Equal(t, "/trades", target)
	rec = postJSON(t, router, cookie12, "/trades", tradeBody(t, "trade13",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	second := tradesOf(t, "trade12")[1]
	require.NoError(t, tests.DB.Model(&models.Card{}).
		Where("owner_id = ? AND card_id = ?", userID(t, "trade13"), cardTungsten).
		Update("owner_id", userID(t, "trade14")).
		Error)

	target = tradeAction(t, router, cookie13, second.ID, "accept")
	require.Equal(t, "/trades?error=settle", target)
	require.Equal(t, models.TradePending, tradesOf(t, "trade12")[1].State)
	require.Equal(t, int64(1), ownedCardCount(t, "trade12", cardFordT, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "trade14", cardTungsten, ""))
}

// The trade menu opens on a viewed collection, the stacks come from its API
func TestTradeMenuAndStacks(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade15", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade16", "foo-bar-baz-qux", 0)
	giveCards(t, "trade15", cardFordT, 2)
	giveCardAt(t, "trade15", cardTungsten, "foil3", time.Now())
	giveCards(t, "trade16", cardMcMurdo, 1)
	setCoins(t, "trade15", 55)

	cookie15 := login(t, router, "trade15", "foo-bar-baz-qux")

	// The menu of another user's collection
	body := getInventory(t, router, cookie15, "/trades/new?u=trade16")
	require.Contains(t, body, `id="trade-menu"`)
	require.Contains(t, body, "Trade with trade16")
	require.Contains(t, body, `data-stacks-url="/api/trade/stacks?u=trade16"`)

	// No menu for oneself or for a user that does not exist
	rec := get(t, router, cookie15, "/trades/new?u=trade15")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "You cannot trade with yourself")
	rec = get(t, router, cookie15, "/trades/new?u=nobody")
	require.Equal(t, http.StatusNotFound, rec.Code)

	// The stacks of both sides, with the fields the menu shows
	rec = get(t, router, cookie15, "/api/trade/stacks?u=trade16")
	require.Equal(t, http.StatusOK, rec.Code)
	var stacks struct {
		Coins  uint `json:"coins"`
		Mine   []stackJSON
		Theirs []stackJSON
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stacks))
	require.Equal(t, uint(55), stacks.Coins)
	require.Len(t, stacks.Mine, 2)
	// Foils fold on their own
	require.Equal(t, uint(cardFordT), stacks.Mine[0].CardID)
	require.Equal(t, uint(2), stacks.Mine[0].Count)
	require.Equal(t, "Ford Model T", stacks.Mine[0].Title)
	require.Equal(t, uint(cardTungsten), stacks.Mine[1].CardID)
	require.Equal(t, "foil3", stacks.Mine[1].Foil)
	require.Equal(t, uint(1), stacks.Mine[1].Count)
	require.Len(t, stacks.Theirs, 1)
	require.Equal(t, uint(cardMcMurdo), stacks.Theirs[0].CardID)
	require.Equal(t, "McMurdo Station", stacks.Theirs[0].Title)
	require.NotEmpty(t, stacks.Theirs[0].Image)

	// Every stack brings its card render along, foils wear their effect
	require.Contains(t, stacks.Mine[0].HTML, `class="card-container"`)
	require.Contains(t, stacks.Mine[0].HTML, "card-stack")
	require.Contains(t, stacks.Mine[1].HTML, "card-foil")

	// The stacks of oneself are not a trade
	rec = get(t, router, cookie15, "/api/trade/stacks?u=trade15")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// Fields of one stack in the reply of the trade stacks API
type stackJSON struct {
	CardID uint   `json:"card_id"`
	Foil   string `json:"foil"`
	Count  uint   `json:"count"`
	Title  string `json:"title"`
	Rarity string `json:"rarity"`
	Image  string `json:"image"`
	HTML   string `json:"html"`
}

// GET a page as the authenticated user, without following redirects
func get(t *testing.T, router *gin.Engine, cookie *http.Cookie, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// The trade button shows on the collection of another user only
func TestTradeButtonOnCollection(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade17", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade18", "foo-bar-baz-qux", 0)
	cookie17 := login(t, router, "trade17", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie17, "/inventory?u=trade18")
	require.Contains(t, body, `class="inventory-trade"`)
	require.Contains(t, body, `href="/trades/new?u=trade18"`)

	// One's own collection has nobody to trade with
	body = getInventory(t, router, cookie17, "/inventory")
	require.NotContains(t, body, "inventory-trade")
}

// Trades are listed for their two sides only, and erased with their users
func TestTradeVisibilityAndErasure(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade19", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade20", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade21", "foo-bar-baz-qux", 0)
	giveCards(t, "trade19", cardFordT, 1)
	giveCards(t, "trade20", cardTungsten, 1)

	cookie19 := login(t, router, "trade19", "foo-bar-baz-qux")
	cookie20 := login(t, router, "trade20", "foo-bar-baz-qux")
	cookie21 := login(t, router, "trade21", "foo-bar-baz-qux")

	rec := postJSON(t, router, cookie19, "/trades", tradeBody(t, "trade20",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	// A stranger sees an empty trades page
	body := getInventory(t, router, cookie21, "/trades")
	require.Contains(t, body, "No incoming offers")
	require.NotContains(t, body, "trade19")
	require.NotContains(t, body, "trade20")

	// Erasing a user erases their offers
	rec = postForm(t, router, cookie20, fmt.Sprintf("/profiles/%d/delete", userID(t, "trade20")), url.Values{})
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Empty(t, tradesOf(t, "trade19"))
}

// The trade pages ask for an account
func TestTradeRequiresLogin(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := get(t, router, nil, "/trades")
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/login", rec.Header().Get("Location"))

	rec = get(t, router, nil, "/trades/new?u=someone")
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/login", rec.Header().Get("Location"))

	rec = postJSON(t, router, nil, "/trades", "{}")
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = get(t, router, nil, "/api/trade/stacks?u=someone")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// An offer and a counter notify their receiver
func TestTradeOfferNotifies(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade22", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade23", "foo-bar-baz-qux", 0)
	giveCards(t, "trade22", cardFordT, 1)
	giveCards(t, "trade23", cardTungsten, 1)

	cookie22 := login(t, router, "trade22", "foo-bar-baz-qux")
	cookie23 := login(t, router, "trade23", "foo-bar-baz-qux")

	rec := postJSON(t, router, cookie22, "/trades", tradeBody(t, "trade23",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	// The receiver reads the offer on their notifications page
	body := getInventory(t, router, cookie23, "/notifications")
	require.Contains(t, body, "trade22 sent you a trade offer")

	var offerNotif models.Notification
	require.NoError(t, tests.DB.
		Where("user_id = ?", userID(t, "trade23")).
		Order("id").
		Last(&offerNotif).
		Error)
	require.Equal(t, models.NotificationTrade, offerNotif.Type)
	require.Equal(t, "/trades", offerNotif.Action)

	// Countering notifies the original sender
	offer := tradesOf(t, "trade22")[0]
	rec = postJSON(t, router, cookie23, "/trades", tradeBody(t, "trade22",
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0,
		[]models.TradeItem{item(cardFordT, "", 1)}, offer.ID))
	require.Equal(t, http.StatusCreated, rec.Code)

	body = getInventory(t, router, cookie22, "/notifications")
	require.Contains(t, body, "trade23 countered your trade offer")

	var counterNotif models.Notification
	require.NoError(t, tests.DB.
		Where("user_id = ?", userID(t, "trade22")).
		Order("id").
		Last(&counterNotif).
		Error)
	require.Equal(t, models.NotificationTradeCounter, counterNotif.Type)
	require.Equal(t, "/trades", counterNotif.Action)
}

// Cards and coins up in an offer stay locked until it is settled or withdrawn
func TestTradeAssetsStayLocked(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade24", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade25", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade26", "foo-bar-baz-qux", 0)
	giveCards(t, "trade24", cardFordT, 2)
	giveCards(t, "trade25", cardTungsten, 1)
	giveCards(t, "trade26", cardMcMurdo, 1)
	setCoins(t, "trade24", 100)

	cookie24 := login(t, router, "trade24", "foo-bar-baz-qux")
	cookie25 := login(t, router, "trade25", "foo-bar-baz-qux")

	// One of the two Ford Model T and 40 of the 100 coins go up in an offer
	rec := postJSON(t, router, cookie24, "/trades", tradeBody(t, "trade25",
		[]models.TradeItem{item(cardFordT, "", 1)}, 40,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	// The copies and coins up in that offer cannot be put up in another one
	rec = postJSON(t, router, cookie24, "/trades", tradeBody(t, "trade26",
		[]models.TradeItem{item(cardFordT, "", 2)}, 0,
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Some copies of a card are up in another trade")

	rec = postJSON(t, router, cookie24, "/trades", tradeBody(t, "trade26",
		[]models.TradeItem{item(cardFordT, "", 1)}, 70,
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Some coins are up in another trade")

	// What is missing altogether is still said so
	rec = postJSON(t, router, cookie24, "/trades", tradeBody(t, "trade26",
		[]models.TradeItem{item(cardFordT, "", 3)}, 0,
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Not enough copies of a card")

	rec = postJSON(t, router, cookie24, "/trades", tradeBody(t, "trade26",
		[]models.TradeItem{item(cardFordT, "", 1)}, 200,
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Not enough coins")

	// Refused offers leave nothing behind
	require.Len(t, tradesOf(t, "trade24"), 1)

	// What is still free can be put up in a second offer
	rec = postJSON(t, router, cookie24, "/trades", tradeBody(t, "trade26",
		[]models.TradeItem{item(cardFordT, "", 1)}, 60,
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	// Declining gives the assets of the offer back
	target := tradeAction(t, router, cookie25, tradesOf(t, "trade24")[0].ID, "decline")
	require.Equal(t, "/trades", target)
	rec = postJSON(t, router, cookie24, "/trades", tradeBody(t, "trade25",
		[]models.TradeItem{item(cardFordT, "", 1)}, 40,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
}

// The shop spends the coins that are not up in trades only
func TestTradeShopSpendsFreeCoinsOnly(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade27", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade28", "foo-bar-baz-qux", 0)
	giveCards(t, "trade27", cardFordT, 1)
	giveCards(t, "trade28", cardTungsten, 1)
	setCoins(t, "trade27", 305)

	cookie27 := login(t, router, "trade27", "foo-bar-baz-qux")

	// 200 of the 305 coins go up in an offer
	rec := postJSON(t, router, cookie27, "/trades", tradeBody(t, "trade28",
		[]models.TradeItem{item(cardFordT, "", 1)}, 200,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	offer := tradesOf(t, "trade27")[0]

	// The shop shows what is free to spend and what is up in trades
	body := shopBody(shopRequest(t, router, http.MethodGet, "/shop", nil, cookie27))
	require.Contains(t, body, "105 coins")
	require.Contains(t, body, "200 up in trades")

	// A booster cannot eat into the coins of the offer
	rec = shopRequest(t, router, http.MethodPost, "/shop/buy", url.Values{
		"collection": {"1"},
	}, cookie27)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/shop?error=funds", rec.Header().Get("Location"))
	require.Equal(t, uint(305), shopBalance(t, "trade27"))
	require.Empty(t, shopBoosters(t, "trade27"))

	// Withdrawing the offer frees its coins for the shop again
	target := tradeAction(t, router, cookie27, offer.ID, "cancel")
	require.Equal(t, "/trades", target)
	rec = shopRequest(t, router, http.MethodPost, "/shop/buy", url.Values{
		"collection": {"1"},
	}, cookie27)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/shop?bought=1", rec.Header().Get("Location"))
	require.Equal(t, uint(55), shopBalance(t, "trade27"))
	require.Len(t, shopBoosters(t, "trade27"), 1)
}

// A counter starts mirrored, the coins of the other side included
func TestTradeCounterMirrorsCoins(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade29", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade30", "foo-bar-baz-qux", 0)
	giveCards(t, "trade29", cardFordT, 1)
	giveCards(t, "trade30", cardTungsten, 1)
	setCoins(t, "trade29", 300)
	setCoins(t, "trade30", 200)

	cookie29 := login(t, router, "trade29", "foo-bar-baz-qux")
	cookie30 := login(t, router, "trade30", "foo-bar-baz-qux")

	// A Ford Model T and 100 coins for a Tungsten and 50 coins
	rec := postJSON(t, router, cookie29, "/trades", tradeBodyWantCoins(t, "trade30",
		[]models.TradeItem{item(cardFordT, "", 1)}, 100,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 50, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	original := tradesOf(t, "trade29")[0]

	// Each side of the offer shows its own coins
	body := getInventory(t, router, cookie29, "/trades")
	require.Contains(t, body, "100 coins")
	require.Contains(t, body, "50 coins")

	// The counter starts from the offer mirrored, coins included
	body = getInventory(t, router, cookie30, "/trades/new?counter="+strconv.FormatUint(uint64(original.ID), 10))
	require.Contains(t, body, `data-prefill-give-coins="50"`)
	require.Contains(t, body, `data-prefill-want-coins="100"`)

	// trade30 counters with the very same deal, direction flipped
	rec = postJSON(t, router, cookie30, "/trades", tradeBodyWantCoins(t, "trade29",
		[]models.TradeItem{item(cardTungsten, "", 1)}, 50,
		[]models.TradeItem{item(cardFordT, "", 1)}, 100, original.ID))
	require.Equal(t, http.StatusCreated, rec.Code)

	trades := tradesOf(t, "trade30")
	require.Len(t, trades, 1)
	require.Equal(t, uint(50), trades[0].Coins)
	require.Equal(t, uint(100), trades[0].WantCoins)

	// Accepting moves the cards and coins of both sides
	target := tradeAction(t, router, cookie29, trades[0].ID, "accept")
	require.Equal(t, "/trades", target)
	require.Zero(t, ownedCardCount(t, "trade29", cardFordT, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "trade29", cardTungsten, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "trade30", cardFordT, ""))
	require.Zero(t, ownedCardCount(t, "trade30", cardTungsten, ""))
	require.Equal(t, uint(250), shopBalance(t, "trade29"))
	require.Equal(t, uint(250), shopBalance(t, "trade30"))
}

// Coins alone can be asked for and offered
func TestTradeWantsCoins(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade31", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade32", "foo-bar-baz-qux", 0)
	giveCards(t, "trade31", cardFordT, 1)
	setCoins(t, "trade32", 150)

	cookie31 := login(t, router, "trade31", "foo-bar-baz-qux")
	cookie32 := login(t, router, "trade32", "foo-bar-baz-qux")

	// A card is offered against coins alone
	rec := postJSON(t, router, cookie31, "/trades", tradeBodyWantCoins(t, "trade32",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		nil, 150, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	target := tradeAction(t, router, cookie32, tradesOf(t, "trade31")[0].ID, "accept")
	require.Equal(t, "/trades", target)
	require.Zero(t, ownedCardCount(t, "trade31", cardFordT, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "trade32", cardFordT, ""))
	require.Equal(t, uint(150), shopBalance(t, "trade31"))
	require.Zero(t, shopBalance(t, "trade32"))

	// Coins alone can be put up just the same
	rec = postJSON(t, router, cookie31, "/trades", tradeBody(t, "trade32",
		nil, 100,
		[]models.TradeItem{item(cardFordT, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	target = tradeAction(t, router, cookie31, tradesOf(t, "trade31")[1].ID, "cancel")
	require.Equal(t, "/trades", target)
}

// The stacks of the trade menu show what is free to put up, not what is locked
func TestTradeStacksHideLocked(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade33", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade34", "foo-bar-baz-qux", 0)
	giveCards(t, "trade33", cardFordT, 2)
	giveCards(t, "trade34", cardMcMurdo, 1)
	setCoins(t, "trade33", 50)
	setCoins(t, "trade34", 20)

	cookie33 := login(t, router, "trade33", "foo-bar-baz-qux")
	cookie34 := login(t, router, "trade34", "foo-bar-baz-qux")

	// One of the two Ford Model T and 20 of the 50 coins go up in an offer
	rec := postJSON(t, router, cookie33, "/trades", tradeBody(t, "trade34",
		[]models.TradeItem{item(cardFordT, "", 1)}, 20,
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	offer := tradesOf(t, "trade33")[0]

	var stacks struct {
		Coins      uint        `json:"coins"`
		TheirCoins uint        `json:"their_coins"`
		Mine       []stackJSON `json:"mine"`
		Theirs     []stackJSON `json:"theirs"`
	}

	// The locked copies and coins stay out of the stacks of their owner
	rec = get(t, router, cookie33, "/api/trade/stacks?u=trade34")
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stacks))
	require.Equal(t, uint(30), stacks.Coins)
	require.Equal(t, uint(20), stacks.TheirCoins)
	require.Equal(t, uint(1), stacks.Mine[0].Count)
	require.Equal(t, uint(1), stacks.Theirs[0].Count)

	// A counter sees the contents of the offer free again, to mirror it
	rec = get(t, router, cookie34, "/api/trade/stacks?u=trade33&counter="+strconv.FormatUint(uint64(offer.ID), 10))
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stacks))
	require.Equal(t, uint(20), stacks.Coins)
	require.Equal(t, uint(50), stacks.TheirCoins)
	require.Equal(t, uint(2), stacks.Theirs[0].Count)

	// Without the counter, the locked copies stay out
	rec = get(t, router, cookie34, "/api/trade/stacks?u=trade33")
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stacks))
	require.Equal(t, uint(30), stacks.TheirCoins)
	require.Equal(t, uint(1), stacks.Theirs[0].Count)

	// Only a counterable offer unlocks its side
	rec = get(t, router, cookie34, "/api/trade/stacks?u=trade33&counter=99999")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	rec = get(t, router, cookie33, "/api/trade/stacks?u=trade34&counter="+strconv.FormatUint(uint64(offer.ID), 10))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// The inventory marks the copies up in trades on one's own cards only
func TestTradeInventoryMarksLocked(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade35", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade36", "foo-bar-baz-qux", 0)
	giveCards(t, "trade35", cardFordT, 2)
	giveCards(t, "trade36", cardTungsten, 1)

	cookie35 := login(t, router, "trade35", "foo-bar-baz-qux")
	cookie36 := login(t, router, "trade36", "foo-bar-baz-qux")

	// One of the two Ford Model T goes up in an offer
	rec := postJSON(t, router, cookie35, "/trades", tradeBody(t, "trade36",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	body := getInventory(t, router, cookie35, "/inventory")
	require.Contains(t, body, `<span class="card-locked">1 in trades</span>`)

	// The cards of another user keep their trades to themselves
	body = getInventory(t, router, cookie36, "/inventory?u=trade35")
	require.NotContains(t, body, "in trades")
}

// Accepting cannot give away what the receiver put up in their own offers
func TestTradeAcceptRespectsReceiverLocks(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "trade37", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade38", "foo-bar-baz-qux", 0)
	createAccount(t, router, "trade39", "foo-bar-baz-qux", 0)
	giveCards(t, "trade37", cardFordT, 1)
	giveCards(t, "trade38", cardTungsten, 1)
	giveCards(t, "trade39", cardMcMurdo, 2)
	setCoins(t, "trade37", 100)

	cookie37 := login(t, router, "trade37", "foo-bar-baz-qux")
	cookie39 := login(t, router, "trade39", "foo-bar-baz-qux")

	// The Ford Model T and 80 coins go up in an offer to trade38
	rec := postJSON(t, router, cookie37, "/trades", tradeBody(t, "trade38",
		[]models.TradeItem{item(cardFordT, "", 1)}, 80,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	// trade39 asks for that Ford Model T, then for those coins
	rec = postJSON(t, router, cookie39, "/trades", tradeBody(t, "trade37",
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0,
		[]models.TradeItem{item(cardFordT, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	rec = postJSON(t, router, cookie39, "/trades", tradeBodyWantCoins(t, "trade37",
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0,
		nil, 50, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	trades := tradesOf(t, "trade39")
	require.Len(t, trades, 2)
	for _, trade := range trades {
		target := tradeAction(t, router, cookie37, trade.ID, "accept")
		require.Equal(t, "/trades?error=settle", target)
	}

	// Nothing moved, and every offer is still pending
	require.Equal(t, int64(1), ownedCardCount(t, "trade37", cardFordT, ""))
	require.Equal(t, int64(2), ownedCardCount(t, "trade39", cardMcMurdo, ""))
	require.Equal(t, uint(100), shopBalance(t, "trade37"))
	for _, trade := range tradesOf(t, "trade37") {
		require.Equal(t, models.TradePending, trade.State)
	}
}
