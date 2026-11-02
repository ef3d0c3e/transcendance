package controllers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
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

const defaultBoosterPrice = 250

// boosterPrices overrides defaultBoosterPrice per collection ID.
var boosterPrices = map[int]uint{}

// maxBuyQuantity caps the boosters of one checkout, and the stepper of the buy popup.
const maxBuyQuantity = 10

const boosterPileCap = 5

const boosterPileInterval = 10 * time.Minute

// dailyStreakRewards starts at day 1. Days past the last entry keep paying the last amount.
var dailyStreakRewards = []uint{200, 300, 500, 700, 1000}

func dailyStreakReward(day uint) uint {
	if day < 1 {
		day = 1
	}
	if int(day) > len(dailyStreakRewards) {
		return dailyStreakRewards[len(dailyStreakRewards)-1]
	}
	return dailyStreakRewards[day-1]
}

const MaxBuyQuantity = maxBuyQuantity

func boosterPrice(collectionID int) uint {
	if price, ok := boosterPrices[collectionID]; ok {
		return price
	}
	return defaultBoosterPrice
}

func BoosterPrice(collectionID int) uint {
	return boosterPrice(collectionID)
}

type topupPack struct {
	ID     string
	Amount uint
	// Displayed price in euro cents. Nothing is charged, see shop-topup-note.
	PriceCents uint
}

var topupPacks = []topupPack{
	{ID: "small", Amount: 100, PriceCents: 99},
	{ID: "medium", Amount: 500, PriceCents: 449},
	{ID: "large", Amount: 1000, PriceCents: 799},
	{ID: "xlarge", Amount: 5000, PriceCents: 2999},
	{ID: "xxlarge", Amount: 10000, PriceCents: 4999},
}

func topupAmount(id string) (uint, bool) {
	for _, pack := range topupPacks {
		if pack.ID == id {
			return pack.Amount, true
		}
	}
	return 0, false
}

type TopupChoice struct {
	ID string
	// Coins credited, lets the page gray out packs that fall short
	Amount uint
	Label  string
	Price  string
}

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

func SignedNotice(c *gin.Context, notice string) string {
	user := GetAuthenticatedUser(c)
	if user == nil {
		return notice
	}
	return notice + "&sig=" + noticeSignature(user.ID, notice)
}

func NoticeSigned(c *gin.Context, notice string) bool {
	user := GetAuthenticatedUser(c)
	return user != nil && hmac.Equal([]byte(c.Query("sig")), []byte(noticeSignature(user.ID, notice)))
}

// GetShopNotice shows successes only when the action signed them: errors claim
// nothing and need no signature.
func GetShopNotice(c *gin.Context, renderer *views.Renderer, d *data.Data) ShopNotice {
	if packID := c.Query("topup"); packID != "" && NoticeSigned(c, "topup="+packID) {
		if amount, ok := topupAmount(packID); ok {
			return ShopNotice{Message: renderer.T(c, "shop-topup-success", "amount", amount)}
		}
	}
	if sold, err := strconv.Atoi(c.Query("sold")); err == nil && sold > 0 {
		coins, err := strconv.Atoi(c.Query("coins"))
		if err == nil && coins >= 0 && NoticeSigned(c, "sold="+c.Query("sold")+"&coins="+c.Query("coins")) {
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
	case "quantity":
		return ShopNotice{Message: renderer.T(c, "shop-error-quantity", "max", maxBuyQuantity), Error: true}
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

var errInsufficientFunds = errors.New("insufficient funds")

// ShopBuyPost redirects, so refreshing the shop cannot buy twice.
func (sc *Controller) ShopBuyPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	id, err := strconv.Atoi(c.PostForm("collection"))
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/inventory?error=collection")
		return
	}
	collection := sc.Data.GetCollection(id)
	if collection == nil {
		c.Redirect(http.StatusSeeOther, "/inventory?error=collection")
		return
	}

	quantity := 1
	if raw := c.PostForm("quantity"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxBuyQuantity {
			c.Redirect(http.StatusSeeOther, "/inventory?error=quantity")
			return
		}
		quantity = n
	}

	total := boosterPrice(collection.ID) * uint(quantity)
	now := time.Now()
	boosters := make([]models.Booster, quantity)
	for i := range boosters {
		boosters[i] = models.Booster{
			CollectionID: uint(collection.ID),
			Size:         uint(collection.BoosterSize),
			Acquired:     now,
			OwnerID:      user.ID,
		}
	}

	err = sc.DB.Transaction(func(tx *gorm.DB) error {
		// Locked, so no offer can commit coins between the check and the debit
		if err := LockUsers(tx, user.ID); err != nil {
			return err
		}
		// Coins up in pending trades cannot pay for a booster
		locked, err := TradeCommittedCoins(tx, user.ID)
		if err != nil {
			return err
		}
		// Conditional debit, so concurrent purchases cannot overdraw
		result := tx.Model(&models.User{}).
			Where("id = ? AND coins >= ?", user.ID, total+locked).
			Update("coins", gorm.Expr("coins - ?", total))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errInsufficientFunds
		}
		return tx.Create(&boosters).Error
	})

	switch {
	case errors.Is(err, errInsufficientFunds):
		c.Redirect(http.StatusSeeOther, "/inventory?error=funds")
	case err != nil:
		log.Printf("WARN failed to sell %d boosters of collection %d to user %d: %s", quantity, collection.ID, user.ID, err)
		c.Redirect(http.StatusSeeOther, "/inventory?error=internal")
	default:
		// Straight to opening the first one, the page goes on with the rest
		c.Redirect(http.StatusSeeOther, "/opening?booster="+strconv.FormatUint(uint64(boosters[0].ID), 10))
	}
}

// ShopTopupPost redirects, so refreshing the shop cannot credit a pack twice.
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
		// The pages compare the balance to their prices, so leave out the coins up in
		// pending trades
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

	c.Redirect(http.StatusSeeOther, "/inventory?"+SignedNotice(c, "topup="+packID))
}
