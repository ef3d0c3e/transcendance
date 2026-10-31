package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"transcendance/models"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Global IDs of cards in the dev collection
const (
	cardPythagorean    = 1001
	cardFlyingScotsman = 1002
	cardMcMurdo        = 1003
	cardFordT          = 1004
	cardTungsten       = 1005
)

func giveCards(t *testing.T, username string, cardID uint, count int) {
	var user models.User
	require.NoError(t, tests.DB.Where("username = ?", username).First(&user).Error)

	for range count {
		require.NoError(t, tests.DB.Create(&models.Card{
			CardID:   cardID,
			Acquired: time.Now(),
			OwnerID:  user.ID,
		}).Error)
	}
}

// Give one copy of a card with a foil and acquisition date
func giveCardAt(t *testing.T, username string, cardID uint, foil string, acquired time.Time) {
	require.NoError(t, tests.DB.Create(&models.Card{
		CardID:   cardID,
		Acquired: acquired,
		Foil:     foil,
		OwnerID:  userID(t, username),
	}).Error)
}

// Titles of the displayed cards, in order
func cardTitles(body string) []string {
	var titles []string
	for _, match := range cardTitleRe.FindAllStringSubmatch(body, -1) {
		titles = append(titles, match[1])
	}
	return titles
}

func getInventory(t *testing.T, router *gin.Engine, cookie *http.Cookie, target string) string {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.AddCookie(cookie)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	// Fluent wraps placeables in unicode bidi isolation marks
	return strings.NewReplacer("\u2068", "", "\u2069", "").Replace(rec.Body.String())
}

func countCard(body string, title string) int {
	return strings.Count(body, `<h2 class="card-title">`+title+`</h2>`)
}

func TestInventorySearchOwnedOnly(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invsearch1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "invsearch2", "foo-bar-baz-qux", 0)
	giveCards(t, "invsearch1", cardFordT, 2)
	giveCards(t, "invsearch1", cardTungsten, 1)
	giveCards(t, "invsearch2", cardFlyingScotsman, 1)
	cookie := login(t, router, "invsearch1", "foo-bar-baz-qux")

	// Both the Ford Model T and the Flying Scotsman are vehicles, only the owned one is returned
	body := getInventory(t, router, cookie, "/inventory?q=vehicle")
	// Both copies are folded together
	require.Equal(t, 1, countCard(body, "Ford Model T"))
	require.Contains(t, body, `<span class="card-stack">×2</span>`)
	require.Zero(t, countCard(body, "Flying Scotsman"))
	require.Zero(t, countCard(body, "Tungsten"))
	require.Contains(t, body, `<p class="inventory-count">2 cards found</p>`)
}

