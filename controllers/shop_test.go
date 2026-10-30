package controllers_test

import (
	"encoding/json"
	"fmt"
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
	require.Contains(t, body, "250 coins")
	// Collection tile with its price pill and coin icon
	require.Contains(t, body, `class="shop-booster-display"`)
	require.Contains(t, body, `class="shop-price-pill"`)
	require.Contains(t, body, `class="coin-icon"`)
	require.Contains(t, body, "data-tilt")
	require.Contains(t, body, "/collections/1/thumbnail")
	require.Contains(t, body, "5 cards booster")
	// Buy popup mount, with the tile price the popup compares to the balance
	require.Contains(t, body, `id="shop-buy-popup"`)
	require.Contains(t, body, `data-price="250"`)
	require.Contains(t, body, `data-label-buy="Buy"`)
	// And the bulk quantity cap the popup's stepper must not exceed
	require.Contains(t, body, `data-max-quantity="10"`)
	require.Contains(t, body, `data-label-quantity="Quantity"`)
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
	require.Contains(t, body, "250 pièces")
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
	require.Equal(t, "/inventory?topup=medium", rec.Header().Get("Location"))
	require.Equal(t, uint(500), shopBalance(t, "shopdave"))

	// The shop shows the new balance and confirms the checkout
	rec = shopRequest(t, router, http.MethodGet, "/inventory?topup=medium", nil, cookie)
	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, "Balance: 500 coins")
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
	require.Contains(t, body, "Balance: 1,000 coins")
	require.Contains(t, body, `data-coins="1000"`)
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

func TestShopBuyRequiresLogin(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := shopRequest(t, router, http.MethodPost, "/shop/buy", url.Values{
		"collection": {"1"},
	}, nil)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestShopBuyUnknownCollection(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shopbuy", "foo-bar-qux", 0)
	cookie := login(t, router, "shopbuy", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodPost, "/shop/buy", url.Values{
		"collection": {"999"},
	}, cookie)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?error=collection", rec.Header().Get("Location"))
	require.Equal(t, uint(0), shopBalance(t, "shopbuy"))
	require.Empty(t, shopBoosters(t, "shopbuy"))
}

func TestShopBuyInsufficientFunds(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shopfiona", "foo-bar-qux", 0)
	cookie := login(t, router, "shopfiona", "foo-bar-qux")

	// One booster costs 250 coins, the small pack only credits 100
	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"small"},
	}, cookie)
	require.Equal(t, http.StatusSeeOther, rec.Code)

	rec = shopRequest(t, router, http.MethodPost, "/shop/buy", url.Values{
		"collection": {"1"},
	}, cookie)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?error=funds", rec.Header().Get("Location"))
	require.Equal(t, uint(100), shopBalance(t, "shopfiona"))
	require.Empty(t, shopBoosters(t, "shopfiona"))
}

func TestShopBuyAddsBoosterToInventory(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shopgail", "foo-bar-qux", 0)
	cookie := login(t, router, "shopgail", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"medium"},
	}, cookie)
	require.Equal(t, http.StatusSeeOther, rec.Code)

	rec = shopRequest(t, router, http.MethodPost, "/shop/buy", url.Values{
		"collection": {"1"},
	}, cookie)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	// 500 credited, one 250 coin booster bought
	require.Equal(t, uint(250), shopBalance(t, "shopgail"))

	// The booster lands in the inventory, and the buyer on its opening
	boosters := shopBoosters(t, "shopgail")
	require.Len(t, boosters, 1)
	require.Equal(t, uint(1), boosters[0].CollectionID)
	require.Equal(t, uint(5), boosters[0].Size)
	require.Equal(t, fmt.Sprintf("/opening?booster=%d", boosters[0].ID), rec.Header().Get("Location"))
}

func TestShopBuyBulkAddsBoostersToInventory(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shophank", "foo-bar-qux", 0)
	cookie := login(t, router, "shophank", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"large"},
	}, cookie)
	require.Equal(t, http.StatusSeeOther, rec.Code)

	rec = shopRequest(t, router, http.MethodPost, "/shop/buy", url.Values{
		"collection": {"1"},
		"quantity":   {"3"},
	}, cookie)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	// 1000 credited, three 250 coin boosters bought
	require.Equal(t, uint(250), shopBalance(t, "shophank"))

	boosters := shopBoosters(t, "shophank")
	require.Len(t, boosters, 3)
	for _, booster := range boosters {
		require.Equal(t, uint(1), booster.CollectionID)
		require.Equal(t, uint(5), booster.Size)
	}

	// The first one opens, the page chains into the others
	require.Equal(t, fmt.Sprintf("/opening?booster=%d", boosters[0].ID), rec.Header().Get("Location"))
}

func TestShopBuyRejectsInvalidQuantity(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shopiris", "foo-bar-qux", 0)
	cookie := login(t, router, "shopiris", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"medium"},
	}, cookie)
	require.Equal(t, http.StatusSeeOther, rec.Code)

	// Zero, above the cap, negative and not even a number are all refused
	for _, quantity := range []string{"0", "11", "-2", "many"} {
		rec := shopRequest(t, router, http.MethodPost, "/shop/buy", url.Values{
			"collection": {"1"},
			"quantity":   {quantity},
		}, cookie)

		require.Equal(t, http.StatusSeeOther, rec.Code)
		require.Equal(t, "/inventory?error=quantity", rec.Header().Get("Location"), "quantity %q", quantity)
	}

	// Nothing was sold and the balance is untouched
	require.Equal(t, uint(500), shopBalance(t, "shopiris"))
	require.Empty(t, shopBoosters(t, "shopiris"))

	// The shop explains the range it accepts
	rec = shopRequest(t, router, http.MethodGet, "/inventory?error=quantity", nil, cookie)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, shopBody(rec), "Enter a quantity between 1 and 10")
}

func TestShopBuyInsufficientFundsForBulk(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shopjude", "foo-bar-qux", 0)
	cookie := login(t, router, "shopjude", "foo-bar-qux")

	// 500 coins cover two boosters but not three
	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"medium"},
	}, cookie)
	require.Equal(t, http.StatusSeeOther, rec.Code)

	rec = shopRequest(t, router, http.MethodPost, "/shop/buy", url.Values{
		"collection": {"1"},
		"quantity":   {"3"},
	}, cookie)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?error=funds", rec.Header().Get("Location"))
	// The whole order is refused, not just the boosters that do not fit
	require.Equal(t, uint(500), shopBalance(t, "shopjude"))
	require.Empty(t, shopBoosters(t, "shopjude"))
}
