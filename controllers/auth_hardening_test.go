package controllers_test

import (
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

func postLogin(router *gin.Engine, username string, password string, header http.Header) *httptest.ResponseRecorder {
	form := url.Values{"username": {username}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, values := range header {
		req.Header[key] = values
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRegisterInvalidEmail(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	for i, email := range []string{"hello", "a@", "@example.test", "Bob <bob@example.test>", "a b@example.test"} {
		form := url.Values{
			"username":         {fmt.Sprintf("bademail%d", i)},
			"email":            {email},
			"password":         {"foo-bar-qux"},
			"password_confirm": {"foo-bar-qux"},
			"tos_agree":        {"on"},
		}
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code, email)
	}

	// Surrounding spaces are not part of the address
	createAccount(t, router, "goodemail", "foo-bar-qux", 0)
	var user models.User
	require.NoError(t, tests.DB.Where("username = ?", "goodemail").First(&user).Error)
	require.Equal(t, "goodemail@example.test", user.Email)
}

func TestProfileInvalidEmail(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "profemail1", "foo-bar-qux", 0)
	cookie := login(t, router, "profemail1", "foo-bar-qux")

	rec := postForm(t, router, cookie, "/profile", url.Values{"email": {"not-an-address"}})
	require.Equal(t, http.StatusBadRequest, rec.Code)
	rec = postForm(t, router, cookie, "/profile", url.Values{"email": {" profemail1-new@example.test "}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var user models.User
	require.NoError(t, tests.DB.Where("username = ?", "profemail1").First(&user).Error)
	require.Equal(t, "profemail1-new@example.test", user.Email)
}

// Password guessing is slowed down per account, the right password included
func TestLoginRateLimited(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "ratelim1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "ratelim2", "foo-bar-baz-qux", 0)

	for range 5 {
		require.Equal(t, http.StatusBadRequest, postLogin(router, "ratelim1", "wrong-password", nil).Code)
	}
	rec := postLogin(router, "ratelim1", "foo-bar-baz-qux", nil)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotEmpty(t, rec.Header().Get("Retry-After"))
	require.Contains(t, rec.Body.String(), "Too many failed attempts")

	// Other accounts are not locked out by someone else's mistakes
	require.Equal(t, http.StatusOK, postLogin(router, "ratelim2", "foo-bar-baz-qux", nil).Code)
}

// Unknown accounts fail like wrong passwords, and count as attempts too
func TestLoginUnknownAccount(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	for range 5 {
		rec := postLogin(router, "nobody_here", "some-password", nil)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Contains(t, rec.Body.String(), "Invalid login credentials")
	}
	require.Equal(t, http.StatusTooManyRequests, postLogin(router, "nobody_here", "some-password", nil).Code)
}

// Pages submitting with fetch get 200 for expected errors, so browsers log nothing in the console
func TestSoftErrors(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "softerr1", "foo-bar-baz-qux", 0)

	rec := postLogin(router, "softerr1", "wrong-password", http.Header{"X-Soft-Errors": {"1"}})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "400", rec.Header().Get("X-Status"))
	require.Contains(t, rec.Body.String(), "Invalid login credentials")

	rec = postLogin(router, "softerr1", "foo-bar-baz-qux", http.Header{"X-Soft-Errors": {"1"}})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Header().Get("X-Status"))

	// Without the header, the status is the real one
	require.Equal(t, http.StatusBadRequest, postLogin(router, "softerr1", "wrong-password", nil).Code)
}

func TestSessionCookie(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "cookie1", "foo-bar-baz-qux", 0)

	// Behind the gateway, which terminates TLS, the cookie is never sent over plain HTTP
	rec := postLogin(router, "cookie1", "foo-bar-baz-qux", http.Header{"X-Forwarded-Proto": {"https"}})
	require.Equal(t, http.StatusOK, rec.Code)
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_token" {
			cookie = c
		}
	}
	require.NotNil(t, cookie)
	require.True(t, cookie.Secure)
	require.True(t, cookie.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)

	// The database only holds a hash of the token
	var stored []string
	require.NoError(t, tests.DB.Model(&models.UserSession{}).
		Where("user_id = ?", userID(t, "cookie1")).Pluck("token", &stored).Error)
	require.Equal(t, []string{models.HashSessionToken(cookie.Value)}, stored)
	require.NotContains(t, stored, cookie.Value)
}

