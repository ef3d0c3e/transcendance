package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"transcendance/models"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// daysAgo returns the start of the day n days before today
func daysAgo(n int) time.Time {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	return today.AddDate(0, 0, -n)
}

// setDailyStreak gives a user a connection streak that claimed its last
// reward on the given day, with the given best streak
func setDailyStreak(t *testing.T, username string, day uint, best uint, lastDay *time.Time) {
	t.Helper()
	id := userID(t, username)
	require.NoError(t, tests.DB.Where("user_id = ?", id).Delete(&models.DailyStreak{}).Error)
	require.NoError(t, tests.DB.Create(&models.DailyStreak{
		UserID:  id,
		Day:     day,
		Best:    best,
		LastDay: lastDay,
	}).Error)
}

func dailyStreak(t *testing.T, username string) models.DailyStreak {
	t.Helper()
	var streak models.DailyStreak
	require.NoError(t, tests.DB.Where("user_id = ?", userID(t, username)).First(&streak).Error)
	return streak
}

// claimDaily claims the connection streak reward of the day from the
// inventory's streak section
func claimDaily(t *testing.T, router *gin.Engine, cookie *http.Cookie, tab string) *httptest.ResponseRecorder {
	t.Helper()
	return postForm(t, router, cookie, "/inventory/claim", url.Values{"tab": {tab}})
}

func TestDailyRewardClaim(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak1", "foo-bar-baz-qux")

	rec := claimDaily(t, router, cookie, "")
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory", rec.Header().Get("Location"))

	// The first day of the ladder pays 200 coins
	require.Equal(t, uint(200), shopBalance(t, "streak1"))
	streak := dailyStreak(t, "streak1")
	require.Equal(t, uint(1), streak.Day)
	require.Equal(t, uint(1), streak.Best)
	require.NotNil(t, streak.LastDay)
	require.WithinDuration(t, daysAgo(0), *streak.LastDay, time.Second)
}

func TestDailyRewardClaimTwice(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak2", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak2", "foo-bar-baz-qux")

	require.Equal(t, http.StatusSeeOther, claimDaily(t, router, cookie, "").Code)
	// The bonus of the day pays once only
	require.Equal(t, http.StatusSeeOther, claimDaily(t, router, cookie, "").Code)

	require.Equal(t, uint(200), shopBalance(t, "streak2"))
	require.Equal(t, uint(1), dailyStreak(t, "streak2").Day)
}

func TestDailyRewardContinuesStreak(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak3", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak3", "foo-bar-baz-qux")
	// The fourth day pays 700 coins
	yesterday := daysAgo(1)
	setDailyStreak(t, "streak3", 3, 3, &yesterday)

	claimDaily(t, router, cookie, "")

	require.Equal(t, uint(700), shopBalance(t, "streak3"))
	streak := dailyStreak(t, "streak3")
	require.Equal(t, uint(4), streak.Day)
	require.Equal(t, uint(4), streak.Best)
}

func TestDailyRewardResetsAfterMissedDay(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak4", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak4", "foo-bar-baz-qux")
	// A missed day sends the streak back to day 1, the best is kept
	twoDaysAgo := daysAgo(2)
	setDailyStreak(t, "streak4", 4, 8, &twoDaysAgo)

	claimDaily(t, router, cookie, "")

	require.Equal(t, uint(200), shopBalance(t, "streak4"))
	streak := dailyStreak(t, "streak4")
	require.Equal(t, uint(1), streak.Day)
	require.Equal(t, uint(8), streak.Best)
}

func TestDailyRewardClimbsPastTheCap(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak5", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak5", "foo-bar-baz-qux")
	// Past the last reward day the counter keeps climbing, and the last
	// reward keeps paying
	yesterday := daysAgo(1)
	setDailyStreak(t, "streak5", 5, 5, &yesterday)

	claimDaily(t, router, cookie, "")

	require.Equal(t, uint(1000), shopBalance(t, "streak5"))
	streak := dailyStreak(t, "streak5")
	require.Equal(t, uint(6), streak.Day)
	require.Equal(t, uint(6), streak.Best)
}

