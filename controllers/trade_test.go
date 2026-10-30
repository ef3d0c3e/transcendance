package controllers_test

import (
	"encoding/json"
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
	body, err := json.Marshal(map[string]any{
		"target":     target,
		"give":       give,
		"coins":      coins,
		"want":       want,
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
			error:  "Request at least one card",
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

	// The requested cards get traded away while the offer is pending
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
	rec = postForm(t, router, cookie20, "/api/account_delete", url.Values{})
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
