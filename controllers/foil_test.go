package controllers_test

import (
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

func foilRoll(t *testing.T, router *gin.Engine, cookie *http.Cookie, target string, sacrifices ...string) *httptest.ResponseRecorder {
	return postForm(t, router, cookie, "/foil", url.Values{"target": {target}, "card": sacrifices})
}

// Copies of one card owned by a user, by foil ("" for plain copies)
func ownedFoils(t *testing.T, username string, cardID uint) map[string]int {
	foils := make(map[string]int)
	for _, owned := range ownedCards(t, username) {
		if owned.CardID == cardID {
			foils[owned.Foil]++
		}
	}
	return foils
}

// A roll is won or lost; the sacrifices go either way, the rolled card turns
// foil or is destroyed. Rolled at the 50% cap until both outcomes were seen.
func TestFoilRouletteWinsOrDestroys(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foil1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "foil1", "foo-bar-baz-qux")
	target := strconv.Itoa(cardPythagorean)
	// 1% + 23 * 2% is past the cap
	sacrifices := copies(cardFordT, "", 23)

	won, lost := false, false
	for i := 0; i < 30 && (!won || !lost); i++ {
		require.NoError(t, tests.DB.Where("owner_id = ?", userID(t, "foil1")).Delete(&models.Card{}).Error)
		giveCards(t, "foil1", cardPythagorean, 2)
		giveCards(t, "foil1", cardFordT, 23)

		rec := foilRoll(t, router, cookie, target, sacrifices...)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		location, err := url.Parse(rec.Header().Get("Location"))
		require.NoError(t, err)
		require.Equal(t, "/inventory", location.Path)
		query := location.Query()
		require.Equal(t, "foil", query.Get("tab"))
		require.Equal(t, target, query.Get("result"))

		// Every sacrifice is spent, the other plain copy stays
		require.Empty(t, ownedFoils(t, "foil1", cardFordT))
		foils := ownedFoils(t, "foil1", cardPythagorean)
		require.Equal(t, 1, foils[""])

		if pattern := query.Get("foil"); pattern != "" {
			won = true
			require.Empty(t, query.Get("lost"))
			require.Equal(t, map[string]int{"": 1, pattern: 1}, foils)

			body := getInventory(t, router, cookie, location.String())
			require.Contains(t, body, `id="tradeup-result"`)
			require.Contains(t, body, "It&#39;s a foil!")
			require.Contains(t, body, "/cards/assets/"+pattern+".png")
		} else {
			lost = true
			require.Equal(t, "1", query.Get("lost"))
			require.Equal(t, map[string]int{"": 1}, foils)

			body := getInventory(t, router, cookie, location.String())
			require.Contains(t, body, `data-outcome="lost"`)
			require.Contains(t, body, "Card destroyed")
		}
	}
	require.True(t, won, "no roll was won")
	require.True(t, lost, "no roll was lost")
}

func TestFoilRouletteWithoutSacrifices(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foil2", "foo-bar-baz-qux", 0)
	giveCards(t, "foil2", cardMcMurdo, 1)
	cookie := login(t, router, "foil2", "foo-bar-baz-qux")

	rec := foilRoll(t, router, cookie, strconv.Itoa(cardMcMurdo))

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "result=1003")
	// Kept as a foil, or gone
	foils := ownedFoils(t, "foil2", cardMcMurdo)
	require.LessOrEqual(t, len(foils), 1)
	require.Zero(t, foils[""])
}

func TestFoilRouletteRejectsFoilTargets(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foil3", "foo-bar-baz-qux", 0)
	giveCardAt(t, "foil3", cardPythagorean, "foil1", time.Now())
	cookie := login(t, router, "foil3", "foo-bar-baz-qux")

	rec := foilRoll(t, router, cookie, strconv.Itoa(cardPythagorean)+":foil1")

	require.Equal(t, "/inventory?tab=foil&error=foil", rec.Header().Get("Location"))
	require.Equal(t, map[string]int{"foil1": 1}, ownedFoils(t, "foil3", cardPythagorean))
	require.Contains(t, getInventory(t, router, cookie, "/inventory?tab=foil&error=foil"), "This card is already foil")
}

func TestFoilRouletteRejectsMissingCards(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foil4", "foo-bar-baz-qux", 0)
	giveCards(t, "foil4", cardPythagorean, 2)
	cookie := login(t, router, "foil4", "foo-bar-baz-qux")
	target := strconv.Itoa(cardPythagorean)

	for _, sacrifices := range [][]string{
		// The rolled copy can't be sacrificed too
		copies(cardPythagorean, "", 2),
		copies(cardMcMurdo, "", 1),
		{"9999"},
		{"nope"},
	} {
		rec := foilRoll(t, router, cookie, target, sacrifices...)
		require.Equal(t, "/inventory?tab=foil&error=missing", rec.Header().Get("Location"))
	}
	for _, target := range []string{"", "0", "9999", strconv.Itoa(cardMcMurdo)} {
		rec := foilRoll(t, router, cookie, target)
		require.Equal(t, "/inventory?tab=foil&error=missing", rec.Header().Get("Location"))
	}
	// Nothing was spent
	require.Equal(t, map[string]int{"": 2}, ownedFoils(t, "foil4", cardPythagorean))
}

