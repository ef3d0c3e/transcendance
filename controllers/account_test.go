package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"transcendance/models"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createAccount(t *testing.T, router *gin.Engine, username string, password string, rank uint) {
	form := url.Values{
		"username":         {username},
		"email":            {username + "@example.test"},
		"password":         {password},
		"password_confirm": {password},
		"tos_agree":        {"on"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var user models.User
	err := tests.DB.
		Where("username = ?", username).
		First(&user).
		Error

	require.NoError(t, err)
	require.Equal(t, username, user.Username)
	require.NotEmpty(t, user.PasswordHash)

	tests.DB.Model(&user).
		Update("rank", rank)
}

func login(t *testing.T, router *gin.Engine, username string, password string) *http.Cookie {
	form := url.Values{
		"username": {username},
		"password": {password},
	}

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(form.Encode()),
	)
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	require.Equal(t, http.StatusOK, loginRec.Code)

	// Get the session cookie set by /login
	cookies := loginRec.Result().Cookies()
	require.NotEmpty(t, cookies)

	var cookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			cookie = c
			break
		}
	}
	require.NotNil(t, cookie)
	return cookie
}

func TestAccountDeleteSelf1(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foo1", "123456789", 0)

	// Login
	cookie := login(t, router, "foo1", "123456789")

	// Delete the account using the authenticated session
	deleteReq := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/profiles/%d/delete", userID(t, "foo1")),
		nil,
	)
	deleteReq.AddCookie(cookie)

	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	require.Equal(t, http.StatusNoContent, deleteRec.Code)

	var user models.User
	err := tests.DB.
		Where("username = ?", "foo1").
		First(&user).
		Error

	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// Deleting an account erases the user and everything that belongs to them
func TestAccountDeleteErasesData(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "erase1", "123456789", 0)
	cookie := login(t, router, "erase1", "123456789")

	var user models.User
	require.NoError(t, tests.DB.Where("username = ?", "erase1").First(&user).Error)
	now := time.Now()
	require.NoError(t, tests.DB.Create(&models.Notification{UserID: user.ID, Type: "test"}).Error)
	require.NoError(t, tests.DB.Create(&models.Card{CardID: 1, Acquired: now, OwnerID: user.ID}).Error)
	require.NoError(t, tests.DB.Create(&models.Booster{CollectionID: 1, Size: 1, Acquired: now, OwnerID: user.ID}).Error)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/profiles/%d/delete", userID(t, "erase1")), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
	// The session cookie is cleared
	require.Contains(t, strings.Join(rec.Header().Values("Set-Cookie"), "\n"), "session_token=;")

	count := func(model any, column string) int64 {
		var n int64
		require.NoError(t, tests.DB.Unscoped().Model(model).Where(column+" = ?", user.ID).Count(&n).Error)
		return n
	}
	require.Zero(t, count(&models.User{}, "id"), "user row is kept")
	require.Zero(t, count(&models.UserProfile{}, "user_id"), "profile is kept")
	require.Zero(t, count(&models.UserSession{}, "user_id"), "sessions are kept")
	require.Zero(t, count(&models.Notification{}, "user_id"), "notifications are kept")
	require.Zero(t, count(&models.Card{}, "owner_id"), "cards are kept")
	require.Zero(t, count(&models.Booster{}, "owner_id"), "boosters are kept")
}

