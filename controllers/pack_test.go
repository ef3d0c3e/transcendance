package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"transcendance/models"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openingSlot struct {
	Rarity string `json:"rarity"`
	HTML   string `json:"html"`
	Win    bool   `json:"win"`
}

type openingResponse struct {
	Reels          [][]openingSlot `json:"reels"`
	Next           uint            `json:"next"`
	Remaining      int             `json:"remaining"`
	RemainingLabel string          `json:"remainingLabel"`
	Owned          int             `json:"owned"`
	Message        string          `json:"message"`
}

var cardTitleRe = regexp.MustCompile(`<h2 class="card-title">([^<]*)</h2>`)

func postForm(t *testing.T, router *gin.Engine, cookie *http.Cookie, target string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		req.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func openBooster(t *testing.T, router *gin.Engine, cookie *http.Cookie, boosterID uint) (int, openingResponse) {
	rec := postForm(t, router, cookie, "/opening", url.Values{
		"booster": {strconv.FormatUint(uint64(boosterID), 10)},
	})

	var resp openingResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return rec.Code, resp
}

func userID(t *testing.T, username string) uint {
	var user models.User
	require.NoError(t, tests.DB.Where("username = ?", username).First(&user).Error)
	return user.ID
}

func ownedBoosters(t *testing.T, username string) []models.Booster {
	var boosters []models.Booster
	require.NoError(t, tests.DB.Where("owner_id = ?", userID(t, username)).Order("id").Find(&boosters).Error)
	return boosters
}

func ownedCards(t *testing.T, username string) []models.Card {
	var cards []models.Card
	require.NoError(t, tests.DB.Where("owner_id = ?", userID(t, username)).Find(&cards).Error)
	return cards
}

// Titles of the cards each reel landed on
func winnerTitles(t *testing.T, resp openingResponse) []string {
	titles := make([]string, 0, len(resp.Reels))
	for _, reel := range resp.Reels {
		winners := 0
		for _, slot := range reel {
			if !slot.Win {
				continue
			}
			winners++
			match := cardTitleRe.FindStringSubmatch(slot.HTML)
			require.NotNil(t, match)
			titles = append(titles, match[1])
		}
		require.Equal(t, 1, winners)
	}
	slices.Sort(titles)
	return titles
}

// Titles of owned cards, which must all be in the dev collection
func ownedTitles(t *testing.T, cards []models.Card) []string {
	titles := make([]string, 0, len(cards))
	for _, owned := range cards {
		card, collection := tests.Data.GetCard(int(owned.CardID))
		require.NotNil(t, card)
		require.Equal(t, collectionDev, collection.ID)

		title, err := card.Translate("en", "title")
		require.NoError(t, err)
		titles = append(titles, title)
	}
	slices.Sort(titles)
	return titles
}

func TestOpenBoosterAwardsCards(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "packopen1", "foo-bar-baz-qux", 0)
	giveBoosters(t, "packopen1", collectionDev, 5, 1)
	cookie := login(t, router, "packopen1", "foo-bar-baz-qux")

	booster := ownedBoosters(t, "packopen1")[0]
	code, resp := openBooster(t, router, cookie, booster.ID)
	require.Equal(t, http.StatusOK, code)
	require.Len(t, resp.Reels, 5)
	require.Zero(t, resp.Next)

	// -1 booster, +5 cards, and they are the cards the reels landed on
	require.Empty(t, ownedBoosters(t, "packopen1"))
	cards := ownedCards(t, "packopen1")
	require.Len(t, cards, 5)
	require.Equal(t, winnerTitles(t, resp), ownedTitles(t, cards))

	// Cards are stored in reel order, each winner shows the foil its card was obtained with
	slices.SortFunc(cards, func(a, b models.Card) int { return int(a.ID) - int(b.ID) })
	for i, card := range cards {
		var winner openingSlot
		for _, slot := range resp.Reels[i] {
			if slot.Win {
				winner = slot
			}
		}
		if card.Foil == "" {
			require.NotContains(t, winner.HTML, "card-foil")
		} else {
			require.Contains(t, winner.HTML, `data-mask="/cards/assets/`+card.Foil+`.png"`)
		}
	}
}