func TestFoilRouletteLimitsSacrifices(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foil5", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "foil5", "foo-bar-baz-qux")

	rec := foilRoll(t, router, cookie, strconv.Itoa(cardPythagorean), copies(cardTungsten, "", 101)...)

	require.Equal(t, "/inventory?tab=foil&error=count", rec.Header().Get("Location"))
	require.Contains(t, getInventory(t, router, cookie, "/inventory?tab=foil&error=count"), "A roll can sacrifice at most 100 cards")
}

// Copies up in pending trades leave with them, a roll cannot spend them
func TestFoilRouletteKeepsCopiesUpInTrades(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foil6", "foo-bar-baz-qux", 0)
	createAccount(t, router, "foil7", "foo-bar-baz-qux", 0)
	giveCards(t, "foil6", cardPythagorean, 1)
	giveCards(t, "foil6", cardMcMurdo, 2)
	giveCards(t, "foil7", cardTungsten, 1)
	cookie := login(t, router, "foil6", "foo-bar-baz-qux")

	rec := postJSON(t, router, cookie, "/trades", tradeBody(t, "foil7",
		[]models.TradeItem{item(cardMcMurdo, "", 1)}, 0,
		[]models.TradeItem{item(cardTungsten, "", 1)}, 0))
	require.Equal(t, http.StatusCreated, rec.Code)

	// The tab only offers the free copy
	body := getInventory(t, router, cookie, "/inventory?tab=foil")
	require.Contains(t, body, `data-foil-id="1003" data-foil-foil=""`)
	require.Contains(t, body, `data-foil-count="1" data-foil-rarity="rare"`)

	rec = foilRoll(t, router, cookie, strconv.Itoa(cardPythagorean), copies(cardMcMurdo, "", 2)...)
	require.Equal(t, "/inventory?tab=foil&error=locked", rec.Header().Get("Location"))
	require.Equal(t, map[string]int{"": 2}, ownedFoils(t, "foil6", cardMcMurdo))
	require.Equal(t, map[string]int{"": 1}, ownedFoils(t, "foil6", cardPythagorean))
}

func TestFoilRouletteAnonymousIsSentToLogin(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := foilRoll(t, router, nil, strconv.Itoa(cardPythagorean))

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestFoilRouletteTab(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foil8", "foo-bar-baz-qux", 0)
	giveCards(t, "foil8", cardTungsten, 1)
	giveCards(t, "foil8", cardFordT, 2)
	giveCardAt(t, "foil8", cardPythagorean, "foil2", time.Now())
	cookie := login(t, router, "foil8", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?tab=foil")

	// Next to the tradeup tab
	require.Contains(t, body, `<a class="inventory-tab-tradeup" href="/inventory?tab=tradeup" >`)
	require.Contains(t, body, `<a class="inventory-tab-foil" href="/inventory?tab=foil" aria-current="page">`)
	require.Less(t, strings.Index(body, "inventory-tab-tradeup"), strings.Index(body, "inventory-tab-foil"))

	// Every rarity can be rolled or sacrificed, foils included, in collection order
	require.Equal(t, []string{"Pythagorean tiling", "Ford Model T", "Tungsten"}, cardTitles(body))
	require.Contains(t, body, `data-foil-id="1001" data-foil-foil="foil2"`)
	require.Contains(t, body, `data-foil-count="1" data-foil-rarity="common"`)
	require.Contains(t, body, `data-foil-id="1004" data-foil-foil=""`)
	require.Contains(t, body, `data-foil-count="2" data-foil-rarity="legendary"`)
	require.Contains(t, body, `id="foil-roulette"`)
	require.Contains(t, body, `action="/foil" method="post"`)
	require.Contains(t, body, `data-cap="5000"`)
	require.Contains(t, body, `data-scale="10000"`)
	require.Contains(t, body, `&#34;legendary&#34;:{&#34;base&#34;:500,&#34;bonus&#34;:200}`)
	require.Contains(t, stripIsolation(body), "up to 50%")

	// Search, collection, rarity and sort, applied in the page
	require.Contains(t, body, `id="foil-search"`)
	require.Contains(t, body, `id="foil-collection-filter"`)
	require.Contains(t, body, `id="foil-rarity-filter"`)
	require.Contains(t, body, `<option value="legendary">Legendary</option>`)
	require.Contains(t, body, `id="foil-sort"`)
	require.Contains(t, body, `<option value="number" selected>Collection order</option>`)
	require.Contains(t, body, `id="foil-no-match" hidden>`)
	// Paged in the page, above and below the grid, so the picks survive
	require.Equal(t, 2, strings.Count(body, `class="foil-pager card-pager" data-page-size="20"`))
}

func TestFoilRouletteTabWhenEmpty(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foil9", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "foil9", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?tab=foil")

	require.Contains(t, body, "No cards here can be rolled for a foil")
	require.NotContains(t, body, `id="foil-roulette"`)
}

func TestFoilRouletteTabIsNotOfferedOnOtherInventories(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foil10", "foo-bar-baz-qux", 0)
	createAccount(t, router, "foil11", "foo-bar-baz-qux", 0)
	giveCards(t, "foil10", cardPythagorean, 2)
	cookie := login(t, router, "foil11", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?u=foil10&tab=foil")

	require.NotContains(t, body, "inventory-tab-foil")
	require.NotContains(t, body, `id="foil-roulette"`)
	require.Contains(t, body, `class="inventory-tab-boosters" href="/inventory?u=foil10" aria-current="page"`)
}
