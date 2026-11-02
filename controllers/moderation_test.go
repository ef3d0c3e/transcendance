package controllers_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"transcendance/models"
	"transcendance/tests"

	"github.com/stretchr/testify/require"
)

func setRank(t *testing.T, username string, rank uint) {
	require.NoError(t, tests.DB.Model(&models.User{}).Where("username = ?", username).Update("rank", rank).Error)
}

// Moderators delete shoutbox messages for everyone, others cannot
func TestModeratorDeletesMessages(t *testing.T) {
	router, server := newShoutboxServer(t)
	cookies, conns := chatUsers(t, router, server, "moddel1", "moddel2")
	setRank(t, "moddel1", models.RankModerator)

	shoutboxSend(t, conns[1], "spam spam spam")
	message := chatReadUntil(t, conns[0], "message").Message
	require.NotNil(t, message)

	// A regular user cannot delete it
	shoutboxWrite(t, conns[1], fmt.Sprintf(`{"type":"delete","id":%d}`, message.ID))
	require.Equal(t, "forbidden", chatReadUntil(t, conns[1], "error").Error)

	shoutboxWrite(t, conns[0], fmt.Sprintf(`{"type":"delete","id":%d}`, message.ID))
	require.Equal(t, message.ID, chatReadUntil(t, conns[0], "deleted").ID)
	require.Equal(t, message.ID, chatReadUntil(t, conns[1], "deleted").ID)

	// Gone from the history too
	for _, old := range chatHistory(t, server, cookies[1]).Messages {
		require.NotEqual(t, message.ID, old.ID)
	}

	// The client knows it can moderate
	require.True(t, chatHistory(t, server, cookies[0]).Moderator)
	require.False(t, chatHistory(t, server, cookies[1]).Moderator)

	shoutboxWrite(t, conns[0], fmt.Sprintf(`{"type":"delete","id":%d}`, message.ID))
	require.Equal(t, "message-not-found", chatReadUntil(t, conns[0], "error").Error)
}

// Moderators mute users in the shoutbox for a while, but not their peers nor admins
func TestModeratorMutes(t *testing.T) {
	router, server := newShoutboxServer(t)
	_, conns := chatUsers(t, router, server, "modmute1", "modmute2", "modmute3")
	setRank(t, "modmute1", models.RankModerator)
	setRank(t, "modmute3", models.RankAdmin)

	chatRequest(t, conns[0], map[string]string{"type": "mute", "username": "modmute2", "duration": "hour"})
	chatReadUntil(t, conns[0], "ack")

	shoutboxSend(t, conns[1], "can you hear me?")
	event := chatReadUntil(t, conns[1], "error")
	require.Equal(t, "muted", event.Error)
	require.Positive(t, event.RetryAfter)

	// Private messages still work
	chatSendPrivate(t, conns[1], "modmute1", "sorry")
	chatReadUntil(t, conns[1], "ack")

	chatRequest(t, conns[0], map[string]string{"type": "unmute", "username": "modmute2"})
	chatReadUntil(t, conns[0], "ack")
	shoutboxSend(t, conns[1], "back again")
	chatReadUntil(t, conns[1], "ack")

	chatRequest(t, conns[0], map[string]string{"type": "mute", "username": "modmute3", "duration": "day"})
	require.Equal(t, "forbidden", chatReadUntil(t, conns[0], "error").Error)

	// Regular users cannot mute anyone
	chatRequest(t, conns[1], map[string]string{"type": "mute", "username": "modmute1", "duration": "week"})
	require.Equal(t, "forbidden", chatReadUntil(t, conns[1], "error").Error)
}

// Blocking a user from their profile also stops their private messages and hides their shoutbox messages
func TestProfileBlockAppliesToChat(t *testing.T) {
	router, server := newShoutboxServer(t)
	cookies, conns := chatUsers(t, router, server, "chatblock1", "chatblock2")

	shoutboxSend(t, conns[1], "hello from chatblock2")
	chatReadUntil(t, conns[1], "ack")

	rec := postForm(t, router, cookies[0], fmt.Sprintf("/friendaction?action=block&target=%d", userID(t, "chatblock2")), url.Values{})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	chatSendPrivate(t, conns[1], "chatblock1", "are you there?")
	require.Equal(t, "blocked", chatReadUntil(t, conns[1], "error").Error)
	// Either way
	chatSendPrivate(t, conns[0], "chatblock2", "go away")
	require.Equal(t, "blocked", chatReadUntil(t, conns[0], "error").Error)

	for _, message := range chatHistory(t, server, cookies[0]).Messages {
		require.NotEqual(t, "chatblock2", message.Username)
	}
}

// Renaming applies to connections that were open before
func TestChatUsesCurrentUsername(t *testing.T) {
	router, server := newShoutboxServer(t)
	cookies, conns := chatUsers(t, router, server, "chatname1", "chatname2")

	rec := postForm(t, router, cookies[0], "/profile", url.Values{"username": {"chatname1b"}})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	shoutboxSend(t, conns[0], "new name, same me")
	message := chatReadUntil(t, conns[1], "message").Message
	require.NotNil(t, message)
	require.Equal(t, "chatname1b", message.Username)
}

