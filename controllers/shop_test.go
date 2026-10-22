package controllers_test

import (
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

	rec := shopRequest(t, router, http.MethodGet, "/shop", nil, nil)

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
	// Buying is not wired up yet: the popup only mounts here
	require.Contains(t, body, `id="shop-buy-popup"`)
	require.Contains(t, body, "Coming soon")
	require.NotContains(t, body, "/shop/buy")
}

func TestShopGetAnonymousHidesCheckout(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := shopRequest(t, router, http.MethodGet, "/shop", nil, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, "Log in to get coins and buy boosters")
	require.NotContains(t, body, "/shop/topup")
	require.NotContains(t, body, "Balance:")
	// The buy popup must not advertise a balance to guests
	require.NotContains(t, body, "data-coins")
}

func TestShopGetFollowsLocale(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := shopRequest(t, router, http.MethodGet, "/shop?lang=fr", nil, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, "Boutique")
	require.Contains(t, body, "Développement")
	require.Contains(t, body, "250 pièces")
}

func TestShopTopupRequiresLogin(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"small"},
	}, nil)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/shop?error=auth", rec.Header().Get("Location"))
}

func TestShopTopupUnknownPack(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shopcarol", "foo-bar-qux", 0)
	cookie := login(t, router, "shopcarol", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"diamond"},
	}, cookie)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/shop?error=pack", rec.Header().Get("Location"))
	require.Equal(t, uint(0), shopBalance(t, "shopcarol"))
}

func TestShopTopupCreditsCoins(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "shopdave", "foo-bar-qux", 0)
	cookie := login(t, router, "shopdave", "foo-bar-qux")

	rec := shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"medium"},
	}, cookie)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/shop?topup=medium", rec.Header().Get("Location"))
	require.Equal(t, uint(500), shopBalance(t, "shopdave"))

	// The shop shows the new balance and confirms the checkout
	rec = shopRequest(t, router, http.MethodGet, "/shop?topup=medium", nil, cookie)
	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, "Balance: 500")
	require.Contains(t, body, "500 coins have been added to your balance")
	require.Contains(t, body, "/shop/topup")

	// A second pack stacks on top of the first
	rec = shopRequest(t, router, http.MethodPost, "/shop/topup", url.Values{
		"pack": {"small"},
	}, cookie)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, uint(600), shopBalance(t, "shopdave"))
}

func TestShopGetShowsCollectionTile(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := shopRequest(t, router, http.MethodGet, "/shop", nil, nil)

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

	rec = shopRequest(t, router, http.MethodGet, "/shop", nil, cookie)
	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, "Balance: 1000")
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
