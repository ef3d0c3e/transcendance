package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"transcendance/models"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func shopRequest(
	t *testing.T,
	router *gin.Engine,
	method string,
	target string,
	form url.Values,
	cookie *http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()

	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}

	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Accept-Language", "en")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func shopBalance(t *testing.T, username string) uint {
	t.Helper()

	var user models.User
	require.NoError(t, tests.DB.
		Where("username = ?", username).
		First(&user).
		Error)
	return user.Coins
}

// shopBoosters returns the boosters owned by a user, as the inventory sees them.
func shopBoosters(t *testing.T, username string) []models.Booster {
	t.Helper()

	var user models.User
	require.NoError(t, tests.DB.
		Where("username = ?", username).
		First(&user).
		Error)

	var boosters []models.Booster
	require.NoError(t, tests.DB.
		Where("owner_id = ?", user.ID).
		Order("id").
		Find(&boosters).
		Error)
	return boosters
}

// shopBody strips the bidi isolate characters Fluent wraps around interpolated
// values (U+2068/U+2069), so assertions can match plain text.
func shopBody(rec *httptest.ResponseRecorder) string {
	return strings.Map(func(r rune) rune {
		if r == 0x2068 || r == 0x2069 {
			return -1
		}
		return r
	}, rec.Body.String())
}

func TestShopGetListsBoosters(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shoplist", "foo-bar-qux", 0)
	cookie := login(t, router, "shoplist", "foo-bar-qux")

	// Boosters are sold from the inventory
	rec := shopRequest(t, router, http.MethodGet, "/inventory", nil, cookie)

	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, "Development")
	// Collection tile with its coin icon
	require.Contains(t, body, `class="coin-icon"`)
	require.Contains(t, body, "data-tilt")
	require.Contains(t, body, "/collections/1/thumbnail")
	require.Contains(t, body, "5 cards booster")
	// No booster left: the tile leads to the pack opening page, where some
	// are bought
	require.Contains(t, body, `href="/opening?collection=1"`)
	require.Contains(t, body, `data-booster-open="/opening?collection=1"`)
	require.Contains(t, body, `data-booster-label="Buy booster Development"`)
	// The collection progress under the tile, as on the collections tab
	require.Contains(t, body, `class="inventory-collection-progress"`)
	require.Contains(t, body, "<strong>0</strong> / 5")
	// No pack buying popup, no grayed tile and no price pill
	require.NotContains(t, body, `id="shop-buy-popup"`)
	require.NotContains(t, body, `class="inventory-booster is-empty`)
	require.NotContains(t, body, `shop-price-pill`)
	require.NotContains(t, body, `shop-booster-display`)
}

func TestShopGetRedirectsToInventory(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := shopRequest(t, router, http.MethodGet, "/shop", nil, nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory", rec.Header().Get("Location"))

	// Purchase notices follow
	rec = shopRequest(t, router, http.MethodGet, "/shop?topup=medium", nil, nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?topup=medium", rec.Header().Get("Location"))
}

func TestShopGetFollowsLocale(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	createAccount(t, router, "shoplocale", "foo-bar-qux", 0)
	cookie := login(t, router, "shoplocale", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodGet, "/inventory?lang=fr", nil, cookie)

	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, "Solde")
	require.Contains(t, body, "Développement")
	require.Contains(t, body, `data-booster-label="Acheter un booster Développement"`)
}

func TestShopTopupRequiresLogin(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"small"},
	}, nil)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestShopTopupUnknownPack(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shopcarol", "foo-bar-qux", 0)
	cookie := login(t, router, "shopcarol", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"diamond"},
	}, cookie)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?error=pack", rec.Header().Get("Location"))
	require.Equal(t, uint(0), shopBalance(t, "shopcarol"))
}

func TestShopTopupJSON(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shopjson", "foo-bar-qux", 0)
	cookie := login(t, router, "shopjson", "foo-bar-qux")

	topup := func(pack string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/shop/topup", strings.NewReader(url.Values{
			"pack": {pack},
		}.Encode()))
		req.Header.Set("Accept-Language", "en")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// The opening page tops up without leaving: the new balance, no redirect
	rec := topup("medium")
	require.Equal(t, http.StatusOK, rec.Code)
	var result struct {
		Coins      uint   `json:"coins"`
		CoinsLabel string `json:"coinsLabel"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Equal(t, uint(500), result.Coins)
	require.Equal(t, "500 coins", stripIsolation(result.CoinsLabel))
	require.Equal(t, uint(500), shopBalance(t, "shopjson"))

	rec = topup("unknown")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, uint(500), shopBalance(t, "shopjson"))
}

func TestShopTopupCreditsCoins(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shopdave", "foo-bar-qux", 0)
	cookie := login(t, router, "shopdave", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"medium"},
	}, cookie)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	requireSignedNotice(t, "/inventory?topup=medium", rec.Header().Get("Location"))
	require.Equal(t, uint(500), shopBalance(t, "shopdave"))

	// The shop shows the new balance and confirms the checkout
	rec = shopRequest(t, router, http.MethodGet, rec.Header().Get("Location"), nil, cookie)
	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, `Balance: <span id="shop-coins">500 coins</span>`)
	require.Contains(t, body, "500 coins have been added to your balance")
	require.Contains(t, body, "/shop/topup")
	// Packs are sold from a popup, each with a display price in euros
	require.Contains(t, body, `id="shop-topup-popup"`)
	require.Contains(t, body, "Buy more coins")
	require.Contains(t, body, "€4.49")
	// The biggest packs
	require.Contains(t, body, "5,000 coins")
	require.Contains(t, body, "€29.99")
	require.Contains(t, body, "10,000 coins")
	require.Contains(t, body, "€49.99")

	// A second pack stacks on top of the first
	rec = shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"small"},
	}, cookie)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, uint(600), shopBalance(t, "shopdave"))
}

func TestShopGetShowsCollectionTile(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shoptile", "foo-bar-qux", 0)
	cookie := login(t, router, "shoptile", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodGet, "/inventory", nil, cookie)

	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, `class="card-container"`)
	require.Contains(t, body, `class="collection"`)
	require.Contains(t, body, "--c1: #ff7f5f")
	require.Contains(t, body, "--c2: #7f2fff")
	require.Contains(t, body, "5 CARDS")
	require.Contains(t, body, "NO. 001")
}

func TestShopGetLoggedInShowsCoins(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shoperin", "foo-bar-qux", 0)
	cookie := login(t, router, "shoperin", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"large"},
	}, cookie)
	require.Equal(t, http.StatusSeeOther, rec.Code)

	rec = shopRequest(t, router, http.MethodGet, "/inventory", nil, cookie)
	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, `Balance: <span id="shop-coins">1,000 coins</span>`)
	// With the top-up popup, to buy more
	require.Contains(t, body, `class="shop-topup-open"`)
	require.Contains(t, body, `id="shop-topup-popup"`)
}

func TestCollectionThumbnailServed(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	// The shop's booster tiles load this URL
	rec := shopRequest(t, router, http.MethodGet, "/collections/1/thumbnail", nil, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = shopRequest(t, router, http.MethodGet, "/collections/999/thumbnail", nil, nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCardAssetsServed(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	// The collection tile's foil mask loads this URL
	rec := shopRequest(t, router, http.MethodGet, "/cards/assets/foil1.png", nil, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// Asset names must not escape data/assets (dot segments are rejected)
	rec = shopRequest(t, router, http.MethodGet, "/cards/assets/..", nil, nil)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	rec = shopRequest(t, router, http.MethodGet, "/cards/assets/nope.png", nil, nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}