func TestDailyRewardSectionHoldsTheRecord(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak6", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak6", "foo-bar-baz-qux")
	yesterday := daysAgo(1)
	setDailyStreak(t, "streak6", 12, 12, &yesterday)

	body := getInventory(t, router, cookie, "/inventory")
	// Past the reward cap the raw streak shows next to the tier, and the
	// best streak is the ongoing one
	require.Contains(t, body, `<span class="daily-streak-tier">Tier 5/5</span>`)
	require.Contains(t, body, `<span class="daily-streak-count">Streak 12</span>`)
	require.Contains(t, body, `<span class="daily-streak-best">Best: current</span>`)
	require.Contains(t, body, "Claim 1,000 coins")
}

func TestDailyRewardSectionBelowTheCap(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak7", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak7", "foo-bar-baz-qux")
	yesterday := daysAgo(1)
	setDailyStreak(t, "streak7", 3, 8, &yesterday)

	body := getInventory(t, router, cookie, "/inventory")
	// The tier is the one of today's bonus: the fourth day of the ladder
	require.Contains(t, body, `<span class="daily-streak-tier">Tier 4/5</span>`)
	// Below the cap the raw streak stays hidden
	require.NotContains(t, body, "daily-streak-count")
	require.Contains(t, body, `<span class="daily-streak-best">Best: 8</span>`)
	require.Contains(t, body, "Claim 700 coins")
}

func TestDailyRewardSectionBrokenStreak(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak8", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak8", "foo-bar-baz-qux")
	// The last claim is too old for the streak to continue
	twoDaysAgo := daysAgo(2)
	setDailyStreak(t, "streak8", 12, 12, &twoDaysAgo)

	body := getInventory(t, router, cookie, "/inventory")
	// The bonus of the day restarts the streak at day 1
	require.Contains(t, body, `<span class="daily-streak-tier">Tier 1/5</span>`)
	require.NotContains(t, body, "daily-streak-count")
	require.Contains(t, body, "Claim 200 coins")
	// The ongoing streak holds no record anymore
	require.Contains(t, body, `<span class="daily-streak-best">Best: 12</span>`)
}

func TestDailyRewardSectionClaimed(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak9", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak9", "foo-bar-baz-qux")
	today := daysAgo(0)
	setDailyStreak(t, "streak9", 3, 3, &today)

	body := getInventory(t, router, cookie, "/inventory")
	require.Contains(t, body, `<span class="daily-streak-tier">Tier 3/5</span>`)
	require.Contains(t, body, `<span class="daily-streak-best">Best: current</span>`)
	// The button is disabled until midnight, and counts down to it
	require.Contains(t, body, `class="daily-streak-button" disabled`)
	require.Contains(t, body, "Available in")
	require.Contains(t, body, "data-countdown=")

	// The countdown ends at the next midnight, server time
	match := regexp.MustCompile(`data-countdown="(\d+)"`).FindStringSubmatch(body)
	require.NotNil(t, match)
	seconds, err := strconv.Atoi(match[1])
	require.NoError(t, err)
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	require.InDelta(t, time.Until(midnight).Seconds(), seconds, 30)
}

func TestDailyRewardSectionOnEveryTab(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak10", "foo-bar-baz-qux", 0)
	createAccount(t, router, "streak11", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak10", "foo-bar-baz-qux")

	// The section is on top of every tab of one's own inventory
	require.Contains(t, getInventory(t, router, cookie, "/inventory"), "daily-streak")
	require.Contains(t, getInventory(t, router, cookie, "/inventory?tab=boosters"), "daily-streak")

	// And only on one's own inventory
	require.NotContains(t, getInventory(t, router, cookie, "/inventory?u=streak11"), "daily-streak")
}

func TestDailyRewardSectionClaimsFromTheTab(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak12", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak12", "foo-bar-baz-qux")

	// The claim comes back to the tab it was made from
	rec := claimDaily(t, router, cookie, "boosters")
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=boosters", rec.Header().Get("Location"))
}

func TestDailyRewardUnauthenticated(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak13", "foo-bar-baz-qux", 0)

	rec := claimDaily(t, router, nil, "")
	require.Equal(t, http.StatusForbidden, rec.Code)

	// Only signed in visitors have a streak
	require.Equal(t, uint(0), shopBalance(t, "streak13"))
	var count int64
	require.NoError(t, tests.DB.Model(&models.DailyStreak{}).Where("user_id = ?", userID(t, "streak13")).Count(&count).Error)
	require.Zero(t, count)
}

