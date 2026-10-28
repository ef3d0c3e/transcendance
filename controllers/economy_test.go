package controllers_test

import (
	"encoding/json"
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
)

// daysAgo returns the start of the day n days before today
func daysAgo(n int) time.Time {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	return today.AddDate(0, 0, -n)
}

// setDailyStreak gives a user a connection streak that claimed its last
// reward on the given day
func setDailyStreak(t *testing.T, username string, day uint, lastDay *time.Time) {
	t.Helper()
	id := userID(t, username)
	require.NoError(t, tests.DB.Where("user_id = ?", id).Delete(&models.DailyStreak{}).Error)
	require.NoError(t, tests.DB.Create(&models.DailyStreak{
		UserID:  id,
		Day:     day,
		LastDay: lastDay,
	}).Error)
}

func dailyStreak(t *testing.T, username string) models.DailyStreak {
	t.Helper()
	var streak models.DailyStreak
	require.NoError(t, tests.DB.Where("user_id = ?", userID(t, username)).First(&streak).Error)
	return streak
}

// visit makes an authenticated request, which credits the connection streak
// reward of the day when it is due
func visit(t *testing.T, router *gin.Engine, cookie *http.Cookie, target string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.AddCookie(cookie)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestDailyRewardFirstVisit(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak1", "foo-bar-baz-qux")
	// The account created no reward yet
	setDailyStreak(t, "streak1", 0, nil)

	visit(t, router, cookie, "/shop")

	require.Equal(t, uint(200), shopBalance(t, "streak1"))
	streak := dailyStreak(t, "streak1")
	require.Equal(t, uint(1), streak.Day)
	require.NotNil(t, streak.LastDay)
	require.WithinDuration(t, daysAgo(0), *streak.LastDay, time.Second)

	// The reward is announced with a notification
	res := getNotifications(t, router, cookie, "")
	require.Equal(t, "Daily reward", res.Notifications[0].Title)
	require.Equal(t, "Day 1: 200 coins added to your balance", res.Notifications[0].Desc)

	// A second visit of the day changes nothing
	visit(t, router, cookie, "/shop")
	require.Equal(t, uint(200), shopBalance(t, "streak1"))
	require.Equal(t, uint(1), dailyStreak(t, "streak1").Day)
	require.Len(t, getNotifications(t, router, cookie, "").Notifications, 1)
}

func TestDailyRewardContinuesStreak(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak2", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak2", "foo-bar-baz-qux")
	// Day 4 pays 700 coins
	yesterday := daysAgo(1)
	setDailyStreak(t, "streak2", 3, &yesterday)

	visit(t, router, cookie, "/shop")

	require.Equal(t, uint(700), shopBalance(t, "streak2"))
	require.Equal(t, uint(4), dailyStreak(t, "streak2").Day)
}

func TestDailyRewardResetsAfterMissedDay(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak3", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak3", "foo-bar-baz-qux")
	// A missed day sends the streak back to day 1
	twoDaysAgo := daysAgo(2)
	setDailyStreak(t, "streak3", 4, &twoDaysAgo)

	visit(t, router, cookie, "/shop")

	require.Equal(t, uint(200), shopBalance(t, "streak3"))
	require.Equal(t, uint(1), dailyStreak(t, "streak3").Day)
}

func TestDailyRewardCapsAtFive(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak4", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "streak4", "foo-bar-baz-qux")
	// Past the last reward day the counter stays put, but keeps paying
	yesterday := daysAgo(1)
	setDailyStreak(t, "streak4", 5, &yesterday)

	visit(t, router, cookie, "/shop")

	require.Equal(t, uint(1000), shopBalance(t, "streak4"))
	require.Equal(t, uint(5), dailyStreak(t, "streak4").Day)

	// And the day after that pays again
	setDailyStreak(t, "streak4", 5, &yesterday)
	visit(t, router, cookie, "/shop")
	require.Equal(t, uint(2000), shopBalance(t, "streak4"))
}

func TestDailyRewardUnauthenticated(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "streak5", "foo-bar-baz-qux", 0)
	setDailyStreak(t, "streak5", 0, nil)

	rec := shopRequest(t, router, http.MethodGet, "/shop", nil, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	// Only signed in visitors claim the reward of the day
	require.Equal(t, uint(0), shopBalance(t, "streak5"))
	streak := dailyStreak(t, "streak5")
	require.Zero(t, streak.Day)
	require.Nil(t, streak.LastDay)
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
