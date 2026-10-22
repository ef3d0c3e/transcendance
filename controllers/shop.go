package controllers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"

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
	PriceLabel string
	Data       map[string]any
}

// One coin pack on sale in the shop
type shopTopup struct {
	ID    string
	Label string
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

// shopMessage builds the notice shown after a checkout round-trip, if any.
func (sc *ShopController) shopMessage(c *gin.Context) (string, bool) {
	if packID := c.Query("topup"); packID != "" {
		if amount, ok := topupAmount(packID); ok {
			return sc.Renderer.T(c, "shop-topup-success", "amount", amount), false
		}
	}

	switch reason := c.Query("error"); reason {
	case "":
		return "", false
	case "auth":
		return sc.Renderer.T(c, "shop-error-unauthenticated"), true
	case "pack":
		return sc.Renderer.T(c, "shop-error-pack"), true
	default:
		return sc.Renderer.T(c, "shop-error-internal"), true
	}
}

func (sc *ShopController) ShopGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	locale := sc.Renderer.Localizer.GetLocale(c)
	boosters := make([]shopBooster, 0, len(sc.Data.Collections))
	for _, collection := range sc.Data.Collections {
		title, err := collection.Translate(locale, "title")
		if err != nil {
			title = collection.Name
		}
		boosters = append(boosters, shopBooster{
			ID:         collection.ID,
			Title:      title,
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

	message, messageError := sc.shopMessage(c)

	var coins uint
	if user != nil {
		coins = user.Coins
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": sc.Renderer.T(c, "shop-title"),
		"User":  user,
	})
	builder.Add("shop", "Content", map[string]any{
		"User":         user,
		"Message":      message,
		"MessageError": messageError,
		"Boosters":     boosters,
		"Topups":       topups,
		"Coins":        coins,
	})
	sc.Renderer.Render(c, &builder)
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