// Admins manage accounts from one list, other roles cannot see it
func TestAdminUsersPage(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "adminlist1", "foo-bar-baz-qux", 2)
	createAccount(t, router, "adminlist2", "foo-bar-baz-qux", 0)
	createAccount(t, router, "adminlist3", "foo-bar-baz-qux", 1)
	admin := login(t, router, "adminlist1", "foo-bar-baz-qux")

	// Guests are sent to the login page, users and moderators are refused
	rec := notificationRequest(t, router, nil, http.MethodGet, "/admin/users", nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/login", rec.Header().Get("Location"))
	for _, username := range []string{"adminlist2", "adminlist3"} {
		rec := notificationRequest(t, router, login(t, router, username, "foo-bar-baz-qux"), http.MethodGet, "/admin/users", nil)
		require.Equal(t, http.StatusForbidden, rec.Code, username)
		require.NotContains(t, rec.Body.String(), "adminlist2@example.test")
	}

	body := getPageAs(t, router, admin, "/admin/users?q=adminlist")
	require.Contains(t, body, "Manage accounts")
	require.Contains(t, body, "adminlist2@example.test")
	require.Contains(t, body, `action="/admin/users/`+fmt.Sprint(userID(t, "adminlist2"))+`/rank"`)
	// The admin's own account is not managed from here
	require.NotContains(t, body, `action="/admin/users/`+fmt.Sprint(userID(t, "adminlist1"))+`/rank"`)

	body = getPageAs(t, router, admin, "/admin/users?q=adminlist&role=moderator")
	require.Contains(t, body, "adminlist3")
	require.NotContains(t, body, "adminlist2@example.test")

	rec = postForm(t, router, admin, fmt.Sprintf("/admin/users/%d/rank", userID(t, "adminlist2")),
		url.Values{"rank": {"1"}, "redirect": {"/admin/users?q=adminlist&p=1"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/admin/users?q=adminlist&p=1&done=rank", rec.Header().Get("Location"))
	require.Equal(t, models.RankModerator, rankOf(t, "adminlist2"))

	rec = postForm(t, router, admin, fmt.Sprintf("/admin/users/%d/rank", userID(t, "adminlist1")), url.Values{"rank": {"0"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "error=self")
	require.Equal(t, models.RankAdmin, rankOf(t, "adminlist1"))

	// Moderators cannot change roles
	moderator := login(t, router, "adminlist3", "foo-bar-baz-qux")
	rec = postForm(t, router, moderator, fmt.Sprintf("/admin/users/%d/rank", userID(t, "adminlist2")), url.Values{"rank": {"2"}})
	require.Equal(t, http.StatusForbidden, rec.Code)

	rec = postForm(t, router, admin, fmt.Sprintf("/admin/users/%d/delete", userID(t, "adminlist2")), url.Values{})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	var count int64
	require.NoError(t, tests.DB.Model(&models.User{}).Where("username = ?", "adminlist2").Count(&count).Error)
	require.Zero(t, count)
}

// Admins give a new password to a user who lost theirs, who is logged out everywhere
func TestAdminResetsPassword(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "adminpw1", "foo-bar-baz-qux", 2)
	createAccount(t, router, "adminpw2", "foo-bar-baz-qux", 0)
	admin := login(t, router, "adminpw1", "foo-bar-baz-qux")
	old := login(t, router, "adminpw2", "foo-bar-baz-qux")
	target := fmt.Sprintf("/admin/users/%d/password", userID(t, "adminpw2"))

	rec := postForm(t, router, admin, target, url.Values{"password": {"short"}, "confirm": {"short"}})
	require.Contains(t, rec.Header().Get("Location"), "error=password")
	rec = postForm(t, router, admin, target, url.Values{"password": {"new-pass-word"}, "confirm": {"other-pass-word"}})
	require.Contains(t, rec.Header().Get("Location"), "error=password-match")

	// Users cannot reset each other's password
	rec = postForm(t, router, old, fmt.Sprintf("/admin/users/%d/password", userID(t, "adminpw1")),
		url.Values{"password": {"new-pass-word"}, "confirm": {"new-pass-word"}})
	require.Equal(t, http.StatusForbidden, rec.Code)

	rec = postForm(t, router, admin, target, url.Values{"password": {"new-pass-word"}, "confirm": {"new-pass-word"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Contains(t, rec.Header().Get("Location"), "done=password")

	var sessions int64
	require.NoError(t, tests.DB.Model(&models.UserSession{}).Where("user_id = ?", userID(t, "adminpw2")).Count(&sessions).Error)
	require.Zero(t, sessions)
	login(t, router, "adminpw2", "new-pass-word")
}
