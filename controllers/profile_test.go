package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"transcendance/models"
	"transcendance/tests"

	"github.com/stretchr/testify/require"
)

func userProfile(t *testing.T, username string) models.UserProfile {
	var profile models.UserProfile
	require.NoError(t, tests.DB.Where("user_id = ?", userID(t, username)).First(&profile).Error)
	return profile
}

func TestProfileUnauthenticatedJSON(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := postForm(t, router, nil, "/profile", url.Values{"bio": {"hello"}})
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	require.JSONEq(t, `{"message":"You must be logged in to access this resource"}`, stripIsolation(rec.Body.String()))
}

func TestProfileUsernameCase(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "Profcase", "foo-bar-baz-qux", 0)
	createAccount(t, router, "proftaken", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "Profcase", "foo-bar-baz-qux")

	// Changing the case only is not taken by the user's own account
	rec := postForm(t, router, cookie, "/profile", url.Values{"username": {"profcase"}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	userID(t, "profcase")

	rec = postForm(t, router, cookie, "/profile", url.Values{"username": {"PROFTAKEN"}})
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Username already taken")
}

func TestProfileBirthDate(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "profbirth", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "profbirth", "foo-bar-baz-qux")

	rec := postForm(t, router, cookie, "/profile", url.Values{"birthdate": {"2000-02-03"}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "2000-02-03", userProfile(t, "profbirth").BirthDate.Format("2006-01-02"))

	// An empty date clears it
	rec = postForm(t, router, cookie, "/profile", url.Values{"birthdate": {""}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.True(t, userProfile(t, "profbirth").BirthDate.IsZero())
	require.Contains(t, getInventory(t, router, cookie, "/profile"), `id="birthdate"`)

	// Today somewhere on Earth is accepted, later dates are not
	ahead := time.Now().UTC().Add(14 * time.Hour).Format("2006-01-02")
	rec = postForm(t, router, cookie, "/profile", url.Values{"birthdate": {ahead}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	future := time.Now().UTC().Add(48 * time.Hour).Format("2006-01-02")
	rec = postForm(t, router, cookie, "/profile", url.Values{"birthdate": {future}})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProfileLengthsInBytes(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "proflength", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "proflength", "foo-bar-baz-qux")

	// 122 two-byte characters, 244 bytes
	bio := strings.Repeat("é", 122)
	rec := postForm(t, router, cookie, "/profile", url.Values{"bio": {bio}, "contact": {strings.Repeat("ü", 128)}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, bio, userProfile(t, "proflength").Bio)

	rec = postForm(t, router, cookie, "/profile", url.Values{"bio": {bio + "é"}})
	require.Equal(t, http.StatusBadRequest, rec.Code)
	rec = postForm(t, router, cookie, "/profile", url.Values{"contact": {strings.Repeat("ü", 129)}})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProfilePasswordRevokesOtherSessions(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "profpass", "foo-bar-baz-qux", 0)
	current := login(t, router, "profpass", "foo-bar-baz-qux")
	other := login(t, router, "profpass", "foo-bar-baz-qux")

	rec := postForm(t, router, current, "/profile", url.Values{
		"password":         {"foo-bar-baz-qux"},
		"new_password":     {"qux-baz-bar-foo"},
		"confirm_password": {"qux-baz-bar-foo"},
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var tokens []string
	require.NoError(t, tests.DB.Model(&models.UserSession{}).
		Where("user_id = ?", userID(t, "profpass")).
		Pluck("token", &tokens).Error)
	require.Equal(t, []string{current.Value}, tokens)
	require.NotEqual(t, current.Value, other.Value)
}

// The inventory button shows on other users' profiles only
func TestProfileInventoryButton(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "profbtn1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "profbtn2", "foo-bar-baz-qux", 0)
	createAccount(t, router, "profbtnadm", "foo-bar-baz-qux", 2)

	button := `<a class="profile-inventory" href="/inventory?u=profbtn2">View inventory</a>`

	cookie := login(t, router, "profbtn1", "foo-bar-baz-qux")

	// Someone else's profile links to their inventory
	require.Contains(t, getInventory(t, router, cookie, "/profile/profbtn2"), button)

	// One's own profile has none, in edit or read-only form
	own := getInventory(t, router, cookie, "/profile")
	require.NotContains(t, own, button)
	require.NotContains(t, own, "View inventory")
	require.NotContains(t, getInventory(t, router, cookie, "/profile/profbtn1"), button)

	// An admin editing another user's profile gets the button too
	admin := login(t, router, "profbtnadm", "foo-bar-baz-qux")
	require.Contains(t, getInventory(t, router, admin, "/profile/profbtn2"), button)
	require.NotContains(t, getInventory(t, router, admin, "/profile"), "View inventory")
}

func TestProfileOriginBehindGateway(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "proforigin", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "proforigin", "foo-bar-baz-qux")

	post := func(origin string) int {
		req := httptest.NewRequest(http.MethodPost, "/profile", strings.NewReader(url.Values{"bio": {"hello"}}.Encode()))
		req.Host = "example.com"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", origin)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	// The gateway forwards the host without the port the browser used
	require.Equal(t, http.StatusOK, post("https://example.com:8443"))
	require.Equal(t, http.StatusForbidden, post("https://evil.example:8443"))
}
