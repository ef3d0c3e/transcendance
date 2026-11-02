package controllers_test

import (
	"regexp"
	"testing"
	"time"

	"transcendance/models"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

var searchUsernameRe = regexp.MustCompile(`class="user-result-name" href="/profile/[^"]*">([^<]*)</a>`)

// Usernames of the displayed users, in order
func searchUsernames(body string) []string {
	var usernames []string
	for _, match := range searchUsernameRe.FindAllStringSubmatch(body, -1) {
		usernames = append(usernames, match[1])
	}
	return usernames
}

// Create accounts registered at the given times
func createSearchAccounts(t *testing.T, router *gin.Engine, accounts map[string]time.Time) {
	for username, created := range accounts {
		createAccount(t, router, username, "foo-bar-baz-qux", 0)
		require.NoError(t, tests.DB.Model(&models.User{}).
			Where("username = ?", username).
			Update("created_at", created).Error)
	}
}

func TestSearchUsersRelevance(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	now := time.Now()
	createSearchAccounts(t, router, map[string]time.Time{
		"usrbob":       now,
		"usrbobby":     now,
		"xusrbob":      now,
		"usr_b_o_b":    now,
		"usralice":     now,
		"usrbobdelete": now,
	})
	require.NoError(t, tests.DB.Where("username = ?", "usrbobdelete").Delete(&models.User{}).Error)

	body := getPage(t, router, "/search?q=usrbob")
	// Exact match, prefix, substring, then characters in order
	require.Equal(t, []string{"usrbob", "usrbobby", "xusrbob", "usr_b_o_b"}, searchUsernames(body))
	require.Contains(t, body, `<p class="inventory-count">4 users</p>`)
	require.Contains(t, body, `<option value="" selected>Relevance</option>`)
	require.Contains(t, body, `<a class="user-result-name" href="/profile/usrbob">usrbob</a>`)

	// Short searches only match substrings
	require.Equal(t, []string{"usr_b_o_b", "usralice", "usrbob", "usrbobby", "xusrbob"},
		searchUsernames(getPage(t, router, "/search?q=usr&sort=name")))
	// LIKE wildcards are matched literally
	require.Equal(t, []string{"usr_b_o_b"}, searchUsernames(getPage(t, router, "/search?q=r_b")))
	require.Empty(t, searchUsernames(getPage(t, router, "/search?q=usr%25bob")))
}

func TestSearchUsersFiltersAndSorts(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	now := time.Now()
	createSearchAccounts(t, router, map[string]time.Time{
		"flta": now.Add(-2 * time.Hour),
		"fltb": now.Add(-3 * 24 * time.Hour),
		"fltc": now.Add(-2 * 365 * 24 * time.Hour),
	})
	giveCards(t, "fltb", cardTungsten, 12)
	giveCards(t, "fltc", cardTungsten, 3)

	for target, expected := range map[string][]string{
		"/search?q=flt":                {"flta", "fltb", "fltc"},
		"/search?q=flt&joined=day":     {"flta"},
		"/search?q=flt&joined=week":    {"flta", "fltb"},
		"/search?q=flt&joined=year":    {"flta", "fltb"},
		"/search?q=flt&sort=name-desc": {"fltc", "fltb", "flta"},
		"/search?q=flt&sort=newest":    {"flta", "fltb", "fltc"},
		"/search?q=flt&sort=oldest":    {"fltc", "fltb", "flta"},
		"/search?q=flt&sort=cards":     {"fltb", "fltc", "flta"},
		// Unknown values are ignored
		"/search?q=flt&joined=century&sort=bogus": {"flta", "fltb", "fltc"},
	} {
		require.Equal(t, expected, searchUsernames(getPage(t, router, target)), target)
	}

	body := getPage(t, router, "/search?q=flt&sort=cards")
	require.Contains(t, body, `<dt>Cards:</dt>`)
	require.Contains(t, body, `<option value="cards" selected>Most cards</option>`)
	require.NotContains(t, body, `/inventory?tab=cards`)

	// A signed-in visitor's card-count link opens that user's cards tab.
	cookie := login(t, router, "flta", "foo-bar-baz-qux")
	body = getInventory(t, router, cookie, "/search?q=flt&sort=cards")
	require.Contains(t, body, `<a href="/inventory?tab=cards&amp;u=fltb">12</a>`)
	body = getInventory(t, router, cookie, "/inventory?tab=cards&u=fltb")
	require.Contains(t, body, `<a class="inventory-tab-cards" href="/inventory?tab=cards&amp;u=fltb" aria-current="page">`)
	require.Contains(t, body, `<h2 class="card-title">Tungsten</h2>`)
}

func TestSearchUsersEmpty(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	body := getPage(t, router, "/search?q=nobodyhere&joined=week")
	require.Empty(t, searchUsernames(body))
	require.Contains(t, body, "No users found")
	require.Contains(t, body, `<a href="/search">Clear search</a>`)
	require.Contains(t, body, `<li><a href="/search" aria-current="page">Users</a></li>`)
}

func TestSearchUsersWithoutQuery(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "aaanosearch", "foo-bar-baz-qux", 0)

	body := getPage(t, router, "/search")
	require.Contains(t, searchUsernames(body), "aaanosearch")
	// Relevance needs a search, names are sorted by default
	require.NotContains(t, body, `>Relevance</option>`)
	require.Contains(t, body, `<option value="" selected>Name (A-Z)</option>`)
	require.NotContains(t, body, "Clear search")
}

func TestSearchUsersPagination(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	accounts := map[string]time.Time{}
	for i := range 26 {
		accounts[string(rune('a'+i))+"pagesearch"] = time.Now()
	}
	createSearchAccounts(t, router, accounts)

	first := getPage(t, router, "/search?q=pagesearch&sort=name-desc")
	require.Len(t, searchUsernames(first), 24)
	require.Equal(t, "zpagesearch", searchUsernames(first)[0])
	require.Contains(t, first, `<p class="inventory-count">26 users</p>`)
	require.Contains(t, first, `href="?q=pagesearch&amp;sort=name-desc&amp;p=2"`)

	second := getPage(t, router, "/search?q=pagesearch&sort=name-desc&p=2")
	require.Equal(t, []string{"bpagesearch", "apagesearch"}, searchUsernames(second))
}
