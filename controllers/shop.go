package controllers

import (
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

// defaultBoosterPrice is the coin price of one booster, for collections
// without an entry in boosterPrices.
const defaultBoosterPrice = 250

// boosterPrices sets the coin price of one booster per collection ID.
// Collections without an entry fall back to defaultBoosterPrice.
var boosterPrices = map[int]uint{}

// maxBuyQuantity is the most boosters of one collection a single checkout
// sells. The buy popup advertises it so its stepper cannot exceed it.
const maxBuyQuantity = 10

// MaxBuyQuantity is maxBuyQuantity, for the pages selling boosters
const MaxBuyQuantity = maxBuyQuantity

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

// topupReturn returns where to go back after a top-up, "" when the requested
// page isn't a path of this site
func topupReturn(raw string) string {
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/\\") {
		return ""
	}
	return raw
}

// ShopNotice is the message shown after buying coins or a failed purchase, if any
type ShopNotice struct {
	Message   string
	Error     bool
	LinkHref  string
	LinkLabel string
}

// notice builds the notice shown after a shop round-trip, if any.
// GetShopNotice reads the outcome of a purchase from the query
func GetShopNotice(c *gin.Context, renderer *views.Renderer, d *data.Data) ShopNotice {
	if packID := c.Query("topup"); packID != "" {
		if amount, ok := topupAmount(packID); ok {
			return ShopNotice{Message: renderer.T(c, "shop-topup-success", "amount", amount)}
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

// errInsufficientFunds is returned when the balance does not cover the price.
var errInsufficientFunds = errors.New("insufficient funds")

// ShopBuyPost spends coins on boosters of a collection and adds them to the
// inventory. The reply is a redirect so refreshing the shop cannot buy twice.
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

	// A caller that does not ask for a bulk buy gets a single booster.
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
		// Debit only when the balance covers the total, so concurrent
		// purchases cannot overdraw.
		result := tx.Model(&models.User{}).
			Where("id = ? AND coins >= ?", user.ID, total).
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

// ShopTopupPost credits the coins of a pack to the logged in account.
// The reply is a redirect so refreshing the shop cannot credit a pack twice.
func (sc *Controller) ShopTopupPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	packID := c.PostForm("pack")
	amount, ok := topupAmount(packID)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/inventory?error=pack")
		return
	}

	if err := sc.DB.Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("coins", gorm.Expr("coins + ?", amount)).Error; err != nil {
		log.Printf("WARN failed to credit %d coins to user %d: %s", amount, user.ID, err)
		c.Redirect(http.StatusSeeOther, "/inventory?error=internal")
		return
	}

	// Pages offering the popup outside the shop get the user back to them
	if back := topupReturn(c.PostForm("return")); back != "" {
		c.Redirect(http.StatusSeeOther, back)
		return
	}
	c.Redirect(http.StatusSeeOther, "/inventory?topup="+packID)
}