func TestInventorySearchNoMatch(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invsearch3", "foo-bar-baz-qux", 0)
	giveCards(t, "invsearch3", cardTungsten, 1)
	cookie := login(t, router, "invsearch3", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?q=antarctica")
	require.Zero(t, countCard(body, "Tungsten"))
	require.Contains(t, body, "No cards in this inventory match your search")
	require.NotContains(t, body, `class="inventory-count"`)
}

func TestInventorySearchEmptyQuery(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invsearch4", "foo-bar-baz-qux", 0)
	giveCards(t, "invsearch4", cardFordT, 1)
	giveCards(t, "invsearch4", cardTungsten, 1)
	cookie := login(t, router, "invsearch4", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?q=++")
	require.Equal(t, 1, countCard(body, "Ford Model T"))
	require.Equal(t, 1, countCard(body, "Tungsten"))
	require.Contains(t, body, `<p class="inventory-count">2 cards</p>`)
	require.Equal(t, 1, strings.Count(body, `<nav class="paginator"`))
}

func TestInventorySearchPagination(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invsearch5", "foo-bar-baz-qux", 0)
	// Every foil pattern folds separately: 5 cards × 7 looks = 35 stacks
	for _, card := range []uint{cardPythagorean, cardFlyingScotsman, cardMcMurdo, cardFordT, cardTungsten} {
		for _, foil := range []string{"", "foil1", "foil2", "foil3", "foil4", "foil5", "foil6"} {
			giveCardAt(t, "invsearch5", card, foil, time.Now())
		}
	}
	giveCards(t, "invsearch5", cardTungsten, 2)
	cookie := login(t, router, "invsearch5", "foo-bar-baz-qux")

	all := getInventory(t, router, cookie, "/inventory?tab=cards")
	require.Len(t, cardTitles(all), 20)
	require.Contains(t, all, `href="?tab=cards&amp;p=2"`)
	require.Contains(t, all, `<p class="inventory-count">37 cards</p>`)
	// Multiple pages get a second paginator below the cards
	require.Equal(t, 2, strings.Count(all, `<nav class="paginator"`))

	second := getInventory(t, router, cookie, "/inventory?tab=cards&p=2")
	require.Len(t, cardTitles(second), 15)

	// Searches count matching copies
	search := getInventory(t, router, cookie, "/inventory?q=ford")
	require.Equal(t, 7, countCard(search, "Ford Model T"))
	require.Zero(t, countCard(search, "Tungsten"))
	require.Contains(t, search, `<p class="inventory-count">7 cards found</p>`)
	require.Equal(t, 1, strings.Count(search, `<nav class="paginator"`))

	// Filters and sorts are kept in the paginator: 3 common cards × 7 looks
	filtered := getInventory(t, router, cookie, "/inventory?rarity=common&sort=name")
	require.Len(t, cardTitles(filtered), 20)
	require.Contains(t, filtered, `href="?rarity=common&amp;sort=name&amp;tab=cards&amp;p=2"`)
	require.Contains(t, filtered, `<p class="inventory-count">23 cards found</p>`)
}

func TestInventorySearchOtherUser(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invsearch6", "foo-bar-baz-qux", 0)
	createAccount(t, router, "invsearch7", "foo-bar-baz-qux", 0)
	giveCards(t, "invsearch6", cardFordT, 1)
	giveCards(t, "invsearch7", cardFlyingScotsman, 1)
	giveCards(t, "invsearch7", cardTungsten, 1)
	cookie := login(t, router, "invsearch6", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?u=invsearch7&q=train")
	require.Equal(t, 1, countCard(body, "Flying Scotsman"))
	require.Zero(t, countCard(body, "Ford Model T"))
	require.Zero(t, countCard(body, "Tungsten"))
	require.Contains(t, body, `<p class="inventory-count">1 card found</p>`)
	// The search form keeps the viewed user
	require.Contains(t, body, `<input type="hidden" name="u" value="invsearch7">`)
}

func TestInventoryCardAcquiredDate(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invpopup1", "foo-bar-baz-qux", 0)

	var user models.User
	require.NoError(t, tests.DB.Where("username = ?", "invpopup1").First(&user).Error)
	require.NoError(t, tests.DB.Create(&models.Card{
		CardID:   cardTungsten,
		Acquired: time.Date(2025, time.March, 14, 9, 26, 53, 0, time.Local),
		OwnerID:  user.ID,
	}).Error)
	cookie := login(t, router, "invpopup1", "foo-bar-baz-qux")

	// The card carries its acquisition date for the popup
	body := getInventory(t, router, cookie, "/inventory?tab=cards")
	require.Contains(t, body, `data-copies="1"`)
	require.Contains(t, body, `data-acquired-first="2025-03-14T09:26:53`)
	require.Contains(t, body, `data-acquired-last="2025-03-14T09:26:53`)
	require.Contains(t, body, `id="inventory-card-popup"`)
	require.Contains(t, body, `data-label-acquired="Acquired on"`)

	// Cards outside the inventory have no acquisition date
	search := getInventory(t, router, cookie, "/cards/search/?q=tungsten")
	require.Equal(t, 1, countCard(search, "Tungsten"))
	require.NotContains(t, search, "data-acquired")
	require.NotContains(t, search, "data-copies")
}

// Collection ID of the dev collection
const collectionDev = 1

func giveBoosters(t *testing.T, username string, collectionID uint, size uint, count int) {
	var user models.User
	require.NoError(t, tests.DB.Where("username = ?", username).First(&user).Error)

	for range count {
		require.NoError(t, tests.DB.Create(&models.Booster{
			CollectionID: collectionID,
			Size:         size,
			Acquired:     time.Now(),
			OwnerID:      user.ID,
		}).Error)
	}
}

func TestInventoryBoosters(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invbooster1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "invbooster2", "foo-bar-baz-qux", 0)
	giveBoosters(t, "invbooster1", collectionDev, 5, 3)
	// Acquired before the collection's booster size changed
	giveBoosters(t, "invbooster1", collectionDev, 3, 1)
	giveCards(t, "invbooster1", cardTungsten, 2)
	giveBoosters(t, "invbooster2", collectionDev, 5, 7)
	cookie := login(t, router, "invbooster1", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory")
	// One stack per (collection, size), and the other collections to buy
	require.Equal(t, len(tests.Data.Collections)+1, strings.Count(body, `<article class="collection"`))
	require.Contains(t, body, `<span class="collection-stack">×3</span>`)
	require.Contains(t, body, `<span class="collection-stack">×1</span>`)
	require.Contains(t, body, "5 cards booster")
	require.Contains(t, body, "3 cards booster")
	require.Contains(t, body, `<p class="inventory-count">4 boosters</p>`)
	// Boosters can't be searched, and tabs show both counts
	require.NotContains(t, body, `id="search-form"`)
	require.Contains(t, body, `<span class="inventory-tab-count">2</span>`)
	require.Contains(t, body, `<span class="inventory-tab-count">4</span>`)
	require.Zero(t, countCard(body, "Tungsten"))

	// The cards tab is unaffected
	cards := getInventory(t, router, cookie, "/inventory?tab=cards")
	require.Equal(t, 1, countCard(cards, "Tungsten"))
	require.NotContains(t, cards, `class="collection-stack"`)
}

func TestInventoryBoostersOtherUser(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invbooster3", "foo-bar-baz-qux", 0)
	createAccount(t, router, "invbooster4", "foo-bar-baz-qux", 0)
	giveBoosters(t, "invbooster4", collectionDev, 5, 2)
	cookie := login(t, router, "invbooster3", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?u=invbooster4&tab=boosters")
	require.Contains(t, body, `<span class="collection-stack">×2</span>`)
	// Only the boosters owned, none is sold here
	require.Equal(t, 1, strings.Count(body, `<article class="collection"`))
	require.NotContains(t, body, "shop-booster")
	require.NotContains(t, body, `id="shop-buy-popup"`)
	require.NotContains(t, body, "data-booster-open")
	// Tabs keep the viewed user
	require.Contains(t, body, `href="/inventory?tab=cards&amp;u=invbooster4"`)
	require.Contains(t, body, `href="/inventory?u=invbooster4"`)
}

func TestInventoryBoostersEmpty(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invbooster5", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "invbooster5", "foo-bar-baz-qux")

	// Every collection is listed grayed out, clicking one buys some
	body := getInventory(t, router, cookie, "/inventory")
	require.Equal(t, len(tests.Data.Collections), strings.Count(body, `<div class="inventory-booster is-empty shop-booster"`))
	require.Equal(t, len(tests.Data.Collections), strings.Count(body, `<span class="collection-stack collection-stack-empty">×0</span>`))
	require.Contains(t, body, `data-collection="1"`)
	require.Contains(t, body, `data-price="250"`)
	require.Contains(t, body, `id="shop-buy-popup" data-coins="0"`)
	// With the balance and the coin top-up
	require.Contains(t, body, "Balance: 0 coins")
	require.Contains(t, body, `id="shop-topup-popup"`)
	require.NotContains(t, body, "No boosters yet")

	// Other users with no boosters have nothing to show
	createAccount(t, router, "invbooster6", "foo-bar-baz-qux", 0)
	other := getInventory(t, router, cookie, "/inventory?u=invbooster6")
	require.Contains(t, other, "No boosters yet")
	require.NotContains(t, other, `<article class="collection"`)
}

func TestInventoryDefaultTab(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invtab1", "foo-bar-baz-qux", 0)
	giveCards(t, "invtab1", cardTungsten, 1)
	cookie := login(t, router, "invtab1", "foo-bar-baz-qux")

	// Boosters by default
	body := getInventory(t, router, cookie, "/inventory")
	require.Contains(t, body, `class="inventory-tab-boosters" href="/inventory" aria-current="page"`)
	require.Zero(t, countCard(body, "Tungsten"))

	// A search, filter or sort is about cards
	for _, query := range []string{"q=tungsten", "rarity=common", "effect=none", "sort=name", "tab=cards"} {
		body := getInventory(t, router, cookie, "/inventory?"+query)
		require.Equal(t, 1, countCard(body, "Tungsten"), query)
	}
	// The search form stays on the cards, even emptied
	require.Contains(t, getInventory(t, router, cookie, "/inventory?tab=cards"), `<input type="hidden" name="tab" value="cards">`)
}

func TestInventoryCardFoil(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invfoil1", "foo-bar-baz-qux", 0)

	var user models.User
	require.NoError(t, tests.DB.Where("username = ?", "invfoil1").First(&user).Error)
	require.NoError(t, tests.DB.Create(&[]models.Card{
		{CardID: cardTungsten, Acquired: time.Now(), Foil: "foil3", OwnerID: user.ID},
		{CardID: cardFordT, Acquired: time.Now(), OwnerID: user.ID},
	}).Error)
	cookie := login(t, router, "invfoil1", "foo-bar-baz-qux")

	// Only the foil card has the foil effect, with its own pattern
	body := getInventory(t, router, cookie, "/inventory?tab=cards")
	require.Equal(t, 1, strings.Count(body, "card-effect card-foil"))
	require.Contains(t, body, `data-mask="/cards/assets/foil3.png"`)
	// Glare is the base look of every card, the watermark is reserved
	require.Equal(t, 2, strings.Count(body, "card-effect card-glare"))
	require.NotContains(t, body, "card-watermark")
}

func TestInventoryFoldsCopies(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invfold1", "foo-bar-baz-qux", 0)
	first := time.Date(2025, time.January, 2, 10, 0, 0, 0, time.Local)
	last := time.Date(2025, time.June, 7, 18, 30, 0, 0, time.Local)
	giveCardAt(t, "invfold1", cardFordT, "", last)
	giveCardAt(t, "invfold1", cardFordT, "", first)
	giveCardAt(t, "invfold1", cardFordT, "", first.AddDate(0, 1, 0))
	giveCardAt(t, "invfold1", cardFordT, "foil3", first)
	giveCardAt(t, "invfold1", cardTungsten, "", first)
	cookie := login(t, router, "invfold1", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?tab=cards")
	// Copies fold by card and foil
	require.Equal(t, 2, countCard(body, "Ford Model T"))
	require.Equal(t, 1, countCard(body, "Tungsten"))
	require.Equal(t, 1, strings.Count(body, "card-effect card-foil"))
	require.Equal(t, 1, strings.Count(body, `class="card-stack"`))
	require.Contains(t, body, `<span class="card-stack">×3</span>`)
	require.Contains(t, body, `data-copies="3"`)
	require.Contains(t, body, `data-acquired-first="2025-01-02T10:00:00`)
	require.Contains(t, body, `data-acquired-last="2025-06-07T18:30:00`)
	// The count still covers every copy
	require.Contains(t, body, `<p class="inventory-count">5 cards</p>`)
	require.Contains(t, body, `<span class="inventory-tab-count">5</span>`)
}

func TestInventoryFilters(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invfilter1", "foo-bar-baz-qux", 0)
	giveCardAt(t, "invfilter1", cardFordT, "", time.Now())
	giveCardAt(t, "invfilter1", cardFordT, "foil1", time.Now())
	giveCardAt(t, "invfilter1", cardMcMurdo, "", time.Now())
	giveCardAt(t, "invfilter1", cardTungsten, "foil2", time.Now())
	giveCardAt(t, "invfilter1", cardTungsten, "foil2", time.Now())
	cookie := login(t, router, "invfilter1", "foo-bar-baz-qux")

	legendary := getInventory(t, router, cookie, "/inventory?rarity=legendary")
	require.Equal(t, 2, countCard(legendary, "Ford Model T"))
	require.Zero(t, countCard(legendary, "McMurdo Station"))
	require.Zero(t, countCard(legendary, "Tungsten"))
	require.Contains(t, legendary, `<p class="inventory-count">2 cards found</p>`)
	require.Contains(t, legendary, `<option value="legendary" selected>Legendary</option>`)

	foil := getInventory(t, router, cookie, "/inventory?effect=foil")
	require.Equal(t, []string{"Ford Model T", "Tungsten"}, sortedTitles(foil))
	require.Contains(t, foil, `<p class="inventory-count">3 cards found</p>`)

	none := getInventory(t, router, cookie, "/inventory?effect=none")
	require.Equal(t, []string{"Ford Model T", "McMurdo Station"}, sortedTitles(none))
	require.NotContains(t, none, "card-effect card-foil")

	// Filters combine with each other and the search
	combined := getInventory(t, router, cookie, "/inventory?q=ford&effect=foil&rarity=legendary")
	require.Equal(t, 1, countCard(combined, "Ford Model T"))
	require.Contains(t, combined, "card-effect card-foil")

	empty := getInventory(t, router, cookie, "/inventory?rarity=rare&effect=foil")
	require.Contains(t, empty, "No cards in this inventory match your search")
	require.Contains(t, empty, `<a href="/inventory?tab=cards">Clear search</a>`)

	// Unknown values are ignored
	unknown := getInventory(t, router, cookie, "/inventory?rarity=mythic&effect=sparkles")
	require.Contains(t, unknown, `<p class="inventory-count">5 cards</p>`)
}

// Titles in alphabetical order, for checks that don't depend on the sort
func sortedTitles(body string) []string {
	titles := cardTitles(body)
	slices.Sort(titles)
	return titles
}

func TestInventorySorts(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invsort1", "foo-bar-baz-qux", 0)
	day := func(d int) time.Time { return time.Date(2025, time.May, d, 12, 0, 0, 0, time.Local) }
	// Tungsten: newest, Pythagorean: oldest, McMurdo: most copies
	giveCardAt(t, "invsort1", cardTungsten, "", day(20))
	giveCardAt(t, "invsort1", cardPythagorean, "", day(1))
	giveCardAt(t, "invsort1", cardFordT, "", day(10))
	giveCardAt(t, "invsort1", cardMcMurdo, "", day(5))
	giveCardAt(t, "invsort1", cardMcMurdo, "", day(6))
	giveCardAt(t, "invsort1", cardMcMurdo, "", day(7))
	cookie := login(t, router, "invsort1", "foo-bar-baz-qux")

	cases := map[string][]string{
		// Last acquired by default
		"":             {"Tungsten", "Ford Model T", "McMurdo Station", "Pythagorean tiling"},
		"?sort=recent": {"Tungsten", "Ford Model T", "McMurdo Station", "Pythagorean tiling"},
		"?sort=oldest": {"Pythagorean tiling", "McMurdo Station", "Ford Model T", "Tungsten"},
		// Ties are broken by last acquired
		"?sort=rarity":    {"Ford Model T", "McMurdo Station", "Tungsten", "Pythagorean tiling"},
		"?sort=count":     {"McMurdo Station", "Tungsten", "Ford Model T", "Pythagorean tiling"},
		"?sort=name":      {"Ford Model T", "McMurdo Station", "Pythagorean tiling", "Tungsten"},
		"?sort=name-desc": {"Tungsten", "Pythagorean tiling", "McMurdo Station", "Ford Model T"},
		// Relevance needs a search
		"?sort=relevance": {"Tungsten", "Ford Model T", "McMurdo Station", "Pythagorean tiling"},
		"?sort=unknown":   {"Tungsten", "Ford Model T", "McMurdo Station", "Pythagorean tiling"},
	}
	for query, titles := range cases {
		body := getInventory(t, router, cookie, "/inventory?tab=cards"+strings.Replace(query, "?", "&", 1))
		require.Equal(t, titles, cardTitles(body), query)
	}

	body := getInventory(t, router, cookie, "/inventory?tab=cards")
	require.Contains(t, body, `<option value="" selected>Last acquired</option>`)
	require.NotContains(t, body, `value="relevance"`)

	// Searches rank by relevance unless a sort is picked
	search := getInventory(t, router, cookie, "/inventory?q=tungsten")
	require.Contains(t, search, `<option value="" selected>Relevance</option>`)
	require.Contains(t, search, `<option value="recent" >Last acquired</option>`)
	require.NotContains(t, search, `sort=`)
}

func TestInventorySortNameLocalized(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "invsort2", "foo-bar-baz-qux", 0)
	giveCards(t, "invsort2", cardTungsten, 1)
	giveCards(t, "invsort2", cardFordT, 1)
	cookie := login(t, router, "invsort2", "foo-bar-baz-qux")

	en := cardTitles(getInventory(t, router, cookie, "/inventory?sort=name"))
	fr := cardTitles(getInventory(t, router, cookie, "/inventory?sort=name&lang=fr"))
	require.Len(t, fr, 2)
	// Titles are sorted in the page's language
	require.True(t, slices.IsSorted(en))
	require.True(t, slices.IsSortedFunc(fr, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	}))
}
