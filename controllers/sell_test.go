package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"transcendance/models"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Copies of a card with a foil pattern, unlike giveCards which gives plain ones
func giveFoilCards(t *testing.T, username string, cardID uint, foil string, count int) {
	var user models.User
	require.NoError(t, tests.DB.Where("username = ?", username).First(&user).Error)

	for range count {
		require.NoError(t, tests.DB.Create(&models.Card{
			CardID:   cardID,
			Acquired: time.Now(),
			Foil:     foil,
			OwnerID:  user.ID,
		}).Error)
	}
}

func sell(t *testing.T, router *gin.Engine, cookie *http.Cookie, form url.Values) *httptest.ResponseRecorder {
	return postForm(t, router, cookie, "/shop/sell", form)
}

// The `card`, `foil` and `quantity` fields of a sale of `count` copies
func sellForm(cardID uint, foil string, quantity int) url.Values {
	form := url.Values{
		"card":     {strconv.FormatUint(uint64(cardID), 10)},
		"quantity": {strconv.Itoa(quantity)},
	}
	if foil != "" {
		form.Set("foil", foil)
	}
	return form
}

func TestSellPaysTheFixedPriceWhateverTheQuantity(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell1", "foo-bar-baz-qux", 0)
	giveCards(t, "sell1", cardTungsten, 5)
	cookie := login(t, router, "sell1", "foo-bar-baz-qux")

	// Two common cards are worth 20 coins
	rec := sell(t, router, cookie, sellForm(cardTungsten, "", 2))
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?sold=2&coins=20", rec.Header().Get("Location"))
	require.Equal(t, uint(20), shopBalance(t, "sell1"))

	// The unit price does not change with the quantity
	rec = sell(t, router, cookie, sellForm(cardTungsten, "", 3))
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?sold=3&coins=30", rec.Header().Get("Location"))
	require.Equal(t, uint(50), shopBalance(t, "sell1"))
	require.Zero(t, len(ownedCards(t, "sell1")))
}

func TestSellFoilIsWorthDouble(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell2", "foo-bar-baz-qux", 0)
	giveCards(t, "sell2", cardTungsten, 2)
	giveFoilCards(t, "sell2", cardTungsten, "foil1", 2)
	cookie := login(t, router, "sell2", "foo-bar-baz-qux")

	rec := sell(t, router, cookie, sellForm(cardTungsten, "foil1", 2))

	// The foils are worth 20 each, only the plain copies stay behind
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?sold=2&coins=40", rec.Header().Get("Location"))
	require.Equal(t, uint(40), shopBalance(t, "sell2"))
	owned := ownedCards(t, "sell2")
	require.Equal(t, 2, len(owned))
	for _, card := range owned {
		require.Empty(t, card.Foil)
	}
}

func TestSellPricesFollowTheRarity(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell3", "foo-bar-baz-qux", 0)
	giveCards(t, "sell3", cardTungsten, 1)
	giveCards(t, "sell3", cardMcMurdo, 1)
	giveCards(t, "sell3", cardFordT, 1)
	cookie := login(t, router, "sell3", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory")

	// Common, rare and legendary, one tile each
	require.Contains(t, body, `data-sell-price="10"`)
	require.Contains(t, body, `data-sell-price="50"`)
	require.Contains(t, body, `data-sell-price="250"`)
}

func TestSellRejectsBadQuantities(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell4", "foo-bar-baz-qux", 0)
	giveCards(t, "sell4", cardTungsten, 30)
	cookie := login(t, router, "sell4", "foo-bar-baz-qux")

	for _, quantity := range []int{0, -1} {
		rec := sell(t, router, cookie, sellForm(cardTungsten, "", quantity))
		require.Equal(t, http.StatusSeeOther, rec.Code)
		require.Equal(t, "/inventory?error=sell-quantity", rec.Header().Get("Location"))
	}
	rec := sell(t, router, cookie, url.Values{"card": {"1005"}, "quantity": {"banana"}})
	require.Equal(t, "/inventory?error=sell-quantity", rec.Header().Get("Location"))

	// More copies than the inventory holds
	rec = sell(t, router, cookie, sellForm(cardTungsten, "", 31))
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?error=sell-missing", rec.Header().Get("Location"))

	// Nothing was spent
	require.Zero(t, shopBalance(t, "sell4"))
	require.Equal(t, 30, len(ownedCards(t, "sell4")))

	body := getInventory(t, router, cookie, "/inventory?error=sell-quantity")
	require.Contains(t, body, "Choose a quantity of at least 1")
	body = getInventory(t, router, cookie, "/inventory?error=sell-missing")
	require.Contains(t, body, "You do not have that many copies to sell")
}

func TestSellRejectsUnknownCards(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell5", "foo-bar-baz-qux", 0)
	giveCards(t, "sell5", cardTungsten, 1)
	cookie := login(t, router, "sell5", "foo-bar-baz-qux")

	for _, card := range []string{"9999", "0", "abc"} {
		rec := sell(t, router, cookie, url.Values{"card": {card}, "quantity": {"1"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		require.Equal(t, "/inventory?error=sell-card", rec.Header().Get("Location"))
	}
	require.Zero(t, shopBalance(t, "sell5"))
	require.Equal(t, 1, len(ownedCards(t, "sell5")))

	body := getInventory(t, router, cookie, "/inventory?error=sell-card")
	require.Contains(t, body, "This card cannot be sold")
}

func TestSellAnonymousIsSentToLogin(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	rec := sell(t, router, nil, sellForm(cardTungsten, "", 1))

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestSellPopupShowsTheValueAndOffer(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell7", "foo-bar-baz-qux", 0)
	giveCards(t, "sell7", cardTungsten, 3)
	giveFoilCards(t, "sell7", cardTungsten, "foil1", 2)
	cookie := login(t, router, "sell7", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory")

	// Each tile offers the copies of its own card and foil
	require.Contains(t, body, `data-sell-id="1005"`)
	require.Contains(t, body, `data-sell-price="10"`)
	require.Contains(t, body, `data-sell-owned="3"`)
	require.Contains(t, body, `data-sell-foil="foil1"`)
	require.Contains(t, body, `data-sell-price="20"`)
	require.Contains(t, body, `data-sell-owned="2"`)
	// The popup posts the sale to the shop, with its labels
	require.Contains(t, body, `data-sell-action="/shop/sell"`)
	require.Contains(t, body, `data-label-value="Value"`)
	require.Contains(t, body, `data-label-sell="Sell"`)
	require.Contains(t, body, `data-label-quantity="Quantity"`)
	require.Contains(t, body, `data-label-confirm="Confirm sale"`)
}

func TestSellNotOfferedOnOtherInventories(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell8", "foo-bar-baz-qux", 0)
	createAccount(t, router, "sell9", "foo-bar-baz-qux", 0)
	giveCards(t, "sell8", cardTungsten, 2)
	cookie := login(t, router, "sell9", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?u=sell8")

	// The popup still opens on the copies, but none is for sale
	require.Contains(t, body, `data-acquired-first=`)
	require.NotContains(t, body, `data-sell-id`)
	require.NotContains(t, body, `data-sell-action`)
}

func TestSellNoticeNamesWhatItDid(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "sell10", "foo-bar-baz-qux", 0)
	cookie := login(t, router, "sell10", "foo-bar-baz-qux")

	body := getInventory(t, router, cookie, "/inventory?sold=3&coins=30")
	require.Contains(t, body, "You sold 3 cards for 30 coins")

	body = getInventory(t, router, cookie, "/inventory?sold=1&coins=10")
	require.Contains(t, body, "You sold 1 card for 10 coins")
}
