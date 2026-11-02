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

func sell(t *testing.T, router *gin.Engine, cookie *http.Cookie, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return postForm(t, router, cookie, "/shop/sell", form)
}

func sellForm(cardID uint, foil string, quantity int) url.Values {
	return url.Values{
		"card":     {strconv.FormatUint(uint64(cardID), 10)},
		"foil":     {foil},
		"quantity": {strconv.Itoa(quantity)},
	}
}

func sellDuplicates(t *testing.T, router *gin.Engine, cookie *http.Cookie) string {
	t.Helper()
	rec := postForm(t, router, cookie, "/shop/sell-duplicates", url.Values{})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	return rec.Header().Get("Location")
}

// sellInPlace sells as the pages that stay put do, getting JSON back
func sellInPlace(t *testing.T, router *gin.Engine, cookie *http.Cookie, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// Mirrors saleOutcome in sell.go
type saleOutcome struct {
	Coins      uint   `json:"coins"`
	CoinsLabel string `json:"coinsLabel"`
	NavBalance string `json:"navBalance"`
	Message    string `json:"message"`
	Stacks     []struct {
		Card         uint   `json:"card"`
		Foil         string `json:"foil"`
		Copies       uint   `json:"copies"`
		Owned        uint   `json:"owned"`
		AcquiredLast string `json:"acquiredLast"`
	} `json:"stacks"`
	Duplicates struct {
		Count   uint   `json:"count"`
		Coins   uint   `json:"coins"`
		Text    string `json:"text"`
		Confirm string `json:"confirm"`
	} `json:"duplicates"`
}

type saleFailure struct {
	Error string `json:"error"`
}

func TestSellPaysTheFixedPriceWhateverTheQuantity(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell1", "foo-bar-baz-qux", 0)
	giveCards(t, "sell1", cardRosetta, 5)
	cookie := login(t, router, "sell1", "foo-bar-baz-qux")

	// Two common cards are worth 20 coins
	rec := sell(t, router, cookie, sellForm(cardRosetta, "", 2))
	require.Equal(t, http.StatusSeeOther, rec.Code)
	requireSignedNotice(t, "/inventory?tab=cards&sold=2&coins=20", rec.Header().Get("Location"))
	require.Equal(t, uint(20), shopBalance(t, "sell1"))

	rec = sell(t, router, cookie, sellForm(cardRosetta, "", 3))
	requireSignedNotice(t, "/inventory?tab=cards&sold=3&coins=30", rec.Header().Get("Location"))
	require.Equal(t, uint(50), shopBalance(t, "sell1"))
	require.Empty(t, ownedCards(t, "sell1"))

	body := getInventory(t, router, cookie, rec.Header().Get("Location"))
	require.Contains(t, body, "You sold 3 cards for 30 coins")

	// A link claiming a sale that never happened shows nothing, nor someone else's notice
	require.NotContains(t, getInventory(t, router, cookie, "/inventory?tab=cards&sold=999&coins=999999"), "You sold")
	createAccount(t, router, "sell1b", "foo-bar-baz-qux", 0)
	require.NotContains(t, getInventory(t, router, login(t, router, "sell1b", "foo-bar-baz-qux"), rec.Header().Get("Location")), "You sold")
}

// A success notice is the target of the redirect plus its signature
func requireSignedNotice(t *testing.T, want string, location string, msgAndArgs ...any) {
	t.Helper()
	require.True(t, strings.HasPrefix(location, want+"&sig="), append([]any{"%s does not sign %s", location, want}, msgAndArgs...)...)
}

func TestSellFoilIsWorthDouble(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell2", "foo-bar-baz-qux", 0)
	giveCards(t, "sell2", cardRosetta, 2)
	giveCardAt(t, "sell2", cardRosetta, "foil1", time.Now())
	giveCardAt(t, "sell2", cardRosetta, "foil1", time.Now())
	cookie := login(t, router, "sell2", "foo-bar-baz-qux")

	// The foils are worth 20 each, only the plain copies stay behind
	rec := sell(t, router, cookie, sellForm(cardRosetta, "foil1", 2))
	requireSignedNotice(t, "/inventory?tab=cards&sold=2&coins=40", rec.Header().Get("Location"))
	owned := ownedCards(t, "sell2")
	require.Len(t, owned, 2)
	for _, card := range owned {
		require.Empty(t, card.Foil)
	}
}

func TestSellPopupData(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell3", "foo-bar-baz-qux", 0)
	createAccount(t, router, "sell4", "foo-bar-baz-qux", 0)
	giveCards(t, "sell3", cardRosetta, 1)
	giveCards(t, "sell3", cardMcMurdo, 1)
	giveCards(t, "sell3", cardFordT, 1)
	cookie3 := login(t, router, "sell3", "foo-bar-baz-qux")
	cookie4 := login(t, router, "sell4", "foo-bar-baz-qux")

	// Common, rare and legendary, one tile each
	body := getInventory(t, router, cookie3, "/inventory?tab=cards")
	require.Contains(t, body, `data-sell-price="10"`)
	require.Contains(t, body, `data-sell-price="50"`)
	require.Contains(t, body, `data-sell-price="250"`)
	require.Contains(t, body, `data-sell-action="/shop/sell"`)

	// The cards of someone else cannot be sold
	body = getInventory(t, router, cookie4, "/inventory?tab=cards&u=sell3")
	require.NotContains(t, body, "data-sell-")
}

func TestSellRejectsBadSales(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell5", "foo-bar-baz-qux", 0)
	giveCards(t, "sell5", cardRosetta, 3)
	cookie := login(t, router, "sell5", "foo-bar-baz-qux")

	cases := map[string]url.Values{
		"/inventory?tab=cards&error=sell-quantity": sellForm(cardRosetta, "", 0),
		"/inventory?tab=cards&error=sell-missing":  sellForm(cardRosetta, "", 4),
		"/inventory?tab=cards&error=sell-card":     sellForm(9999, "", 1),
	}
	for target, form := range cases {
		rec := sell(t, router, cookie, form)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		require.Equal(t, target, rec.Header().Get("Location"))
	}
	require.Len(t, ownedCards(t, "sell5"), 3)
	require.Zero(t, shopBalance(t, "sell5"))
	require.Contains(t, getInventory(t, router, cookie, "/inventory?tab=cards&error=sell-missing"),
		"You do not have that many copies to sell")
}

func TestSellKeepsCopiesUpInTrades(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell6", "foo-bar-baz-qux", 0)
	createAccount(t, router, "sell7", "foo-bar-baz-qux", 0)
	giveCards(t, "sell6", cardRosetta, 3)
	giveCards(t, "sell7", cardMcMurdo, 1)
	cookie := login(t, router, "sell6", "foo-bar-baz-qux")

	rec := postJSON(t, router, cookie, "/trades", tradeBody(t, "sell7",
		[]models.TradeItem{item(cardRosetta, "", 2)}, 0,
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	// The popup only offers the free copy
	require.Contains(t, getInventory(t, router, cookie, "/inventory?tab=cards"), `data-sell-owned="1"`)

	rec = sell(t, router, cookie, sellForm(cardRosetta, "", 2))
	require.Equal(t, "/inventory?tab=cards&error=sell-locked", rec.Header().Get("Location"))
	rec = sell(t, router, cookie, sellForm(cardRosetta, "", 1))
	requireSignedNotice(t, "/inventory?tab=cards&sold=1&coins=10", rec.Header().Get("Location"))

	// The trade can still be settled
	target := tradeAction(t, router, login(t, router, "sell7", "foo-bar-baz-qux"), tradesOf(t, "sell6")[0].ID, "accept")
	require.Equal(t, "/trades", target)
	require.Equal(t, int64(2), ownedCardCount(t, "sell7", cardRosetta, ""))
}

func TestSellDuplicates(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell8", "foo-bar-baz-qux", 0)
	createAccount(t, router, "sell9", "foo-bar-baz-qux", 0)
	// 3 common, 2 rare, 2 foil rare, and a single legendary
	first := time.Now().Add(-time.Hour)
	giveCardAt(t, "sell8", cardRosetta, "", first)
	giveCards(t, "sell8", cardRosetta, 2)
	giveCards(t, "sell8", cardMcMurdo, 2)
	giveCardAt(t, "sell8", cardMcMurdo, "foil1", time.Now())
	giveCardAt(t, "sell8", cardMcMurdo, "foil1", time.Now())
	giveCards(t, "sell8", cardFordT, 1)
	giveCards(t, "sell9", cardMcMurdo, 1)
	cookie := login(t, router, "sell8", "foo-bar-baz-qux")

	// One plain rare is up in a trade, so the other is the one kept: 2 commons
	// and 1 foil rare are sold
	rec := postJSON(t, router, cookie, "/trades", tradeBody(t, "sell9",
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0,
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	// The popup says what the sale pays before it is made
	body := getInventory(t, router, cookie, "/inventory?tab=cards")
	require.Contains(t, body, `class="inventory-sell-duplicates"`)
	require.Contains(t, body, "Sell 3 duplicate cards for 120 coins?")
	require.Contains(t, body, "Sell for 120 coins")

	requireSignedNotice(t, "/inventory?tab=cards&sold=3&coins=120", sellDuplicates(t, router, cookie))
	require.Equal(t, uint(120), shopBalance(t, "sell8"))
	require.Equal(t, int64(1), ownedCardCount(t, "sell8", cardRosetta, ""))
	require.Equal(t, int64(2), ownedCardCount(t, "sell8", cardMcMurdo, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "sell8", cardMcMurdo, "foil1"))
	require.Equal(t, int64(1), ownedCardCount(t, "sell8", cardFordT, ""))
	// The oldest copy is the one kept
	var kept models.Card
	require.NoError(t, tests.DB.Where("owner_id = ? AND card_id = ?", userID(t, "sell8"), cardRosetta).First(&kept).Error)
	require.WithinDuration(t, first, kept.Acquired, time.Second)

	// Nothing left to sell: no button, and selling again says so
	require.NotContains(t, getInventory(t, router, cookie, "/inventory?tab=cards"), "inventory-sell-duplicates")
	require.Equal(t, "/inventory?tab=cards&error=sell-none", sellDuplicates(t, router, cookie))
	require.Equal(t, uint(120), shopBalance(t, "sell8"))
}

func TestSellInPlaceReportsWhatChanged(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell10", "foo-bar-baz-qux", 0)
	giveCards(t, "sell10", cardRosetta, 3)
	giveCards(t, "sell10", cardMcMurdo, 2)
	cookie := login(t, router, "sell10", "foo-bar-baz-qux")

	// Two common cards are worth 20 coins
	rec := sellInPlace(t, router, cookie, "/shop/sell", sellForm(cardRosetta, "", 2))
	require.Equal(t, http.StatusOK, rec.Code)
	var outcome saleOutcome
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &outcome))
	require.Equal(t, uint(20), outcome.Coins)
	require.Equal(t, "20 coins", stripIsolation(outcome.CoinsLabel))
	require.Equal(t, "Balance: 20", stripIsolation(outcome.NavBalance))
	require.Equal(t, "You sold 2 cards for 20 coins", stripIsolation(outcome.Message))
	require.Equal(t, uint(20), shopBalance(t, "sell10"))

	// The tile of the stack is left with its one copy
	require.Len(t, outcome.Stacks, 1)
	require.Equal(t, uint(cardRosetta), outcome.Stacks[0].Card)
	require.Empty(t, outcome.Stacks[0].Foil)
	require.Equal(t, uint(1), outcome.Stacks[0].Copies)
	require.Equal(t, uint(1), outcome.Stacks[0].Owned)
	_, err := time.Parse(time.RFC3339, outcome.Stacks[0].AcquiredLast)
	require.NoError(t, err)

	// The rare duplicate left is still for sale, said by the dialog
	require.Equal(t, uint(1), outcome.Duplicates.Count)
	require.Equal(t, uint(50), outcome.Duplicates.Coins)
	require.Equal(t, "Sell 1 duplicate card for 50 coins?", stripIsolation(outcome.Duplicates.Text))
	require.Equal(t, "Sell for 50 coins", stripIsolation(outcome.Duplicates.Confirm))

	// Taking the last copy empties the stack
	rec = sellInPlace(t, router, cookie, "/shop/sell", sellForm(cardRosetta, "", 1))
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &outcome))
	require.Equal(t, "You sold 1 card for 10 coins", stripIsolation(outcome.Message))
	require.Equal(t, uint(30), outcome.Coins)
	require.Zero(t, outcome.Stacks[0].Copies)
	require.Zero(t, outcome.Stacks[0].Owned)
	require.Empty(t, outcome.Stacks[0].AcquiredLast)
	require.Len(t, ownedCards(t, "sell10"), 2)
}

func TestSellInPlaceSaysWhyItFailed(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell11", "foo-bar-baz-qux", 0)
	giveCards(t, "sell11", cardRosetta, 3)

	// Logged out, there is nothing to sell
	rec := sellInPlace(t, router, nil, "/shop/sell", sellForm(cardRosetta, "", 1))
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	cookie := login(t, router, "sell11", "foo-bar-baz-qux")
	cases := map[string]url.Values{
		"Choose a quantity of at least 1":          sellForm(cardRosetta, "", 0),
		"You do not have that many copies to sell": sellForm(cardRosetta, "", 4),
		"This card cannot be sold":                 sellForm(9999, "", 1),
	}
	for message, form := range cases {
		rec := sellInPlace(t, router, cookie, "/shop/sell", form)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		var failure saleFailure
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &failure))
		require.Equal(t, message, stripIsolation(failure.Error))
	}
	require.Len(t, ownedCards(t, "sell11"), 3)
	require.Zero(t, shopBalance(t, "sell11"))
}