// Without JavaScript, the logout form brings the user back home
func TestLogoutForm(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "logoutform1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "logoutform1", "foo-bar-baz-qux")

	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*;q=0.8")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/", rec.Header().Get("Location"))
	require.Contains(t, strings.Join(rec.Header().Values("Set-Cookie"), "\n"), "session_token=;")
}

// Admins manage the ranks of others, never their own
func TestProfileOwnRank(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "ownrank1", "foo-bar-baz-qux", 2)
	cookie := login(t, router, "ownrank1", "foo-bar-baz-qux")

	rec := postForm(t, router, cookie, "/profile", url.Values{"rank": {"0"}})
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "You cannot change your own rank.")
	require.Equal(t, models.RankAdmin, rankOf(t, "ownrank1"))

	// Their own profile has no rank field
	body := getPageAs(t, router, cookie, "/profile")
	require.NotContains(t, body, `name="rank"`)
}

// There is always an administrator left
func TestLastAdminCannotLeave(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	// Earlier tests leave admins behind
	require.NoError(t, tests.DB.Model(&models.User{}).Where("`rank` = ?", models.RankAdmin).
		Update("rank", models.RankUser).Error)
	createAccount(t, router, "lastadmin1", "foo-bar-baz-qux", 2)
	cookie := login(t, router, "lastadmin1", "foo-bar-baz-qux")
	target := fmt.Sprintf("/profiles/%d/delete", userID(t, "lastadmin1"))

	rec := postForm(t, router, cookie, target, nil)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), "only administrator")

	createAccount(t, router, "lastadmin2", "foo-bar-baz-qux", 2)
	rec = postForm(t, router, cookie, target, nil)
	require.Equal(t, http.StatusNoContent, rec.Code)
}

// Browsers count a line break once in a textarea's maxlength, and send it as CRLF
func TestProfileBioLineBreaks(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "biochars1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "biochars1", "foo-bar-baz-qux")

	rec := postForm(t, router, cookie, "/profile", url.Values{"bio": {strings.Repeat("a\r\n", 122)}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var profile models.UserProfile
	require.NoError(t, tests.DB.Where("user_id = ?", userID(t, "biochars1")).First(&profile).Error)
	require.Equal(t, strings.Repeat("a\n", 122), profile.Bio)
}

// Actions only go back to pages of the site
func TestActionRedirectStaysLocal(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "redirect1", "123456789", 0)
	cookie := login(t, router, "redirect1", "123456789")

	for _, target := range []string{"//evil.example", "/\\evil.example", "/\t/evil.example", "https://evil.example"} {
		rec := notificationRequest(t, router, cookie, http.MethodPost, "/api/notifications/read_all",
			url.Values{"redirect": {target}})
		require.Equal(t, http.StatusNoContent, rec.Code, target)
	}
	rec := notificationRequest(t, router, cookie, http.MethodPost, "/api/notifications/read_all",
		url.Values{"redirect": {"/notifications?p=2"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/notifications?p=2", rec.Header().Get("Location"))
}

func rankOf(t *testing.T, username string) uint {
	var user models.User
	require.NoError(t, tests.DB.Select("rank").Where("username = ?", username).First(&user).Error)
	return user.Rank
}

// The school's network shares one address: failures on other accounts from it never block a login
func TestLoginNotLimitedPerAddress(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sharedip1", "foo-bar-baz-qux", 0)
	for i := range 30 {
		require.Equal(t, http.StatusBadRequest, postLogin(router, fmt.Sprintf("someone%d", i), "wrong-password", nil).Code)
	}
	require.Equal(t, http.StatusOK, postLogin(router, "sharedip1", "foo-bar-baz-qux", nil).Code)
}
