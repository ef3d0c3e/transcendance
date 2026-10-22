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

type notificationsResponse struct {
	Locale        string `json:"locale"`
	Notifications []struct {
		ID    uint
		Type  string
		Read  bool
		Title string
		Desc  string
	} `json:"notifications"`
	Unread int64 `json:"unread"`
	Total  int64 `json:"total"`
	Pages  int64 `json:"pages"`
}

func notificationRequest(t *testing.T, router *gin.Engine, cookie *http.Cookie, method string, target string, form url.Values) *httptest.ResponseRecorder {
	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, body)
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

// Fluent wraps arguments in unicode isolation marks
func stripIsolation(s string) string {
	return strings.NewReplacer("\u2068", "", "\u2069", "").Replace(s)
}

func getNotifications(t *testing.T, router *gin.Engine, cookie *http.Cookie, query string) notificationsResponse {
	rec := notificationRequest(t, router, cookie, http.MethodGet, "/api/notifications"+query, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	var res notificationsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	for i := range res.Notifications {
		res.Notifications[i].Desc = stripIsolation(res.Notifications[i].Desc)
	}
	return res
}

func addFriendRequest(t *testing.T, router *gin.Engine, cookie *http.Cookie, username string) {
	rec := notificationRequest(t, router, cookie, http.MethodPost, "/add_notification", url.Values{
		"type":     {models.NotificationFriendRequest},
		"username": {username},
	})
	require.Equal(t, http.StatusSeeOther, rec.Code)
}

func TestNotificationsUnauthenticated(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := notificationRequest(t, router, nil, http.MethodGet, "/api/notifications", nil)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = notificationRequest(t, router, nil, http.MethodPost, "/api/notifications/read_all", nil)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = notificationRequest(t, router, nil, http.MethodGet, "/notifications", nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/login", rec.Header().Get("Location"))

	rec = notificationRequest(t, router, nil, http.MethodPost, "/add_notification", url.Values{"type": {models.NotificationInfo}})
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestNotificationsList(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "notif1", "123456789", 0)
	cookie := login(t, router, "notif1", "123456789")

	for i := range 12 {
		addFriendRequest(t, router, cookie, fmt.Sprintf("sender%d", i))
	}

	// Count is the real total, not the page size
	res := getNotifications(t, router, cookie, "?limit=5")
	require.Len(t, res.Notifications, 5)
	require.Equal(t, int64(12), res.Unread)
	require.Equal(t, int64(12), res.Total)
	require.Equal(t, int64(3), res.Pages)

	// Newest first, translated
	require.Equal(t, "Friend Request", res.Notifications[0].Title)
	require.Equal(t, "sender11 sent you a friend request", res.Notifications[0].Desc)

	res = getNotifications(t, router, cookie, "?limit=5&page=3")
	require.Len(t, res.Notifications, 2)
	require.Equal(t, "sender1 sent you a friend request", res.Notifications[0].Desc)

	res = getNotifications(t, router, cookie, "?lang=fr")
	require.Equal(t, "fr", res.Locale)
	require.Equal(t, "Demande en Ami", res.Notifications[0].Title)
}

func TestNotificationsRead(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "notif2", "123456789", 0)
	createAccount(t, router, "notif3", "123456789", 0)
	cookie := login(t, router, "notif2", "123456789")
	other := login(t, router, "notif3", "123456789")

	addFriendRequest(t, router, cookie, "alice")
	addFriendRequest(t, router, cookie, "bob")
	addFriendRequest(t, router, cookie, "carol")
	res := getNotifications(t, router, cookie, "")
	require.Len(t, res.Notifications, 3)
	carol, bob := res.Notifications[0].ID, res.Notifications[1].ID

	// Only the owner can read a notification
	rec := notificationRequest(t, router, other, http.MethodPost, fmt.Sprintf("/api/notifications/%d/read", carol), nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
	rec = notificationRequest(t, router, other, http.MethodGet, fmt.Sprintf("/notification/%d", carol), nil)
	require.Equal(t, http.StatusNotFound, rec.Code)

	rec = notificationRequest(t, router, cookie, http.MethodPost, fmt.Sprintf("/api/notifications/%d/read", carol), nil)
	require.Equal(t, http.StatusNoContent, rec.Code)

	// Opening a notification reads it and goes to its page
	rec = notificationRequest(t, router, cookie, http.MethodGet, fmt.Sprintf("/notification/%d", bob), nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/search?username=bob", rec.Header().Get("Location"))

	// Read notifications are still listed
	res = getNotifications(t, router, cookie, "")
	require.Len(t, res.Notifications, 3)
	require.Equal(t, int64(1), res.Unread)
	require.True(t, res.Notifications[0].Read)
	require.True(t, res.Notifications[1].Read)
	require.False(t, res.Notifications[2].Read)

	res = getNotifications(t, router, cookie, "?filter=unread")
	require.Len(t, res.Notifications, 1)
	require.Equal(t, "alice sent you a friend request", res.Notifications[0].Desc)

	rec = notificationRequest(t, router, cookie, http.MethodPost, "/api/notifications/read_all", nil)
	require.Equal(t, http.StatusNoContent, rec.Code)
	res = getNotifications(t, router, cookie, "")
	require.Equal(t, int64(0), res.Unread)
}

func TestNotificationsActions(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "notif4", "123456789", 0)
	cookie := login(t, router, "notif4", "123456789")

	addFriendRequest(t, router, cookie, "alice")
	addFriendRequest(t, router, cookie, "bob")
	res := getNotifications(t, router, cookie, "")
	bob, alice := res.Notifications[0].ID, res.Notifications[1].ID

	// Form posts go back to local pages only
	rec := notificationRequest(t, router, cookie, http.MethodPost, fmt.Sprintf("/api/notifications/%d/read", bob), url.Values{
		"redirect": {"/notifications?filter=unread&p=1"},
	})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/notifications?filter=unread&p=1", rec.Header().Get("Location"))
	rec = notificationRequest(t, router, cookie, http.MethodPost, fmt.Sprintf("/api/notifications/%d/read", bob), url.Values{
		"redirect": {"//example.com"},
	})
	require.Equal(t, http.StatusNoContent, rec.Code)

	rec = notificationRequest(t, router, cookie, http.MethodPost, fmt.Sprintf("/api/notifications/%d/unread", bob), nil)
	require.Equal(t, http.StatusNoContent, rec.Code)
	res = getNotifications(t, router, cookie, "")
	require.Equal(t, int64(2), res.Unread)
	require.False(t, res.Notifications[0].Read)

	rec = notificationRequest(t, router, cookie, http.MethodPost, fmt.Sprintf("/api/notifications/%d/delete", alice), nil)
	require.Equal(t, http.StatusNoContent, rec.Code)
	res = getNotifications(t, router, cookie, "")
	require.Len(t, res.Notifications, 1)
	require.Equal(t, bob, res.Notifications[0].ID)

	rec = notificationRequest(t, router, cookie, http.MethodPost, fmt.Sprintf("/api/notifications/%d/delete", alice), nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestNotificationsPage(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "notif5", "123456789", 0)
	cookie := login(t, router, "notif5", "123456789")

	for i := range 17 {
		addFriendRequest(t, router, cookie, fmt.Sprintf("pager%d", i))
	}

	rec := notificationRequest(t, router, cookie, http.MethodGet, "/notifications", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	body := stripIsolation(rec.Body.String())
	require.Contains(t, body, "pager16 sent you a friend request")
	require.NotContains(t, body, "pager1 sent you a friend request")
	require.Contains(t, body, `class="paginator"`)

	rec = notificationRequest(t, router, cookie, http.MethodGet, "/notifications?p=2", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, stripIsolation(rec.Body.String()), "pager0 sent you a friend request")

	// Past the last page
	rec = notificationRequest(t, router, cookie, http.MethodGet, "/notifications?filter=unread&p=5", nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/notifications?filter=unread&p=2", rec.Header().Get("Location"))
}
