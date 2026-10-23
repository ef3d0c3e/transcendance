package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// The `card` fields of a contract, one per copy taken
func contract(values ...string) url.Values {
	return url.Values{"card": values}
}

// `n` copies of the same card in a contract
func copies(cardID uint, foil string, n int) []string {
	id := strconv.FormatUint(uint64(cardID), 10)
	if foil != "" {
		id += ":" + foil
	}
	values := make([]string, n)
	for i := range values {
		values[i] = id
	}
	return values
}

func tradeup(t *testing.T, router *gin.Engine, cookie *http.Cookie, form url.Values) *httptest.ResponseRecorder {
	return postForm(t, router, cookie, "/tradeup", form)
}

// Number of copies of one card owned by a user
func ownedCopies(t *testing.T, username string, cardID uint) int {
	n := 0
	for _, owned := range ownedCards(t, username) {
		if owned.CardID == cardID {
			n++
		}
	}
	return n
}

func TestTradeupConsumesCardsAndAwardsOneTierUp(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup1", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup1", cardPythagorean, 10)
	cookie := login(t, router, "tradeup1", "foo-bar-baz-qux")

	rec := tradeup(t, router, cookie, contract(copies(cardPythagorean, "", 10)...))

	// McMurdo Station is the only rare in the dev collection
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=tradeup&result=1003", rec.Header().Get("Location"))
	require.Zero(t, ownedCopies(t, "tradeup1", cardPythagorean))
	require.Equal(t, 1, ownedCopies(t, "tradeup1", cardMcMurdo))

	// The page names what came out of it, in a popup the client opens
	body := getInventory(t, router, cookie, "/inventory?tab=tradeup&result=1003")
	require.Contains(t, body, `id="tradeup-result"`)
	require.Contains(t, body, "You obtained")
	require.Contains(t, body, `<h2 class="card-title">McMurdo Station</h2>`)
}

func TestTradeupAwardsOneTierUpFromRareCards(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup2", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup2", cardMcMurdo, 10)
	cookie := login(t, router, "tradeup2", "foo-bar-baz-qux")

	rec := tradeup(t, router, cookie, contract(copies(cardMcMurdo, "", 10)...))

	// Ford Model T is the only legendary in the dev collection
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=tradeup&result=1004", rec.Header().Get("Location"))
	require.Zero(t, ownedCopies(t, "tradeup2", cardMcMurdo))
	require.Equal(t, 1, ownedCopies(t, "tradeup2", cardFordT))
}

func TestTradeupMixesCollectionsOfOneRarity(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup3", "foo-bar-baz-qux", 0)
	// Three distinct commons, all in the dev collection
	giveCards(t, "tradeup3", cardPythagorean, 5)
	giveCards(t, "tradeup3", cardFlyingScotsman, 3)
	giveCards(t, "tradeup3", cardTungsten, 2)
	cookie := login(t, router, "tradeup3", "foo-bar-baz-qux")

	form := contract(append(append(
		copies(cardPythagorean, "", 5),
		copies(cardFlyingScotsman, "", 3)...),
		copies(cardTungsten, "", 2)...)...)
	rec := tradeup(t, router, cookie, form)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=tradeup&result=1003", rec.Header().Get("Location"))
	// The 10 cards are gone, only the reward is left
	require.Equal(t, 1, len(ownedCards(t, "tradeup3")))
	require.Equal(t, 1, ownedCopies(t, "tradeup3", cardMcMurdo))
}

func TestTradeupRequiresExactlyTenCards(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup4", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup4", cardPythagorean, 30)
	cookie := login(t, router, "tradeup4", "foo-bar-baz-qux")

	for _, n := range []int{0, 1, 9, 11} {
		rec := tradeup(t, router, cookie, contract(copies(cardPythagorean, "", n)...))
		require.Equal(t, http.StatusSeeOther, rec.Code)
		require.Equal(t, "/inventory?tab=tradeup&error=count", rec.Header().Get("Location"))
	}
	// Nothing was spent
	require.Equal(t, 30, ownedCopies(t, "tradeup4", cardPythagorean))

	body := getInventory(t, router, cookie, "/inventory?tab=tradeup&error=count")
	require.Contains(t, body, "A tradeup needs exactly 10 cards")
}

func TestTradeupRejectsMixedRarities(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup5", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup5", cardPythagorean, 5)
	giveCards(t, "tradeup5", cardMcMurdo, 5)
	cookie := login(t, router, "tradeup5", "foo-bar-baz-qux")

	form := contract(append(
		copies(cardPythagorean, "", 5),
		copies(cardMcMurdo, "", 5)...)...)
	rec := tradeup(t, router, cookie, form)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=tradeup&error=rarity", rec.Header().Get("Location"))
	require.Equal(t, 5, ownedCopies(t, "tradeup5", cardPythagorean))
	require.Equal(t, 5, ownedCopies(t, "tradeup5", cardMcMurdo))
}

func TestTradeupRejectsHighestRarity(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup6", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup6", cardFordT, 10)
	cookie := login(t, router, "tradeup6", "foo-bar-baz-qux")

	rec := tradeup(t, router, cookie, contract(copies(cardFordT, "", 10)...))

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=tradeup&error=tier", rec.Header().Get("Location"))
	require.Equal(t, 10, ownedCopies(t, "tradeup6", cardFordT))
}

