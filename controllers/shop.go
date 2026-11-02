package controllers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"transcendance/data"
	"transcendance/models"
	"transcendance/views"
)

// defaultBoosterPrice is the coin price of one booster, for collections
// without an entry in boosterPrices.
const defaultBoosterPrice = 250

// boosterPrices sets the coin price of one booster per collection ID.
// Collections without an entry fall back to defaultBoosterPrice.
var boosterPrices = map[int]uint{}

// boosterPileCap is how many default boosters a pile holds at most.
const boosterPileCap = 5

// boosterPileInterval is the time between two default boosters of a pile
// that is not full.
const boosterPileInterval = 10 * time.Minute

// dailyStreakRewards is the coin reward of each day of the connection
// streak, day 1 first. Days past the last entry keep paying the last amount.
var dailyStreakRewards = []uint{200, 300, 500, 700, 1000}

// dailyStreakReward returns the coins a streak day pays.
func dailyStreakReward(day uint) uint {
	if day < 1 {
		day = 1
	}
	if int(day) > len(dailyStreakRewards) {
		return dailyStreakRewards[len(dailyStreakRewards)-1]
	}
	return dailyStreakRewards[day-1]
}

// boosterPrice returns the coin price of one booster of a collection.
func boosterPrice(collectionID int) uint {
	if price, ok := boosterPrices[collectionID]; ok {
		return price
	}
	return defaultBoosterPrice
}

// BoosterPrice returns the coin price of one booster of a collection, as sold
// by the shop.
func BoosterPrice(collectionID int) uint {
	return boosterPrice(collectionID)
}

// topupPack is one bundle of coins sold by the shop's checkout.
type topupPack struct {
	ID     string
	Amount uint
	// Displayed price in euro cents. Nothing is charged, see shop-topup-note.
	PriceCents uint
}

// Coins sold by the demo checkout. Crediting a pack takes no real payment:
// see shop-topup-note.
var topupPacks = []topupPack{
	{ID: "small", Amount: 100, PriceCents: 99},
	{ID: "medium", Amount: 500, PriceCents: 449},
	{ID: "large", Amount: 1000, PriceCents: 799},
	{ID: "xlarge", Amount: 5000, PriceCents: 2999},
	{ID: "xxlarge", Amount: 10000, PriceCents: 4999},
}

// topupAmount returns the coins a pack credits, if the pack is on sale.
func topupAmount(id string) (uint, bool) {
	for _, pack := range topupPacks {
		if pack.ID == id {
			return pack.Amount, true
		}
	}
	return 0, false
}

// One coin pack on sale in the top-up popup
type TopupChoice struct {
	ID string
	// Coins credited, lets the page gray out packs that fall short
	Amount uint
	Label  string
	Price  string
}

// TopupChoices returns the coin packs of the top-up popup, with their labels
func TopupChoices(c *gin.Context, renderer *views.Renderer) []TopupChoice {
	topups := make([]TopupChoice, 0, len(topupPacks))
	for _, pack := range topupPacks {
		// Euros and cents go as strings so they skip number grouping
		topups = append(topups, TopupChoice{
			ID:     pack.ID,
			Amount: pack.Amount,
			Label:  renderer.T(c, "shop-price", "amount", pack.Amount),
			Price: renderer.T(c, "shop-topup-price",
				"euros", strconv.Itoa(int(pack.PriceCents/100)),
				"cents", fmt.Sprintf("%02d", pack.PriceCents%100)),
		})
	}
	return topups
}

// ShopNotice is the message shown after buying coins or a failed purchase, if any
type ShopNotice struct {
	Message   string
	Error     bool
	LinkHref  string
	LinkLabel string
}

// Secret signing the success notices of redirects, new on every start: a notice link lasts until then
var noticeSecret = func() []byte {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	return secret
}()

// Signature of a success notice for a user: a link made by someone else, or for someone else, shows nothing
func noticeSignature(userID uint, notice string) string {
	mac := hmac.New(sha256.New, noticeSecret)
	_, _ = fmt.Fprintf(mac, "%d:%s", userID, notice)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16])
}

// signedNotice is the query of a success notice for the user, with its signature
func signedNotice(c *gin.Context, notice string) string {
	user := GetAuthenticatedUser(c)
	if user == nil {
		return notice
	}
	return notice + "&sig=" + noticeSignature(user.ID, notice)
}

