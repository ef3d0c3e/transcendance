package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"transcendance/controllers"
	"transcendance/models"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Concurrent requests of the same user, as from several tabs, or a script
const concurrentRequests = 8

func raceForm(router *gin.Engine, cookie *http.Cookie, target string, form url.Values, n int) []*httptest.ResponseRecorder {
	recs := make([]*httptest.ResponseRecorder, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(cookie)
			recs[i] = httptest.NewRecorder()
			<-start
			router.ServeHTTP(recs[i], req)
		}()
	}
	close(start)
	wg.Wait()
	return recs
}

// A user with exactly the price of one booster gets one, however many buys run at once
func TestConcurrentBuysSpendOnce(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "racebuy1", "foo-bar-baz-qux", 0)
	setCoins(t, "racebuy1", controllers.BoosterPrice(collectionDev))
	cookie := login(t, router, "racebuy1", "foo-bar-baz-qux")

	raceForm(router, cookie, "/shop/buy", url.Values{"collection": {strconv.Itoa(collectionDev)}}, concurrentRequests)

	require.Len(t, shopBoosters(t, "racebuy1"), 1)
	require.Zero(t, shopBalance(t, "racebuy1"))
}

func TestConcurrentOpeningsOpenOnce(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "raceopen1", "foo-bar-baz-qux", 0)
	giveBoosters(t, "raceopen1", collectionDev, 5, 1)
	cookie := login(t, router, "raceopen1", "foo-bar-baz-qux")
	booster := ownedBoosters(t, "raceopen1")[0]

	recs := raceForm(router, cookie, "/opening", url.Values{"booster": {strconv.FormatUint(uint64(booster.ID), 10)}}, concurrentRequests)

	opened := 0
	for _, rec := range recs {
		if rec.Code == http.StatusOK {
			opened++
		}
	}
	require.Equal(t, 1, opened)
	require.Empty(t, ownedBoosters(t, "raceopen1"))
	require.Len(t, ownedCards(t, "raceopen1"), 5)
}

func TestConcurrentDailyClaimsRewardOnce(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "raceclaim1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "raceclaim2", "foo-bar-baz-qux", 0)

	// One claim alone, for reference
	claimDaily(t, router, login(t, router, "raceclaim2", "foo-bar-baz-qux"), "cards")

	cookie := login(t, router, "raceclaim1", "foo-bar-baz-qux")
	raceForm(router, cookie, "/inventory/claim", url.Values{"tab": {"cards"}}, concurrentRequests)

	require.Equal(t, uint(1), dailyStreak(t, "raceclaim1").Day)
	require.Equal(t, shopBalance(t, "raceclaim2"), shopBalance(t, "raceclaim1"))
	require.Equal(t, len(ownedBoosters(t, "raceclaim2")), len(ownedBoosters(t, "raceclaim1")))
}