// Owned leaves out the copies up in trades, Copies counts them
func TestSellInPlaceKeepsTheCopiesUpInTrades(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell12", "foo-bar-baz-qux", 0)
	createAccount(t, router, "sell13", "foo-bar-baz-qux", 0)
	giveCards(t, "sell12", cardRosetta, 3)
	giveCards(t, "sell13", cardMcMurdo, 1)
	cookie := login(t, router, "sell12", "foo-bar-baz-qux")

	rec := postJSON(t, router, cookie, "/trades", tradeBody(t, "sell13",
		[]models.TradeItem{item(cardRosetta, "", 2)}, 0,
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = sellInPlace(t, router, cookie, "/shop/sell", sellForm(cardRosetta, "", 1))
	require.Equal(t, http.StatusOK, rec.Code)
	var outcome saleOutcome
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &outcome))
	// The free copy sold left the two of the trade behind
	require.Equal(t, uint(2), outcome.Stacks[0].Copies)
	require.Zero(t, outcome.Stacks[0].Owned)
}

func TestSellDuplicatesInPlace(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell14", "foo-bar-baz-qux", 0)
	// 3 common, 2 rare and 1 foil rare
	giveCards(t, "sell14", cardRosetta, 3)
	giveCards(t, "sell14", cardMcMurdo, 2)
	giveCardAt(t, "sell14", cardMcMurdo, "foil1", time.Now())
	cookie := login(t, router, "sell14", "foo-bar-baz-qux")

	// 2 commons and 1 rare are the duplicates
	rec := sellInPlace(t, router, cookie, "/shop/sell-duplicates", url.Values{})
	require.Equal(t, http.StatusOK, rec.Code)
	var outcome saleOutcome
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &outcome))
	require.Equal(t, "You sold 3 cards for 70 coins", stripIsolation(outcome.Message))
	require.Equal(t, uint(70), outcome.Coins)
	require.Equal(t, uint(70), shopBalance(t, "sell14"))

	// One copy of each stack is left, worth nothing more to sell
	require.Len(t, outcome.Stacks, 2)
	for _, stack := range outcome.Stacks {
		require.Equal(t, uint(1), stack.Copies)
		require.Equal(t, uint(1), stack.Owned)
		if stack.Card == cardMcMurdo {
			require.Empty(t, stack.Foil)
		}
	}
	require.Zero(t, outcome.Duplicates.Count)
	require.Equal(t, int64(1), ownedCardCount(t, "sell14", cardRosetta, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "sell14", cardMcMurdo, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "sell14", cardMcMurdo, "foil1"))

	// Nothing left to sell says so without a redirect either
	rec = sellInPlace(t, router, cookie, "/shop/sell-duplicates", url.Values{})
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var failure saleFailure
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &failure))
	require.Equal(t, "You have no duplicate cards to sell", stripIsolation(failure.Error))
	require.Equal(t, uint(70), shopBalance(t, "sell14"))
}
