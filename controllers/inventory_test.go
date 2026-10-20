package controllers_test

import (
	"net/http"
	"net/http/httptest"
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
	cardFlyingScotsman = 1002
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
	require.Equal(t, 2, countCard(body, "Ford Model T"))
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
	giveCards(t, "invsearch5", cardFordT, 25)
	giveCards(t, "invsearch5", cardTungsten, 3)
	cookie := login(t, router, "invsearch5", "foo-bar-baz-qux")

	first := getInventory(t, router, cookie, "/inventory?q=ford")
	require.Equal(t, 20, countCard(first, "Ford Model T"))
	require.Contains(t, first, `href="?q=ford&amp;p=2"`)
	require.Contains(t, first, `<p class="inventory-count">25 cards found</p>`)
	// Multiple pages get a second paginator below the cards
	require.Equal(t, 2, strings.Count(first, `<nav class="paginator"`))

	second := getInventory(t, router, cookie, "/inventory?q=ford&p=2")
	require.Equal(t, 5, countCard(second, "Ford Model T"))
	require.Zero(t, countCard(second, "Tungsten"))
	require.Contains(t, second, `<p class="inventory-count">25 cards found</p>`)

	// Without a search, the count covers the whole inventory
	all := getInventory(t, router, cookie, "/inventory")
	require.Contains(t, all, `<p class="inventory-count">28 cards</p>`)
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
