package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"transcendance/models"
	"transcendance/tests"

	ctrldata "transcendance/controllers/data"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Global card IDs are `collection_id * 1000 + card_id`. The `dev` collection
// is #1 and its cards are numbered as in `data/cards/*/card.toml`.
const (
	cardPythagorean = 1001
	cardScotsman    = 1002
	cardFordT       = 1004
)

func getInventory(t *testing.T, router *gin.Engine, cookie *http.Cookie, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func countOwned(t *testing.T, ownerID uint) int64 {
	var total int64
	require.NoError(t, tests.DB.
		Model(&models.Card{}).
		Where("owner_id = ?", ownerID).
		Count(&total).
		Error)
	return total
}

func addCard(t *testing.T, router *gin.Engine, cookie *http.Cookie, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestInventoryRequiresAuth(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := getInventory(t, router, nil, "/inventory")

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "You must be authenticated to access inventories")
}

func TestInventoryGlobalIDStable(t *testing.T) {
	// The global ID formula must keep resolving to the same cards, or stored
	// inventories would silently point at different cards over time.
	card, collection := tests.Data.GetCard(cardScotsman)
	require.NotNil(t, card)
	require.NotNil(t, collection)
	require.Equal(t, 2, card.ID)
	require.Equal(t, 1, collection.ID)
	require.Equal(t, "flying-scotsman", card.Name)
}

func TestInventoryShowsOwnedCards(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_show", "foo-bar-qux", 0)
	cookie := login(t, router, "inv_show", "foo-bar-qux")

	rec := addCard(t, router, cookie, "/add?id=1002")
	require.Equal(t, http.StatusOK, rec.Code)

	var user models.User
	require.NoError(t, tests.DB.
		Where("username = ?", "inv_show").
		First(&user).
		Error)

	var row models.Card
	require.NoError(t, tests.DB.
		Where("owner_id = ? AND card_id = ?", user.ID, cardScotsman).
		First(&row).
		Error)
	require.Equal(t, uint(cardScotsman), row.CardID)

	rec = getInventory(t, router, cookie, "/inventory")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Flying Scotsman")

	require.Equal(t, int64(1), countOwned(t, user.ID))
}

func TestInventoryAddRejectsUnknownCards(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_badid", "foo-bar-qux", 0)
	cookie := login(t, router, "inv_badid", "foo-bar-qux")

	rec := addCard(t, router, cookie, "/add?id=999999")
	require.Equal(t, http.StatusBadRequest, rec.Code)

	rec = addCard(t, router, cookie, "/add?id=nope")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestInventoryAcquisitionDate(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_date", "foo-bar-qux", 0)
	cookie := login(t, router, "inv_date", "foo-bar-qux")

	var user models.User
	require.NoError(t, tests.DB.
		Where("username = ?", "inv_date").
		First(&user).
		Error)

	acquired := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	require.NoError(t, tests.DB.Create(&models.Card{
		CardID:   cardFordT,
		Acquired: acquired,
		OwnerID:  user.ID,
	}).Error)

	var row models.Card
	require.NoError(t, tests.DB.
		Where("owner_id = ? AND card_id = ?", user.ID, cardFordT).
		First(&row).
		Error)

	rec := getInventory(t, router, cookie, "/inventory")
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	// The date is not printed on the card face: it rides along in data
	// attributes that feed the click popup
	require.NotContains(t, body, `class="card-acquired"`)
	require.Contains(t, body, `data-acquired="Acquired `)
	require.Contains(t, body, row.Acquired.Format("2006-01-02"))

	// The popup links to the card's collection page
	require.Contains(t, body, `data-collection-url="/collections/1"`)
	require.Contains(t, body, `data-collection-title="Development"`)
	require.Contains(t, body, `data-card-title="Ford Model T"`)
}

func TestInventoryEffectsRendered(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_fx", "foo-bar-qux", 0)
	cookie := login(t, router, "inv_fx", "foo-bar-qux")

	// A plain copy gets only the base glare — foil and watermark are not a
	// default treatment anymore
	rec := addCard(t, router, cookie, "/add?id=1002")
	require.Equal(t, http.StatusOK, rec.Code)

	rec = getInventory(t, router, cookie, "/inventory")
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "card-glare")
	require.NotContains(t, body, "card-foil")
	require.NotContains(t, body, "card-watermark")

	// A copy with special effects displays them on demand
	rec = addCard(t, router, cookie, "/add?id=1004&effects=rainbow,sparkle,foil,watermark")
	require.Equal(t, http.StatusOK, rec.Code)

	rec = getInventory(t, router, cookie, "/inventory")
	require.Equal(t, http.StatusOK, rec.Code)
	body = rec.Body.String()
	require.Contains(t, body, "card-rainbow")
	require.Contains(t, body, "card-sparkle")
	require.Contains(t, body, "card-foil")
	require.Contains(t, body, "card-watermark")
	require.Contains(t, body, "card-glare")
}

func TestInventoryEffectsStored(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_store", "foo-bar-qux", 0)
	cookie := login(t, router, "inv_store", "foo-bar-qux")

	rec := addCard(t, router, cookie, "/add?id=1002&effects=foil,sparkle")
	require.Equal(t, http.StatusOK, rec.Code)

	var user models.User
	require.NoError(t, tests.DB.
		Where("username = ?", "inv_store").
		First(&user).
		Error)

	var row models.Card
	require.NoError(t, tests.DB.
		Where("owner_id = ? AND card_id = ?", user.ID, cardScotsman).
		First(&row).
		Error)

	require.Len(t, row.Effects, 2)
	require.Equal(t, "foil", row.Effects[0].Name)
	require.Equal(t, "sparkle", row.Effects[1].Name)
}

func TestInventoryAddRejectsUnknownEffects(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_badfx", "foo-bar-qux", 0)
	cookie := login(t, router, "inv_badfx", "foo-bar-qux")

	rec := addCard(t, router, cookie, "/add?id=1002&effects=foil,hack")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestInventoryEffectParams(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_param", "foo-bar-qux", 0)
	cookie := login(t, router, "inv_param", "foo-bar-qux")

	var user models.User
	require.NoError(t, tests.DB.
		Where("username = ?", "inv_param").
		First(&user).
		Error)

	// Valid parameters are rendered as data attributes
	require.NoError(t, tests.DB.Create(&models.Card{
		CardID:   cardScotsman,
		Acquired: time.Now(),
		Effects: models.CardEffects{
			{Name: "foil", Params: map[string]string{"hue": "90deg"}},
		},
		OwnerID: user.ID,
	}).Error)

	// ...and only those: unknown keys and unsafe values are dropped at render
	// time even if they somehow made it into the database
	require.NoError(t, tests.DB.Create(&models.Card{
		CardID:   cardFordT,
		Acquired: time.Now(),
		Effects: models.CardEffects{
			{Name: "sparkle", Params: map[string]string{
				"onclick":     "alert(1)",
				"hue":         `" onmouseover="alert(2)`,
				"data-static": "1",
			}},
		},
		OwnerID: user.ID,
	}).Error)

	rec := getInventory(t, router, cookie, "/inventory")
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, `data-hue="90deg"`)
	require.NotContains(t, body, "onclick")
	require.NotContains(t, body, "onmouseover")
}

func TestValidateEffects(t *testing.T) {
	require.NoError(t, ctrldata.ValidateEffects(models.CardEffects{
		{Name: "foil", Params: map[string]string{"hue": "90deg"}},
		{Name: "watermark"},
	}))

	require.Error(t, ctrldata.ValidateEffects(models.CardEffects{
		{Name: "hack"},
	}))
	// Glare is the base treatment of every card, not a stored special effect
	require.Error(t, ctrldata.ValidateEffects(models.CardEffects{
		{Name: "glare"},
	}))
	require.Error(t, ctrldata.ValidateEffects(models.CardEffects{
		{Name: "foil", Params: map[string]string{"onclick": "alert(1)"}},
	}))
	require.Error(t, ctrldata.ValidateEffects(models.CardEffects{
		{Name: "foil", Params: map[string]string{"hue": `"><script>`}},
	}))
	require.Error(t, ctrldata.ValidateEffects(models.CardEffects{
		{Name: "foil", Params: map[string]string{"hue": strings.Repeat("a", 300)}},
	}))
}

func TestInventoryPagination(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_page", "foo-bar-qux", 0)
	cookie := login(t, router, "inv_page", "foo-bar-qux")

	var user models.User
	require.NoError(t, tests.DB.
		Where("username = ?", "inv_page").
		First(&user).
		Error)

	// 25 copies over two pages of 20
	acquired := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for i := range 25 {
		require.NoError(t, tests.DB.Create(&models.Card{
			CardID:   cardScotsman,
			Acquired: acquired.Add(time.Duration(i) * time.Minute),
			OwnerID:  user.ID,
		}).Error)
	}

	page1 := getInventory(t, router, cookie, "/inventory")
	require.Equal(t, http.StatusOK, page1.Code)
	require.Equal(t, 20, strings.Count(page1.Body.String(), `class="card-container"`))

	page2 := getInventory(t, router, cookie, "/inventory?p=2")
	require.Equal(t, http.StatusOK, page2.Code)
	require.Equal(t, 5, strings.Count(page2.Body.String(), `class="card-container"`))

	// Ordering is deterministic across requests
	again := getInventory(t, router, cookie, "/inventory")
	require.Equal(t, page1.Body.String(), again.Body.String())
}

func TestInventoryOtherUser(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_owner", "foo-bar-qux", 0)
	createAccount(t, router, "inv_guest", "foo-bar-qux", 0)
	ownerCookie := login(t, router, "inv_owner", "foo-bar-qux")
	guestCookie := login(t, router, "inv_guest", "foo-bar-qux")

	require.Equal(t, http.StatusOK, addCard(t, router, ownerCookie, "/add?id=1002").Code)

	// Enough copies for the inventory to span several pages, so the paginator
	// actually links to page 2
	var owner models.User
	require.NoError(t, tests.DB.
		Where("username = ?", "inv_owner").
		First(&owner).
		Error)
	for range 21 {
		require.NoError(t, tests.DB.Create(&models.Card{
			CardID:   cardScotsman,
			Acquired: time.Now(),
			OwnerID:  owner.ID,
		}).Error)
	}

	rec := getInventory(t, router, guestCookie, "/inventory?u=inv_owner")
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "Flying Scotsman")
	require.Contains(t, body, "inv_owner")

	// Paging another user's inventory keeps looking at that user
	// (`&amp;` is the HTML-escaped `&` of "?u=inv_owner&p=")
	require.Contains(t, body, "?u=inv_owner&amp;p=")

	// Our own inventory stays empty
	rec = getInventory(t, router, guestCookie, "/inventory")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Nothing to show here")
}

func TestInventoryUnknownUser(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_nouser", "foo-bar-qux", 0)
	cookie := login(t, router, "inv_nouser", "foo-bar-qux")

	rec := getInventory(t, router, cookie, "/inventory?u=does-not-exist")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "User not found")
}

func TestInventoryStaleCardRow(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "inv_stale", "foo-bar-qux", 0)
	cookie := login(t, router, "inv_stale", "foo-bar-qux")

	var user models.User
	require.NoError(t, tests.DB.
		Where("username = ?", "inv_stale").
		First(&user).
		Error)

	// A row pointing at a card that no longer exists in the data files must not
	// take the whole inventory page down.
	require.NoError(t, tests.DB.Create(&models.Card{
		CardID:   999999,
		Acquired: time.Now(),
		OwnerID:  user.ID,
	}).Error)
	require.NoError(t, tests.DB.Create(&models.Card{
		CardID:   cardScotsman,
		Acquired: time.Now(),
		OwnerID:  user.ID,
	}).Error)

	rec := getInventory(t, router, cookie, "/inventory")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Flying Scotsman")
}