// setBoosterPile gives a user a pile of default boosters
func setBoosterPile(t *testing.T, username string, count uint, nextAt *time.Time) {
	t.Helper()
	id := userID(t, username)
	require.NoError(t, tests.DB.Where("owner_id = ?", id).Delete(&models.BoosterPile{}).Error)
	require.NoError(t, tests.DB.Create(&models.BoosterPile{
		OwnerID: id,
		Count:   count,
		NextAt:  nextAt,
	}).Error)
}

func boosterPile(t *testing.T, username string) models.BoosterPile {
	t.Helper()
	var pile models.BoosterPile
	require.NoError(t, tests.DB.Where("owner_id = ?", userID(t, username)).First(&pile).Error)
	return pile
}

func TestBoosterPileEmpty(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "pile1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "pile1", "foo-bar-baz-qux")
	setBoosterPile(t, "pile1", 0, nil)

	body := getInventory(t, router, cookie, "/inventory?tab=boosters")
	// The pile shows its count and the countdown of its next booster
	require.Contains(t, body, `<span class="booster-pile-count">×0</span>`)
	require.Contains(t, body, `data-countdown=`)
	require.Contains(t, body, "Default booster")
	// An empty pile is de-saturated, and cannot be opened
	require.Contains(t, body, `class="card booster-pile-art booster-pile-empty"`)
	require.NotContains(t, body, `data-booster-open="/opening?booster=default"`)
}

func TestBoosterPileFull(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "pile2", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "pile2", "foo-bar-baz-qux")
	setBoosterPile(t, "pile2", 5, nil)

	body := getInventory(t, router, cookie, "/inventory?tab=boosters")
	require.Contains(t, body, `<span class="booster-pile-count">×5</span>`)
	require.Contains(t, body, `data-booster-open="/opening?booster=default"`)
	// A full pile waits for nothing: no countdown pill
	require.NotContains(t, body, "data-countdown")
	require.NotContains(t, body, "booster-pile-empty")
}

func TestBoosterPileCountedInTotals(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "pile3", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "pile3", "foo-bar-baz-qux")
	giveBoosters(t, "pile3", collectionDev, 5, 2)
	nextAt := time.Now().Add(time.Hour)
	setBoosterPile(t, "pile3", 3, &nextAt)

	body := getInventory(t, router, cookie, "/inventory?tab=boosters")
	require.Contains(t, body, `<span class="booster-pile-count">×3</span>`)
	// The pile counts as boosters, next to the collection stacks
	require.Contains(t, body, `<p class="inventory-count">5 boosters</p>`)
	require.Contains(t, body, `<span class="inventory-tab-count">5</span>`)
	require.Equal(t, 1, strings.Count(body, `<article class="collection"`))
}

func TestBoosterPileOfOtherUser(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "pile4", "foo-bar-baz-qux", 0)
	createAccount(t, router, "pile5", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "pile4", "foo-bar-baz-qux")
	nextAt := time.Now().Add(time.Hour)
	setBoosterPile(t, "pile5", 3, &nextAt)

	body := getInventory(t, router, cookie, "/inventory?u=pile5&tab=boosters")
	require.NotContains(t, body, "booster-pile")
	require.NotContains(t, body, "data-booster-open")
}

func TestBoosterPileAccruesOverTime(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "pile6", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "pile6", "foo-bar-baz-qux")
	// Two boosters were due while the pile was left alone, a third one is
	// not due yet
	past := time.Now().Add(-25 * time.Minute)
	setBoosterPile(t, "pile6", 1, &past)

	body := getInventory(t, router, cookie, "/inventory?tab=boosters")
	require.Contains(t, body, `<span class="booster-pile-count">×4</span>`)
	pile := boosterPile(t, "pile6")
	require.Equal(t, uint(4), pile.Count)
	require.NotNil(t, pile.NextAt)
	// The countdown keeps counting from the moment the pile stopped being
	// full, 5 minutes before the next booster
	require.WithinDuration(t, time.Now().Add(5*time.Minute), *pile.NextAt, time.Minute)
}

func TestBoosterPileAccrualStopsAtCap(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "pile7", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "pile7", "foo-bar-baz-qux")
	past := time.Now().Add(-time.Hour)
	setBoosterPile(t, "pile7", 4, &past)

	body := getInventory(t, router, cookie, "/inventory?tab=boosters")
	require.Contains(t, body, `<span class="booster-pile-count">×5</span>`)
	pile := boosterPile(t, "pile7")
	require.Equal(t, uint(5), pile.Count)
	// A full pile has no countdown
	require.Nil(t, pile.NextAt)
	require.NotContains(t, body, "data-countdown")
}