func TestOpenBoosterUsesBoosterSize(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "packopen2", "foo-bar-baz-qux", 0)
	giveBoosters(t, "packopen2", collectionDev, 3, 1)
	cookie := login(t, router, "packopen2", "foo-bar-baz-qux")

	code, resp := openBooster(t, router, cookie, ownedBoosters(t, "packopen2")[0].ID)
	require.Equal(t, http.StatusOK, code)
	require.Len(t, resp.Reels, 3)
	require.Len(t, ownedCards(t, "packopen2"), 3)
}

func TestOpenBoosterChainsStack(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "packopen3", "foo-bar-baz-qux", 0)
	giveBoosters(t, "packopen3", collectionDev, 5, 2)
	giveBoosters(t, "packopen3", collectionDev, 3, 1)
	cookie := login(t, router, "packopen3", "foo-bar-baz-qux")

	boosters := ownedBoosters(t, "packopen3")
	code, resp := openBooster(t, router, cookie, boosters[0].ID)
	require.Equal(t, http.StatusOK, code)
	// Next booster of the same stack, not the 3-card one
	require.Equal(t, boosters[1].ID, resp.Next)
	require.Equal(t, 1, resp.Remaining)
	require.Equal(t, "1 booster left", stripIsolation(resp.RemainingLabel))

	code, resp = openBooster(t, router, cookie, resp.Next)
	require.Equal(t, http.StatusOK, code)
	require.Zero(t, resp.Next)
	require.Zero(t, resp.Remaining)
	require.Equal(t, "No boosters left", stripIsolation(resp.RemainingLabel))

	require.Len(t, ownedBoosters(t, "packopen3"), 1)
	require.Len(t, ownedCards(t, "packopen3"), 10)
}

func TestOpenBoosterOnlyOnce(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "packopen4", "foo-bar-baz-qux", 0)
	giveBoosters(t, "packopen4", collectionDev, 5, 1)
	cookie := login(t, router, "packopen4", "foo-bar-baz-qux")

	booster := ownedBoosters(t, "packopen4")[0]
	code, _ := openBooster(t, router, cookie, booster.ID)
	require.Equal(t, http.StatusOK, code)

	code, resp := openBooster(t, router, cookie, booster.ID)
	require.Equal(t, http.StatusNotFound, code)
	require.NotEmpty(t, resp.Message)
	require.Len(t, ownedCards(t, "packopen4"), 5)
}

func TestOpenBoosterOfOtherUser(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "packopen5", "foo-bar-baz-qux", 0)
	createAccount(t, router, "packopen6", "foo-bar-baz-qux", 0)
	giveBoosters(t, "packopen6", collectionDev, 5, 1)
	cookie := login(t, router, "packopen5", "foo-bar-baz-qux")

	code, _ := openBooster(t, router, cookie, ownedBoosters(t, "packopen6")[0].ID)
	require.Equal(t, http.StatusNotFound, code)
	require.Len(t, ownedBoosters(t, "packopen6"), 1)
	require.Empty(t, ownedCards(t, "packopen5"))
	require.Empty(t, ownedCards(t, "packopen6"))

	// The opening page sends back to the inventory
	req := httptest.NewRequest(http.MethodGet, "/opening?booster="+strconv.FormatUint(uint64(ownedBoosters(t, "packopen6")[0].ID), 10), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=boosters", rec.Header().Get("Location"))
}

