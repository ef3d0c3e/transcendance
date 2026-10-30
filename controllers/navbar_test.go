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

	body := getPage(t, router, "/shop")
	require.Contains(t, body, `id="login-popup"`)
	require.Contains(t, body, `id="register-popup"`)
	require.Contains(t, body, `<li><a href="/cards/search/" >Cards</a></li>`)
	require.Contains(t, body, `<li><a href="/shop" aria-current="page">Shop</a></li>`)
	// Visitors have no inventory or account
	require.NotContains(t, body, `href="/inventory"`)
	require.NotContains(t, body, `class="nav-user"`)
	require.NotContains(t, body, "data-logout")
}

func TestNavbarAuthenticated(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "navbar1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "navbar1", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/cards/search/?q=tungsten")
	require.Contains(t, body, `<a class="nav-user" href="/profile"`)
	require.Contains(t, body, `<span>navbar1</span>`)
	require.Contains(t, body, `data-logout="/api/logout"`)
	require.Contains(t, body, `<li><a href="/inventory" >Inventory</a></li>`)
	require.Contains(t, body, `<li><a href="/cards/search/" aria-current="page">Cards</a></li>`)
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
	body := getInventory(t, router, cookie, "/shop")
	require.Contains(t, body, `<span class="nav-balance">`)
	require.Contains(t, body, "Balance: 1,234")

	// Visitors have no balance to show
	body = getPage(t, router, "/shop")
	require.NotContains(t, body, "nav-balance")
}

func TestLogoutDeletesSession(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "logout1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "logout1", "foo-bar-baz-qux")

	req := httptest.NewRequest(http.MethodGet, "/api/logout", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var sessions int64
	require.NoError(t, tests.DB.Model(&models.UserSession{}).Where("token = ?", cookie.Value).Count(&sessions).Error)
	require.Zero(t, sessions)

	// The old cookie no longer signs in
	body := getInventory(t, router, cookie, "/shop")
	require.Contains(t, body, `id="login-popup"`)
}
