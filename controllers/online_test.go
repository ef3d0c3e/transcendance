package controllers_test

import (
	"regexp"
	"testing"
	"time"

	"transcendance/models"
	"transcendance/tests"

	"github.com/stretchr/testify/require"
)

var onlineStatusRe = regexp.MustCompile(`\s+`)

func lastSeen(t *testing.T, username string) *time.Time {
	var user models.User
	require.NoError(t, tests.DB.Select("last_seen_at").Where("username = ?", username).First(&user).Error)
	return user.LastSeenAt
}

func setLastSeen(t *testing.T, username string, seen any) {
	require.NoError(t, tests.DB.Model(&models.User{}).
		Where("username = ?", username).
		UpdateColumn("last_seen_at", seen).Error)
}

// Status line of the first user status, with collapsed whitespace
func onlineStatus(body string) string {
	match := regexp.MustCompile(`(?s)<p class="user-status[^>]*>(.*?)</p>`).FindStringSubmatch(body)
	if match == nil {
		return ""
	}
	return onlineStatusRe.ReplaceAllString(regexp.MustCompile(`<[^>]*>`).ReplaceAllString(match[1], ""), " ")
}

func TestLastSeenRefresh(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "seenref", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "seenref", "foo-bar-baz-qux")
	setLastSeen(t, "seenref", nil)

	getInventory(t, router, cookie, "/shop")
	first := lastSeen(t, "seenref")
	require.NotNil(t, first)
	require.WithinDuration(t, time.Now(), *first, time.Minute)

	// Not written again until the refresh delay passed
	recent := time.Now().Add(-models.LastSeenRefresh / 2).Truncate(time.Second)
	setLastSeen(t, "seenref", recent)
	getInventory(t, router, cookie, "/shop")
	require.WithinDuration(t, recent, *lastSeen(t, "seenref"), time.Second)

	stale := time.Now().Add(-2 * models.LastSeenRefresh)
	setLastSeen(t, "seenref", stale)
	getInventory(t, router, cookie, "/shop")
	require.WithinDuration(t, time.Now(), *lastSeen(t, "seenref"), time.Minute)
}

func TestSearchUsersOnline(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	now := time.Now()
	createSearchAccounts(t, router, map[string]time.Time{
		"onla": now,
		"onlb": now,
		"onlc": now,
	})
	setLastSeen(t, "onla", now.Add(-time.Minute))
	setLastSeen(t, "onlb", now.Add(-2*models.OnlineWindow))
	setLastSeen(t, "onlc", nil)

	body := getPage(t, router, "/search?q=onl")
	require.Equal(t, []string{"onla", "onlb", "onlc"}, searchUsernames(body))
	require.Contains(t, body, `<option value="online" >Online</option>`)
	require.Equal(t, " Online ", onlineStatus(body))
	require.Contains(t, body, `<p class="user-status">`)
	require.Contains(t, body, "Last seen:")
	require.Contains(t, body, "Offline")

	body = getPage(t, router, "/search?q=onl&status=online")
	require.Equal(t, []string{"onla"}, searchUsernames(body))
	require.Contains(t, body, `<option value="online" selected>Online</option>`)

	// Unknown statuses are ignored
	require.Equal(t, []string{"onla", "onlb", "onlc"}, searchUsernames(getPage(t, router, "/search?q=onl&status=away")))
}

func TestProfileOnlineStatus(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "profonl", "foo-bar-baz-qux", 0)
	createAccount(t, router, "profview", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "profview", "foo-bar-baz-qux")

	setLastSeen(t, "profonl", time.Now())
	body := getInventory(t, router, cookie, "/profile/profonl")
	require.Contains(t, body, `<p class="user-status is-online"`)
	require.Equal(t, " Online ", onlineStatus(body))

	seen := time.Now().Add(-3 * time.Hour)
	setLastSeen(t, "profonl", seen)
	body = getInventory(t, router, cookie, "/profile/profonl")
	require.Contains(t, onlineStatus(body), "Last seen:")
	require.Contains(t, body, `<time datetime="`+seen.Format("2006-01-02T15:04:05"))

	setLastSeen(t, "profonl", nil)
	require.Equal(t, " Offline ", onlineStatus(getInventory(t, router, cookie, "/profile/profonl")))

	// Not shown on the own profile editor
	require.NotContains(t, getInventory(t, router, cookie, "/profile"), `class="user-status`)
}