func TestConcurrentAcceptsSettleOnce(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "raceacc1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "raceacc2", "foo-bar-baz-qux", 0)
	giveCards(t, "raceacc1", cardFordT, 1)
	giveCards(t, "raceacc2", cardRosetta, 1)
	setCoins(t, "raceacc1", 100)
	cookie1 := login(t, router, "raceacc1", "foo-bar-baz-qux")
	cookie2 := login(t, router, "raceacc2", "foo-bar-baz-qux")

	rec := postJSON(t, router, cookie1, "/trades", tradeBody(t, "raceacc2",
		[]models.TradeItem{item(cardFordT, "", 1)}, 100,
		[]models.TradeItem{item(cardRosetta, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	trade := tradesOf(t, "raceacc1")[0]

	raceForm(router, cookie2, fmt.Sprintf("/trades/%d/accept", trade.ID), url.Values{}, concurrentRequests)

	require.Equal(t, models.TradeAccepted, tradesOf(t, "raceacc1")[0].State)
	require.Equal(t, int64(1), ownedCardCount(t, "raceacc2", cardFordT, ""))
	require.Equal(t, int64(1), ownedCardCount(t, "raceacc1", cardRosetta, ""))
	require.Zero(t, ownedCardCount(t, "raceacc1", cardFordT, ""))
	require.Equal(t, uint(100), shopBalance(t, "raceacc2"))
	require.Zero(t, shopBalance(t, "raceacc1"))
}

// Selling a card while its offer is accepted: one of the two wins, the card is never sold and traded both
func TestConcurrentSellAndAccept(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "racesell1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "racesell2", "foo-bar-baz-qux", 0)
	giveCards(t, "racesell1", cardFordT, 1)
	giveCards(t, "racesell2", cardRosetta, 1)
	cookie1 := login(t, router, "racesell1", "foo-bar-baz-qux")
	cookie2 := login(t, router, "racesell2", "foo-bar-baz-qux")

	rec := postJSON(t, router, cookie1, "/trades", tradeBody(t, "racesell2",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		[]models.TradeItem{item(cardRosetta, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)
	trade := tradesOf(t, "racesell1")[0]

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		raceForm(router, cookie1, "/shop/sell", sellForm(cardFordT, "", 1), 1)
	}()
	go func() {
		defer wg.Done()
		raceForm(router, cookie2, fmt.Sprintf("/trades/%d/accept", trade.ID), url.Values{}, 1)
	}()
	wg.Wait()

	// The card is up in the offer, the sale is refused and the offer settles
	require.Equal(t, models.TradeAccepted, tradesOf(t, "racesell1")[0].State)
	require.Equal(t, int64(1), ownedCardCount(t, "racesell2", cardFordT, ""))
	require.Zero(t, ownedCardCount(t, "racesell1", cardFordT, ""))
	require.Zero(t, shopBalance(t, "racesell1"))
}

// Foils are exactly a known pattern: " " or "FOIL1" would match other cards in the database
func TestSellRejectsFoilVariants(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sellfoil1", "foo-bar-baz-qux", 0)
	giveCards(t, "sellfoil1", cardFordT, 2)
	cookie := login(t, router, "sellfoil1", "foo-bar-baz-qux")

	for _, foil := range []string{" ", "FOIL1", "foil1 ", "foil9"} {
		sell(t, router, cookie, sellForm(cardFordT, foil, 1))
		require.Equal(t, int64(2), ownedCardCount(t, "sellfoil1", cardFordT, ""), foil)
		require.Zero(t, shopBalance(t, "sellfoil1"), foil)
	}

	// A plain legendary sells at the plain price
	sell(t, router, cookie, sellForm(cardFordT, "", 1))
	require.Equal(t, int64(1), ownedCardCount(t, "sellfoil1", cardFordT, ""))
	require.Equal(t, controllers.SellPrice("legendary", ""), shopBalance(t, "sellfoil1"))
}

// Cards up in a pending offer stay there, whatever the spelling of their foil
func TestReservedCardsKeepTheirOffer(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "reserve1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "reserve2", "foo-bar-baz-qux", 0)
	giveCards(t, "reserve1", cardFordT, 1)
	giveCards(t, "reserve2", cardRosetta, 1)
	cookie1 := login(t, router, "reserve1", "foo-bar-baz-qux")

	rec := postJSON(t, router, cookie1, "/trades", tradeBody(t, "reserve2",
		[]models.TradeItem{item(cardFordT, "", 1)}, 0,
		[]models.TradeItem{item(cardRosetta, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	sell(t, router, cookie1, sellForm(cardFordT, " ", 1))
	require.Equal(t, int64(1), ownedCardCount(t, "reserve1", cardFordT, ""))

	rec = postForm(t, router, cookie1, "/tradeup", url.Values{"card": {"1004: ", "1004: "}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, int64(1), ownedCardCount(t, "reserve1", cardFordT, ""))

	// Unknown foils cannot be offered either
	rec = postJSON(t, router, cookie1, "/trades", tradeBody(t, "reserve2",
		[]models.TradeItem{item(cardFordT, "FOIL1", 1)}, 0,
		[]models.TradeItem{item(cardRosetta, "", 1)}, 0))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// Usernames match ignoring case and trailing spaces in the database, users are compared by ID
func TestTradeWithYourselfPadded(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "selftrade1", "foo-bar-baz-qux", 0)
	giveCards(t, "selftrade1", cardFordT, 1)
	cookie := login(t, router, "selftrade1", "foo-bar-baz-qux")

	for _, target := range []string{"selftrade1 ", "SELFTRADE1"} {
		rec := postJSON(t, router, cookie, "/trades", tradeBody(t, target,
			[]models.TradeItem{item(cardFordT, "", 1)}, 0, nil, 0))
		require.Equal(t, http.StatusBadRequest, rec.Code, target)
	}
	require.Empty(t, tradesOf(t, "selftrade1"))
}

// Every offer notifies its receiver, a user cannot flood another with them
func TestTradePendingOffersLimit(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "offerlim1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "offerlim2", "foo-bar-baz-qux", 0)
	giveCards(t, "offerlim2", cardRosetta, 1)
	setCoins(t, "offerlim1", 100)
	cookie := login(t, router, "offerlim1", "foo-bar-baz-qux")

	offer := tradeBody(t, "offerlim2", nil, 1, []models.TradeItem{item(cardRosetta, "", 1)}, 0)
	for range 5 {
		require.Equal(t, http.StatusCreated, postJSON(t, router, cookie, "/trades", offer).Code)
	}
	rec := postJSON(t, router, cookie, "/trades", offer)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), "too many offers")
}

// Huge page numbers would overflow the offsets
func TestHugePageNumbers(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "hugepage1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "hugepage2", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "hugepage1", "foo-bar-baz-qux")

	huge := "461168601842738792"
	for _, target := range []string{"/inventory?tab=cards&p=" + huge, "/inventory?tab=boosters&p=" + huge,
		"/cards/search/?p=" + huge, "/search?p=" + huge, "/notifications?p=" + huge,
		"/api/trade/stacks?u=hugepage2&page=" + huge} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Less(t, rec.Code, http.StatusInternalServerError, target)
	}
}