func TestAccountDeleteSelf2(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "foo2", "123456789", 0)

	// Login
	cookie := login(t, router, "foo2", "123456789")

	// Delete the account using the authenticated session
	deleteReq := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/profiles/%d/delete", userID(t, "foo2")),
		strings.NewReader("username=foo2"),
	)
	deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleteReq.AddCookie(cookie)

	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	require.Equal(t, http.StatusNoContent, deleteRec.Code)

	var user models.User
	err := tests.DB.
		Where("username = ?", "foo2").
		First(&user).
		Error

	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestAccountDeleteOtherPermission(t *testing.T) {
	// User -> User: Fail
	{
		router := tests.Run(tests.Data, tests.DB)
		createAccount(t, router, "foo3", "123456789", 0)
		createAccount(t, router, "other1", "123456789", 0)

		// Login
		cookie := login(t, router, "foo3", "123456789")

		// Delete the account using the authenticated session
		deleteReq := httptest.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/profiles/%d/delete", userID(t, "other1")),
			strings.NewReader("username=other1"),
		)
		deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		deleteReq.AddCookie(cookie)

		deleteRec := httptest.NewRecorder()
		router.ServeHTTP(deleteRec, deleteReq)

		require.Equal(t, http.StatusForbidden, deleteRec.Code)
	}

	// Mod -> User: Fail
	{
		router := tests.Run(tests.Data, tests.DB)
		createAccount(t, router, "foo4", "123456789", 1)
		createAccount(t, router, "other2", "123456789", 0)

		// Login
		cookie := login(t, router, "foo4", "123456789")

		// Delete the account using the authenticated session
		deleteReq := httptest.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/profiles/%d/delete", userID(t, "other2")),
			strings.NewReader("username=other2"),
		)
		deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		deleteReq.AddCookie(cookie)

		deleteRec := httptest.NewRecorder()
		router.ServeHTTP(deleteRec, deleteReq)

		require.Equal(t, http.StatusForbidden, deleteRec.Code)

		var user models.User
		err := tests.DB.
			Where("username = ?", "other2").
			First(&user).
			Error
		require.NoError(t, err)
	}

	// Mod -> Mod: Fail
	{
		router := tests.Run(tests.Data, tests.DB)
		createAccount(t, router, "foo5", "123456789", 1)
		createAccount(t, router, "other3", "123456789", 1)

		// Login
		cookie := login(t, router, "foo5", "123456789")

		// Delete the account using the authenticated session
		deleteReq := httptest.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/profiles/%d/delete", userID(t, "other3")),
			strings.NewReader("username=other3"),
		)
		deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		deleteReq.AddCookie(cookie)

		deleteRec := httptest.NewRecorder()
		router.ServeHTTP(deleteRec, deleteReq)

		require.Equal(t, http.StatusForbidden, deleteRec.Code)
	}

	// Amin -> Mod: Ok
	{
		router := tests.Run(tests.Data, tests.DB)
		createAccount(t, router, "foo6", "123456789", 2)
		createAccount(t, router, "other4", "123456789", 1)

		// Login
		cookie := login(t, router, "foo6", "123456789")

		// Delete the account using the authenticated session
		deleteReq := httptest.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/profiles/%d/delete", userID(t, "other4")),
			strings.NewReader("username=other4"),
		)
		deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		deleteReq.AddCookie(cookie)

		deleteRec := httptest.NewRecorder()
		router.ServeHTTP(deleteRec, deleteReq)

		require.Equal(t, http.StatusNoContent, deleteRec.Code)

		var user models.User
		err := tests.DB.
			Where("username = ?", "other4").
			First(&user).
			Error
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	}

	// Amin -> Admin: Ok
	{
		router := tests.Run(tests.Data, tests.DB)
		createAccount(t, router, "foo7", "123456789", 2)
		createAccount(t, router, "other5", "123456789", 2)

		// Login
		cookie := login(t, router, "foo7", "123456789")

		// Delete the account using the authenticated session
		deleteReq := httptest.NewRequest(
			http.MethodPost,
			fmt.Sprintf("/profiles/%d/delete", userID(t, "other5")),
			strings.NewReader("username=other5"),
		)
		deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		deleteReq.AddCookie(cookie)

		deleteRec := httptest.NewRecorder()
		router.ServeHTTP(deleteRec, deleteReq)

		require.Equal(t, http.StatusNoContent, deleteRec.Code)

		var user models.User
		err := tests.DB.
			Where("username = ?", "other5").
			First(&user).
			Error
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	}
}