type defaultOpeningResponse struct {
	Reels          [][]openingSlot `json:"reels"`
	Next           string          `json:"next"`
	Remaining      int             `json:"remaining"`
	RemainingLabel string          `json:"remainingLabel"`
	Message        string          `json:"message"`
}

func openDefaultBooster(t *testing.T, router *gin.Engine, cookie *http.Cookie) (int, defaultOpeningResponse) {
	t.Helper()
	rec := postForm(t, router, cookie, "/opening", url.Values{
		"booster": {"default"},
	})

	var resp defaultOpeningResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return rec.Code, resp
}

func TestOpenDefaultBooster(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "dpile1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "dpile1", "foo-bar-baz-qux")
	nextAt := time.Now().Add(10 * time.Minute)
	setBoosterPile(t, "dpile1", 2, &nextAt)

	code, resp := openDefaultBooster(t, router, cookie)
	require.Equal(t, http.StatusOK, code)
	// A default booster drops 5 cards from all collections
	require.Len(t, resp.Reels, 5)
	require.Equal(t, "default", resp.Next)
	require.Equal(t, 1, resp.Remaining)
	require.Equal(t, "1 booster left", stripIsolation(resp.RemainingLabel))

	cards := ownedCards(t, "dpile1")
	require.Len(t, cards, 5)
	for _, card := range cards {
		drawn, _ := tests.Data.GetCard(int(card.CardID))
		require.NotNil(t, drawn)
	}

	// The last booster of the pile stops the chain
	code, resp = openDefaultBooster(t, router, cookie)
	require.Equal(t, http.StatusOK, code)
	require.Empty(t, resp.Next)
	require.Zero(t, resp.Remaining)
	require.Equal(t, "No boosters left", stripIsolation(resp.RemainingLabel))
	require.Len(t, ownedCards(t, "dpile1"), 10)
	require.Zero(t, boosterPile(t, "dpile1").Count)

	// The pile is empty now
	code, resp = openDefaultBooster(t, router, cookie)
	require.Equal(t, http.StatusNotFound, code)
	require.Equal(t, "No default booster available", stripIsolation(resp.Message))
	require.Len(t, ownedCards(t, "dpile1"), 10)
}

func TestOpenDefaultBoosterFromFullPileStartsCountdown(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "dpile2", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "dpile2", "foo-bar-baz-qux")
	setBoosterPile(t, "dpile2", 5, nil)

	code, _ := openDefaultBooster(t, router, cookie)
	require.Equal(t, http.StatusOK, code)

	// Taking the booster that filled the pile starts the countdown
	pile := boosterPile(t, "dpile2")
	require.Equal(t, uint(4), pile.Count)
	require.NotNil(t, pile.NextAt)
	require.WithinDuration(t, time.Now().Add(10*time.Minute), *pile.NextAt, time.Minute)
}

func TestOpenDefaultBoosterPage(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "dpile3", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "dpile3", "foo-bar-baz-qux")
	nextAt := time.Now().Add(10 * time.Minute)
	setBoosterPile(t, "dpile3", 1, &nextAt)

	req := httptest.NewRequest(http.MethodGet, "/opening?booster=default", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	body := shopBody(rec)
	require.Contains(t, body, `<input type="hidden" name="booster" value="default">`)
	require.Contains(t, body, "Default booster")
	require.Contains(t, body, `<span class="booster-pile-count">×1</span>`)

	// An empty pile has nothing to open: back to the inventory
	setBoosterPile(t, "dpile3", 0, &nextAt)
	req = httptest.NewRequest(http.MethodGet, "/opening?booster=default", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=boosters", rec.Header().Get("Location"))
}

func TestOpenDefaultBoosterUnauthenticated(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "dpile4", "foo-bar-baz-qux", 0)
	nextAt := time.Now().Add(10 * time.Minute)
	setBoosterPile(t, "dpile4", 1, &nextAt)

	code, _ := openDefaultBooster(t, router, nil)
	require.Equal(t, http.StatusForbidden, code)
	require.Equal(t, uint(1), boosterPile(t, "dpile4").Count)
}

func TestRegisterCreatesBoosterPile(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "pile8", "foo-bar-baz-qux", 0)

	// New accounts start with an empty pile that already refills
	pile := boosterPile(t, "pile8")
	require.Zero(t, pile.Count)
	require.NotNil(t, pile.NextAt)
	require.WithinDuration(t, time.Now().Add(10*time.Minute), *pile.NextAt, time.Minute)
}
