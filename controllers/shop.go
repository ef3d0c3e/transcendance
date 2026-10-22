package controllers

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"transcendance/data"
	"transcendance/models"
	"transcendance/views"
)

type ShopController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
	Data     *data.Data
}

// defaultBoosterPrice is the coin price of one booster, for collections
// without an entry in boosterPrices.
const defaultBoosterPrice = 250

// boosterPrices sets the coin price of one booster per collection ID.
// Collections without an entry fall back to defaultBoosterPrice.
var boosterPrices = map[int]uint{}

// maxBuyQuantity is the most boosters of one collection a single checkout
// sells. The buy popup advertises it so its stepper cannot exceed it.
const maxBuyQuantity = 10

// boosterPrice returns the coin price of one booster of a collection.
func boosterPrice(collectionID int) uint {
	if price, ok := boosterPrices[collectionID]; ok {
		return price
	}
	return defaultBoosterPrice
}

// topupPack is one bundle of coins sold by the shop's checkout.
type topupPack struct {
	ID     string
	Amount uint
}

// Coins sold by the demo checkout. Crediting a pack takes no real payment:
// see shop-topup-note.
var topupPacks = []topupPack{
	{ID: "small", Amount: 100},
	{ID: "medium", Amount: 500},
	{ID: "large", Amount: 1000},
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

// One booster on sale in the shop. Data feeds the "collection" partial.
type shopBooster struct {
	ID         int
	Title      string
	Price      uint
	PriceLabel string
	Data       map[string]any
}

// One coin pack on sale in the shop
type shopTopup struct {
	ID    string
	Label string
}

// collectionTitle returns the localized title of a collection, falling back
// to its directory name.
func collectionTitle(collection *data.Collection, locale string) string {
	title, err := collection.Translate(locale, "title")
	if err != nil {
		return collection.Name
	}
	return title
}

// collectionTile builds the template data consumed by the "collection" partial.
func (sc *ShopController) collectionTile(c *gin.Context, collection *data.Collection, title string) map[string]any {
	return map[string]any{
		"CollectionContainerData": template.HTMLAttr(`data-tilt`),
		"CollectionArtwork":       "/collections/" + strconv.Itoa(collection.ID) + "/thumbnail",
		"CollectionTitle":         title,
		"ColorPrimary":            template.CSS(collection.ColorPrimary),
		"ColorSecondary":          template.CSS(collection.ColorSecondary),
		"CollectionDescription":   sc.Renderer.T(c, "collection-booster", "count", collection.BoosterSize),
		"CollectionCount":         sc.Renderer.T(c, "collection-count", "count", len(collection.Cards)),
		"CollectionID":            sc.Renderer.T(c, "collection-id", "id", fmt.Sprintf("%03d", collection.ID)),
	}
}

// shopNotice is the message shown after a shop round-trip, if any.
type shopNotice struct {
	Message   string
	Error     bool
	LinkHref  string
	LinkLabel string
}

// notice builds the notice shown after a shop round-trip, if any.
func (sc *ShopController) notice(c *gin.Context) shopNotice {
	locale := sc.Renderer.Localizer.GetLocale(c)

	if bought := c.Query("bought"); bought != "" {
		id, err := strconv.Atoi(bought)
		if err == nil {
			if collection := sc.Data.GetCollection(id); collection != nil {
				// A bare bought= means one booster; a bulk checkout adds its quantity.
				count := 1
				if raw := c.Query("qty"); raw != "" {
					if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= maxBuyQuantity {
						count = n
					}
				}
				return shopNotice{
					Message: sc.Renderer.T(c, "shop-buy-success",
						"collection", collectionTitle(collection, locale),
						"count", count),
					LinkHref:  "/inventory?tab=boosters",
					LinkLabel: sc.Renderer.T(c, "shop-buy-view-inventory"),
				}
			}
		}
	}

	if packID := c.Query("topup"); packID != "" {
		if amount, ok := topupAmount(packID); ok {
			return shopNotice{Message: sc.Renderer.T(c, "shop-topup-success", "amount", amount)}
		}
	}

	switch reason := c.Query("error"); reason {
	case "":
		return shopNotice{}
	case "auth":
		return shopNotice{Message: sc.Renderer.T(c, "shop-error-unauthenticated"), Error: true}
	case "pack":
		return shopNotice{Message: sc.Renderer.T(c, "shop-error-pack"), Error: true}
	case "collection":
		return shopNotice{Message: sc.Renderer.T(c, "collection-error-not-found"), Error: true}
	case "funds":
		return shopNotice{Message: sc.Renderer.T(c, "shop-error-funds"), Error: true}
	case "quantity":
		return shopNotice{Message: sc.Renderer.T(c, "shop-error-quantity", "max", maxBuyQuantity), Error: true}
	default:
		return shopNotice{Message: sc.Renderer.T(c, "shop-error-internal"), Error: true}
	}
}

func (sc *ShopController) ShopGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	locale := sc.Renderer.Localizer.GetLocale(c)
	boosters := make([]shopBooster, 0, len(sc.Data.Collections))
	for _, collection := range sc.Data.Collections {
		title := collectionTitle(collection, locale)
		boosters = append(boosters, shopBooster{
			ID:         collection.ID,
			Title:      title,
			Price:      boosterPrice(collection.ID),
			PriceLabel: sc.Renderer.T(c, "shop-price", "amount", boosterPrice(collection.ID)),
			Data:       sc.collectionTile(c, collection, title),
		})
	}
	sort.Slice(boosters, func(i, j int) bool {
		return boosters[i].ID < boosters[j].ID
	})

	// Checkout is only offered once logged in
	topups := make([]shopTopup, 0, len(topupPacks))
	if user != nil {
		for _, pack := range topupPacks {
			topups = append(topups, shopTopup{
				ID:    pack.ID,
				Label: sc.Renderer.T(c, "shop-topup-button", "amount", pack.Amount),
			})
		}
	}

	notice := sc.notice(c)

	var coins uint
	if user != nil {
		coins = user.Coins
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": sc.Renderer.T(c, "shop-title"),
		"User":  user,
	})
	builder.Add("shop", "Content", map[string]any{
		"User":        user,
		"Notice":      notice,
		"Boosters":    boosters,
		"Topups":      topups,
		"Coins":       coins,
		"MaxQuantity": maxBuyQuantity,
	})
	sc.Renderer.Render(c, &builder)
}

