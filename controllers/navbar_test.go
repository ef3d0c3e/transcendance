package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"transcendance/models"
	"transcendance/tests"

	"github.com/stretchr/testify/require"
)

func TestNavbarVisitor(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	body := getPage(t, router, "/cards/search/")
	require.Contains(t, body, `id="login-popup"`)
	require.Contains(t, body, `id="register-popup"`)
	// Cards are browsed from the inventory
	require.NotContains(t, body, `>Cards</a></li>`)
	// Boosters are bought from the inventory, there is no shop page
	require.NotContains(t, body, `href="/shop"`)
	// Visitors have no inventory or account
	require.NotContains(t, body, `href="/inventory"`)
	require.NotContains(t, body, `class="nav-user"`)
	require.NotContains(t, body, "data-logout")
}

func TestNavbarAuthenticated(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "navbar1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "navbar1", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/cards/search/?q=rosetta")
	require.Contains(t, body, `<a class="nav-user" href="/profile"`)
	require.Contains(t, body, `<span>navbar1</span>`)
	require.Contains(t, body, `data-logout="/api/logout"`)
	require.Contains(t, body, `<li><a href="/inventory" >Inventory</a></li>`)
	require.NotContains(t, body, `>Cards</a></li>`)
	require.NotContains(t, body, `id="login-popup"`)
	require.NotContains(t, body, `id="register-popup"`)
}

// The header shows the coin balance of the account on every page
func TestNavbarShowsBalance(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "navcoin", "foo-bar-baz-qux", 0)
	require.NoError(t, tests.DB.Model(&models.User{}).
		Where("username = ?", "navcoin").
		Update("coins", 1234).Error)

	cookie := login(t, router, "navcoin", "foo-bar-baz-qux")
	body := getInventory(t, router, cookie, "/cards/search/")
	require.Contains(t, body, `<span class="nav-balance">`)
	require.Contains(t, body, "Balance: 1,234")

	// Visitors have no balance to show
	body = getPage(t, router, "/cards/search/")
	require.NotContains(t, body, "nav-balance")
}

func TestLogoutDeletesSession(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "logout1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "logout1", "foo-bar-baz-qux")

	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var sessions int64
	require.NoError(t, tests.DB.Model(&models.UserSession{}).
		Where("token = ?", models.HashSessionToken(cookie.Value)).Count(&sessions).Error)
	require.Zero(t, sessions)

	// Logging out changes state, it is not a link other sites could make the browser follow
	req = httptest.NewRequest(http.MethodGet, "/api/logout", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)

	// The old cookie no longer signs in
	body := getPage(t, router, "/cards/search/")
	require.Contains(t, body, `id="login-popup"`)
}