// Whether the query holds the signature of the notice for the user, as signedNotice made it
func noticeSigned(c *gin.Context, notice string) bool {
	user := GetAuthenticatedUser(c)
	return user != nil && hmac.Equal([]byte(c.Query("sig")), []byte(noticeSignature(user.ID, notice)))
}

// GetShopNotice reads the outcome of a purchase or a sale from the query. Successes are only shown when the
// action signed them, errors claim nothing and need no signature.
func GetShopNotice(c *gin.Context, renderer *views.Renderer, d *data.Data) ShopNotice {
	if packID := c.Query("topup"); packID != "" && noticeSigned(c, "topup="+packID) {
		if amount, ok := topupAmount(packID); ok {
			return ShopNotice{Message: renderer.T(c, "shop-topup-success", "amount", amount)}
		}
	}
	if sold, err := strconv.Atoi(c.Query("sold")); err == nil && sold > 0 {
		coins, err := strconv.Atoi(c.Query("coins"))
		if err == nil && coins >= 0 && noticeSigned(c, "sold="+c.Query("sold")+"&coins="+c.Query("coins")) {
			return ShopNotice{Message: renderer.T(c, "inventory-sold", "count", sold, "amount", coins)}
		}
	}

	switch reason := c.Query("error"); reason {
	case "":
		return ShopNotice{}
	case "auth":
		return ShopNotice{Message: renderer.T(c, "shop-error-unauthenticated"), Error: true}
	case "pack":
		return ShopNotice{Message: renderer.T(c, "shop-error-pack"), Error: true}
	case "collection":
		return ShopNotice{Message: renderer.T(c, "collection-error-not-found"), Error: true}
	case "funds":
		return ShopNotice{Message: renderer.T(c, "shop-error-funds"), Error: true}
	case "opening":
		return ShopNotice{Message: renderer.T(c, "pack-opening-error-internal"), Error: true}
	case "sell-card", "sell-quantity", "sell-missing", "sell-locked", "sell-none", "sell-internal":
		code := strings.TrimPrefix(reason, "sell-")
		return ShopNotice{Message: renderer.T(c, "inventory-sell-error-"+code), Error: true}
	default:
		return ShopNotice{Message: renderer.T(c, "shop-error-internal"), Error: true}
	}
}

// ShopGet: boosters are bought from the inventory, purchase notices included
func (sc *Controller) ShopGet(c *gin.Context) {
	target := "/inventory"
	if query := c.Request.URL.RawQuery; query != "" {
		target += "?" + query
	}
	c.Redirect(http.StatusSeeOther, target)
}

// ShopTopupPost credits the coins of a pack to the logged in account.
// The reply is a redirect so refreshing the shop cannot credit a pack twice.
func (sc *Controller) ShopTopupPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// Pages topping up without leaving ask for the new balance instead
	asJSON := c.GetHeader("Accept") == "application/json"

	packID := c.PostForm("pack")
	amount, ok := topupAmount(packID)
	if !ok {
		if asJSON {
			c.JSON(http.StatusBadRequest, gin.H{"error": sc.Renderer.T(c, "shop-error-pack")})
			return
		}
		c.Redirect(http.StatusSeeOther, "/inventory?error=pack")
		return
	}

	var coins uint
	err := sc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).
			Where("id = ?", user.ID).
			Update("coins", gorm.Expr("coins + ?", amount)).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Pluck("coins", &coins).Error; err != nil {
			return err
		}
		// The pages compare the balance to their prices: coins up in pending
		// trades cannot pay, so they are left out
		locked, err := TradeCommittedCoins(tx, user.ID)
		if err != nil {
			return err
		}
		coins = FreeCoins(coins, locked)
		return nil
	})
	if err != nil {
		log.Printf("WARN failed to credit %d coins to user %d: %s", amount, user.ID, err)
		if asJSON {
			c.JSON(http.StatusInternalServerError, gin.H{"error": sc.Renderer.T(c, "shop-error-internal")})
			return
		}
		c.Redirect(http.StatusSeeOther, "/inventory?error=internal")
		return
	}

	if asJSON {
		c.JSON(http.StatusOK, gin.H{
			"coins":      coins,
			"coinsLabel": sc.Renderer.T(c, "shop-price", "amount", coins),
			"navBalance": sc.Renderer.T(c, "header-balance", "amount", coins),
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/inventory?"+signedNotice(c, "topup="+packID))
}