// errInsufficientFunds is returned when the balance does not cover the price.
var errInsufficientFunds = errors.New("insufficient funds")

// ShopBuyPost spends coins on boosters of a collection and adds them to the
// inventory. The reply is a redirect so refreshing the shop cannot buy twice.
func (sc *ShopController) ShopBuyPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/shop?error=auth")
		return
	}

	id, err := strconv.Atoi(c.PostForm("collection"))
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/shop?error=collection")
		return
	}
	collection := sc.Data.GetCollection(id)
	if collection == nil {
		c.Redirect(http.StatusSeeOther, "/shop?error=collection")
		return
	}

	// A caller that does not ask for a bulk buy gets a single booster.
	quantity := 1
	if raw := c.PostForm("quantity"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxBuyQuantity {
			c.Redirect(http.StatusSeeOther, "/shop?error=quantity")
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
		c.Redirect(http.StatusSeeOther, "/shop?error=funds")
	case err != nil:
		log.Printf("WARN failed to sell %d boosters of collection %d to user %d: %s", quantity, collection.ID, user.ID, err)
		c.Redirect(http.StatusSeeOther, "/shop?error=internal")
	default:
		// The quantity only needs saying when it is not the default one.
		location := "/shop?bought=" + strconv.Itoa(collection.ID)
		if quantity > 1 {
			location += "&qty=" + strconv.Itoa(quantity)
		}
		c.Redirect(http.StatusSeeOther, location)
	}
}

// ShopTopupPost credits the coins of a pack to the logged in account.
// The reply is a redirect so refreshing the shop cannot credit a pack twice.
func (sc *ShopController) ShopTopupPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/shop?error=auth")
		return
	}

	packID := c.PostForm("pack")
	amount, ok := topupAmount(packID)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/shop?error=pack")
		return
	}

	if err := sc.DB.Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("coins", gorm.Expr("coins + ?", amount)).Error; err != nil {
		log.Printf("WARN failed to credit %d coins to user %d: %s", amount, user.ID, err)
		c.Redirect(http.StatusSeeOther, "/shop?error=internal")
		return
	}

	c.Redirect(http.StatusSeeOther, "/shop?topup="+packID)
}
