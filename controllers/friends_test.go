package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"transcendance/models"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type friendUser struct {
	ID     uint
	Cookie *http.Cookie
}

func friendUsers(t *testing.T, router *gin.Engine, names ...string) []friendUser {
	users := make([]friendUser, len(names))
	for i, name := range names {
		createAccount(t, router, name, "foo-bar-baz-qux", 0)
		var user models.User
		require.NoError(t, tests.DB.Where("username = ?", name).First(&user).Error)
		users[i] = friendUser{ID: user.ID, Cookie: login(t, router, name, "foo-bar-baz-qux")}
	}
	return users
}

type friendResponse struct {
	Code     int
	Relation string
	Message  string
}

// A page as seen by a user
func getPageAs(t *testing.T, router *gin.Engine, cookie *http.Cookie, target string) string {
	rec := notificationRequest(t, router, cookie, http.MethodGet, target, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}

func friendAction(t *testing.T, router *gin.Engine, from friendUser, target uint, action string) friendResponse {
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/friendaction?target=%d&action=%s", target, action), nil)
	if from.Cookie != nil {
		req.AddCookie(from.Cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	res := friendResponse{Code: rec.Code}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	return res
}

// The user's notifications, as "type: description", newest first
func friendNotifications(t *testing.T, router *gin.Engine, user friendUser) []string {
	var list []string
	for _, notif := range getNotifications(t, router, user.Cookie, "").Notifications {
		read := ""
		if notif.Read {
			read = " (read)"
		}
		list = append(list, notif.Type+": "+notif.Desc+read)
	}
	return list
}

func TestFriendRequest(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	users := friendUsers(t, router, "fra_ann", "fra_ben", "fra_cid")
	alice, bob, carol := users[0], users[1], users[2]

	// One request per user, sent to several users at once
	require.Equal(t, friendResponse{Code: http.StatusOK, Relation: "pending"}, friendAction(t, router, alice, bob.ID, "add"))
	require.Equal(t, friendResponse{Code: http.StatusOK, Relation: "pending"}, friendAction(t, router, alice, bob.ID, "add"))
	require.Equal(t, "pending", friendAction(t, router, alice, carol.ID, "add").Relation)
	require.Equal(t, []string{"friend-request: fra_ann sent you a friend request"}, friendNotifications(t, router, bob))

	// Listed on both sides
	require.Equal(t, []string{"fra_ben", "fra_cid"}, searchUsernames(getPageAs(t, router, alice.Cookie, "/friends?list=sent")))
	require.Equal(t, []string{"fra_ann"}, searchUsernames(getPageAs(t, router, bob.Cookie, "/friends?list=received")))
	require.Empty(t, searchUsernames(getPageAs(t, router, alice.Cookie, "/friends")))

	// Accepting makes both friends and notifies the sender
	require.Equal(t, "friends", friendAction(t, router, bob, alice.ID, "accept").Relation)
	require.Equal(t, []string{"friend-request: fra_ann sent you a friend request (read)"}, friendNotifications(t, router, bob))
	require.Equal(t, []string{"friend-accepted: fra_ben accepted your friend request"}, friendNotifications(t, router, alice))
	require.Equal(t, []string{"fra_ben"}, searchUsernames(getPageAs(t, router, alice.Cookie, "/friends")))
	require.Equal(t, []string{"fra_ann"}, searchUsernames(getPageAs(t, router, bob.Cookie, "/friends")))
	require.Empty(t, searchUsernames(getPageAs(t, router, alice.Cookie, "/friends?q=zzz")))

	// Denying reads the request
	require.Equal(t, "", friendAction(t, router, carol, alice.ID, "deny").Relation)
	require.Equal(t, []string{"friend-request: fra_ann sent you a friend request (read)"}, friendNotifications(t, router, carol))

	// Cancelling removes it
	require.Equal(t, "pending", friendAction(t, router, carol, bob.ID, "add").Relation)
	require.Equal(t, "", friendAction(t, router, carol, bob.ID, "cancel").Relation)
	require.Len(t, friendNotifications(t, router, bob), 1)

	// Requests sent to each other make friends
	require.Equal(t, "pending", friendAction(t, router, carol, alice.ID, "add").Relation)
	require.Equal(t, "friends", friendAction(t, router, alice, carol.ID, "add").Relation)

	// Unfriending goes both ways
	require.Equal(t, "", friendAction(t, router, bob, alice.ID, "unfriend").Relation)
	require.Equal(t, []string{"fra_cid"}, searchUsernames(getPageAs(t, router, alice.Cookie, "/friends")))
}

func TestFriendBlock(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	users := friendUsers(t, router, "frb_ann", "frb_ben")
	alice, bob := users[0], users[1]

	require.Equal(t, "pending", friendAction(t, router, alice, bob.ID, "add").Relation)
	require.Equal(t, "blocked", friendAction(t, router, bob, alice.ID, "block").Relation)
	// The request is gone with its notification
	require.Empty(t, friendNotifications(t, router, bob))
	require.Equal(t, friendResponse{
		Code:     http.StatusConflict,
		Relation: "block",
		Message:  "Can't add this user as a friend",
	}, friendAction(t, router, alice, bob.ID, "add"))

	// Blocking back then unblocking doesn't lift the other block
	require.Equal(t, "blocked", friendAction(t, router, alice, bob.ID, "block").Relation)
	require.Equal(t, http.StatusConflict, friendAction(t, router, alice, bob.ID, "add").Code)
	require.Equal(t, "block", friendAction(t, router, alice, bob.ID, "unblock").Relation)
	require.Equal(t, "blocked", friendAction(t, router, bob, alice.ID, "add").Relation)

	require.Equal(t, "", friendAction(t, router, bob, alice.ID, "unblock").Relation)
	require.Equal(t, "pending", friendAction(t, router, alice, bob.ID, "add").Relation)
}

func TestFriendErrors(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	users := friendUsers(t, router, "fre_ann")
	alice := users[0]

	require.Equal(t, http.StatusUnauthorized, friendAction(t, router, friendUser{}, alice.ID, "add").Code)
	require.Equal(t, http.StatusNotFound, friendAction(t, router, alice, 999999, "add").Code)
	require.Equal(t, http.StatusBadRequest, friendAction(t, router, alice, alice.ID, "add").Code)
	require.Equal(t, http.StatusBadRequest, friendAction(t, router, alice, alice.ID, "hug").Code)

	rec := notificationRequest(t, router, nil, http.MethodGet, "/friends", nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
}