func TestOpeningPage(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "packopen8", "foo-bar-baz-qux", 0)
	giveBoosters(t, "packopen8", collectionDev, 5, 2)
	cookie := login(t, router, "packopen8", "foo-bar-baz-qux")

	booster := ownedBoosters(t, "packopen8")[0]
	page := func() string {
		req := httptest.NewRequest(http.MethodGet, "/opening?booster="+strconv.FormatUint(uint64(booster.ID), 10), nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		return stripIsolation(rec.Body.String())
	}

	body := page()
	require.Contains(t, body, `<span id="pack-remaining">2 boosters left</span>`)
	require.Contains(t, body, `<strong id="pack-owned">0</strong> / <span id="pack-total">5</span>`)
	// The dev collection holds every rarity, so the drop table is unchanged
	require.Regexp(t, `data-rarity="common">\s*<span class="pack-odds-label">Common</span>\s*<span class="pack-odds-value">80%</span>`, body)
	require.Regexp(t, `data-rarity="rare">\s*<span class="pack-odds-label">Rare</span>\s*<span class="pack-odds-value">15%</span>`, body)
	require.Regexp(t, `data-rarity="legendary">\s*<span class="pack-odds-label">Legendary</span>\s*<span class="pack-odds-value">5%</span>`, body)
	require.Regexp(t, `data-rarity="foil">\s*<span class="pack-odds-label">Foil</span>\s*<span class="pack-odds-value">5%</span>`, body)
	require.Contains(t, body, `id="pack-background"`)

	// Rarest cards first
	chase := body[strings.Index(body, `class="pack-chase"`):]
	require.Contains(t, chase, `href="/collections/1/"`)
	require.Less(t, strings.Index(chase, "LEGENDARY"), strings.Index(chase, "RARE"))

	// Opening refreshes the progress with the distinct cards obtained
	code, resp := openBooster(t, router, cookie, booster.ID)
	require.Equal(t, http.StatusOK, code)
	distinct := map[uint]bool{}
	for _, card := range ownedCards(t, "packopen8") {
		distinct[card.CardID] = true
	}
	require.Equal(t, len(distinct), resp.Owned)
}

func TestOpenBoosterUnauthenticated(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "packopen7", "foo-bar-baz-qux", 0)
	giveBoosters(t, "packopen7", collectionDev, 5, 1)

	code, _ := openBooster(t, router, nil, ownedBoosters(t, "packopen7")[0].ID)
	require.Equal(t, http.StatusForbidden, code)
	require.Len(t, ownedBoosters(t, "packopen7"), 1)
}

func TestAddBooster(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "packadd1", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "packadd1", "foo-bar-baz-qux")

	rec := postForm(t, router, cookie, "/add_booster", url.Values{"collection": {"1"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=boosters", rec.Header().Get("Location"))

	boosters := ownedBoosters(t, "packadd1")
	require.Len(t, boosters, 1)
	require.Equal(t, uint(collectionDev), boosters[0].CollectionID)
	// Size of the collection's boosters
	require.Equal(t, uint(5), boosters[0].Size)

	rec = postForm(t, router, cookie, "/add_booster", url.Values{"collection": {"999"}})
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Len(t, ownedBoosters(t, "packadd1"), 1)
}

func TestInventoryBoosterOpenConfirm(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "packbtn1", "foo-bar-baz-qux", 0)
	createAccount(t, router, "packbtn2", "foo-bar-baz-qux", 0)
	giveBoosters(t, "packbtn1", collectionDev, 5, 2)
	giveBoosters(t, "packbtn2", collectionDev, 5, 1)
	cookie := login(t, router, "packbtn1", "foo-bar-baz-qux")

	// Clicking a booster asks to open the oldest of its stack
	oldest := ownedBoosters(t, "packbtn1")[0]
	body := getInventory(t, router, cookie, "/inventory?tab=boosters")
	require.Contains(t, body, `data-booster-open="/opening?booster=`+strconv.FormatUint(uint64(oldest.ID), 10)+`"`)
	require.Contains(t, body, `data-booster-confirm="Open booster Development?"`)
	require.Contains(t, body, `id="inventory-booster-popup"`)

	// Other users' boosters can't be opened
	other := getInventory(t, router, cookie, "/inventory?u=packbtn2&tab=boosters")
	require.Contains(t, other, `<span class="collection-stack">×1</span>`)
	require.NotContains(t, other, "data-booster-open")
	require.NotContains(t, other, `id="inventory-booster-popup"`)
}

func TestAddBoosterPageNotLocalized(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "packadd2", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "packadd2", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/add_booster?lang=fr")
	require.Contains(t, body, "<h2>Add a booster (debug)</h2>")
	require.Contains(t, body, `<button type="submit">Add booster</button>`)
}