func TestTradeupRejectsCardsNotOwned(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup7", "foo-bar-baz-qux", 0)
	// One copy short of a contract
	giveCards(t, "tradeup7", cardPythagorean, 9)
	cookie := login(t, router, "tradeup7", "foo-bar-baz-qux")

	rec := tradeup(t, router, cookie, contract(copies(cardPythagorean, "", 10)...))

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=tradeup&error=missing", rec.Header().Get("Location"))
	require.Equal(t, 9, ownedCopies(t, "tradeup7", cardPythagorean))
}

func TestTradeupRejectsUnknownCards(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup8", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup8", cardPythagorean, 10)
	cookie := login(t, router, "tradeup8", "foo-bar-baz-qux")

	values := copies(cardPythagorean, "", 9)
	values = append(values, "9999")
	rec := tradeup(t, router, cookie, contract(values...))

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=tradeup&error=missing", rec.Header().Get("Location"))
	require.Equal(t, 10, ownedCopies(t, "tradeup8", cardPythagorean))
}

func TestTradeupConsumesOnlyTheSelectedFoil(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup9", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup9", cardPythagorean, 10)
	for range 3 {
		giveCardAt(t, "tradeup9", cardPythagorean, "foil1", time.Now())
	}
	cookie := login(t, router, "tradeup9", "foo-bar-baz-qux")

	// The contract asks for the plain copies, the foils stay behind
	rec := tradeup(t, router, cookie, contract(copies(cardPythagorean, "", 10)...))

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=tradeup&result=1003", rec.Header().Get("Location"))

	foils, plain := 0, 0
	for _, owned := range ownedCards(t, "tradeup9") {
		if owned.CardID != cardPythagorean {
			continue
		}
		if owned.Foil == "foil1" {
			foils++
		} else {
			plain++
		}
	}
	require.Equal(t, 3, foils)
	require.Zero(t, plain)
	// The three foils and the reward
	require.Equal(t, 4, len(ownedCards(t, "tradeup9")))
	require.Equal(t, 1, ownedCopies(t, "tradeup9", cardMcMurdo))
}

func TestTradeupAnonymousIsSentToLogin(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := tradeup(t, router, nil, contract(copies(cardPythagorean, "", 10)...))

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestTradeupTabListsOnlyCardsWithATierAbove(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup10", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup10", cardPythagorean, 2)
	giveCards(t, "tradeup10", cardFordT, 1)
	cookie := login(t, router, "tradeup10", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?tab=tradeup")

	// Ford Model T is legendary, there is no tier above it
	require.Contains(t, body, `<h2 class="card-title">Pythagorean tiling</h2>`)
	require.Zero(t, strings.Count(body, `<h2 class="card-title">Ford Model T</h2>`))
	require.Contains(t, body, `id="tradeup-picker"`)
	require.Contains(t, body, `data-tradeup-count="2"`)
	require.Contains(t, body, `data-tradeup-rarity="common"`)
	require.Contains(t, body, `action="/tradeup" method="post"`)
	require.Contains(t, body, `data-cost="10"`)
	require.Contains(t, body, "Trade 10 cards of the same rarity for one card of the next rarity")
	// The contract sits in a sidebar beside the cards
	require.Contains(t, body, `class="tradeup-layout"`)
}

func TestTradeupTabIsNotOfferedOnOtherInventories(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup11", "foo-bar-baz-qux", 0)
	createAccount(t, router, "tradeup12", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup11", cardPythagorean, 2)
	cookie := login(t, router, "tradeup12", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?u=tradeup11&tab=tradeup")

	require.NotContains(t, body, "inventory-tab-tradeup")
	require.NotContains(t, body, `id="tradeup-picker"`)
	// Falls back to the cards tab
	require.Contains(t, body, `<h2 class="card-title">Pythagorean tiling</h2>`)
}

func TestTradeupTabOffersTheContractEvenWhenEmpty(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup13", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "tradeup13", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?tab=tradeup")

	require.Contains(t, body, "No cards here can be traded up")
	require.NotContains(t, body, `id="tradeup-picker"`)
}

func TestTradeupTabCollectionFilter(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup14", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup14", cardPythagorean, 2)
	cookie := login(t, router, "tradeup14", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?tab=tradeup")

	// All collections by default, then the one the grid holds
	require.Contains(t, body, `id="tradeup-collection-filter"`)
	require.Contains(t, body, "All collections")
	require.Contains(t, body, `value="1">`)
	// Each tile names its collection, that's what the filter reads
	require.Contains(t, body, `data-tradeup-collection-id="1"`)
}

func TestTradeupTabOrdersCardsByCollectionRarityAndCount(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "tradeup15", "foo-bar-baz-qux", 0)
	giveCards(t, "tradeup15", cardPythagorean, 30)
	giveCards(t, "tradeup15", cardMcMurdo, 5)
	giveCards(t, "tradeup15", cardTungsten, 2)
	cookie := login(t, router, "tradeup15", "foo-bar-baz-qux")

	// One collection, so rarity decides first, then how many copies to spend
	body := getInventory(t, router, cookie, "/inventory?tab=tradeup")
	require.Equal(t, []string{"McMurdo Station", "Pythagorean tiling", "Tungsten"}, cardTitles(body))
}
