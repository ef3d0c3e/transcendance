package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Get a page as a visitor, without bidi isolation marks
func getPage(t *testing.T, router *gin.Engine, target string) string {
	req := httptest.NewRequest(http.MethodGet, target, nil)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	return strings.NewReplacer("\u2068", "", "\u2069", "").Replace(rec.Body.String())
}

func TestCatalogListsEveryCard(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	body := getPage(t, router, "/cards/search/")
	// Collections come in order, the dev collection first
	require.Equal(t, []string{
		"Pythagorean tiling", "Flying Scotsman", "McMurdo Station", "Ford Model T", "Tungsten",
	}, cardTitles(body)[:5])
	require.Contains(t, body, fmt.Sprintf(`<p class="inventory-count">%d cards</p>`, len(tests.Data.Cards)))
	require.Contains(t, body, `<a class="inventory-tab-cards" href="/cards/search/" aria-current="page">`)
	// Relevance needs a search
	require.NotContains(t, body, `value="relevance"`)
}

func TestCatalogSearchFiltersAndSorts(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	// Within the dev collection, other collections have their own cards
	cases := map[string][]string{
		"/cards/search/?collection=1&rarity=common":                {"Pythagorean tiling", "Flying Scotsman", "Tungsten"},
		"/cards/search/?collection=1&rarity=common&sort=name-desc": {"Tungsten", "Pythagorean tiling", "Flying Scotsman"},
		"/cards/search/?collection=1&sort=rarity":                  {"Ford Model T", "McMurdo Station", "Pythagorean tiling", "Flying Scotsman", "Tungsten"},
		"/cards/search/?collection=1&sort=name":                    {"Flying Scotsman", "Ford Model T", "McMurdo Station", "Pythagorean tiling", "Tungsten"},
		"/cards/search/?q=tungsten":                                {"Tungsten"},
		"/cards/search/?q=tungsten&rarity=rare":                    nil,
		"/cards/search/?collection=1&rarity=rare":                  {"McMurdo Station"},
		// Unknown filters are ignored
		"/cards/search/?collection=999&rarity=mythic": cardTitles(getPage(t, router, "/cards/search/")),
	}
	for target, titles := range cases {
		body := getPage(t, router, target)
		require.Equal(t, titles, cardTitles(body), target)
	}

	searched := getPage(t, router, "/cards/search/?q=tungsten")
	require.Contains(t, searched, `<p class="inventory-count">1 card found</p>`)
	// The search follows to the collections tab
	require.Contains(t, searched, `href="/collections/search/?q=tungsten"`)

	empty := getPage(t, router, "/cards/search/?q=tungsten&rarity=rare")
	require.Contains(t, empty, "No cards found")
	require.Contains(t, empty, `<a href="/cards/search/">Clear search</a>`)
}

func TestCatalogCardsPaginate(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	// Past the last page shows nothing, but keeps the filters in the paginator
	body := getPage(t, router, "/cards/search/?collection=1&rarity=common&p=2")
	require.Empty(t, cardTitles(body))
	require.Contains(t, body, "No cards found")
}

func TestCatalogCollectionsTab(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	body := getPage(t, router, "/collections/search/")
	require.Contains(t, body, `<a class="catalog-collection" href="/collections/1/">`)
	require.Contains(t, body, fmt.Sprintf(`<p class="inventory-count">%d collections</p>`, len(tests.Data.Collections)))
	require.Contains(t, body, `<a class="inventory-tab-boosters" href="/collections/search/" aria-current="page">`)
	// Card filters are only on the cards tab
	require.NotContains(t, body, `name="rarity"`)
}

func TestCollectionShowsProgress(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "colprogress1", "foo-bar-baz-qux", 0)
	giveCards(t, "colprogress1", cardTungsten, 2)
	giveCards(t, "colprogress1", cardFordT, 1)
	cookie := login(t, router, "colprogress1", "foo-bar-baz-qux")

	// 2 of the 5 dev cards collected, the 3 others are grayed out
	body := getInventory(t, router, cookie, "/collections/1/")
	require.Contains(t, body, `<p class="pack-progress-count"><strong>2</strong> / 5</p>`)
	require.Contains(t, body, `<div class="pack-progress-bar" style="width: 40%"></div>`)
	require.Contains(t, body, `class="pack-progress is-static"`)
	require.Equal(t, 3, strings.Count(body, `class="card-container is-missing"`))
	require.Equal(t, 5, strings.Count(body, `<article class="card"`))

	// Visitors see every card as is, without progress
	visitor := getPage(t, router, "/collections/1/")
	require.NotContains(t, visitor, "is-missing")
	require.NotContains(t, visitor, "pack-progress")
}

func TestCollectionCardsFilterAndSort(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	// The collection's cards, with the catalog filters applied within it
	cases := map[string][]string{
		"/collections/1/": {"Pythagorean tiling", "Flying Scotsman", "McMurdo Station", "Ford Model T", "Tungsten"},
		"/collections/1/?rarity=common&sort=name-desc": {"Tungsten", "Pythagorean tiling", "Flying Scotsman"},
		"/collections/1/?rarity=rare":                  {"McMurdo Station"},
		"/collections/1/?q=tungsten":                   {"Tungsten"},
		// The collection can't be swapped through the query
		"/collections/1/?collection=2&rarity=rare": {"McMurdo Station"},
	}
	for target, titles := range cases {
		body := getPage(t, router, target)
		require.Equal(t, titles, cardTitles(body), target)
	}

	body := getPage(t, router, "/collections/1/?rarity=rare")
	require.Contains(t, body, `action="/collections/1/"`)

	empty := getPage(t, router, "/collections/1/?q=tungsten&rarity=rare")
	require.Contains(t, empty, `<a href="/collections/1/">Clear search</a>`)
	require.Contains(t, body, `name="rarity"`)
	// No tabs or collection picker inside a collection
	require.NotContains(t, body, `name="collection"`)
	require.NotContains(t, body, `inventory-tab-cards`)
}
