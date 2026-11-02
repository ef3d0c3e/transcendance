package controllers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"transcendance/data"
	"transcendance/models"
)

// sellPrices is the coin value the shop pays for one card of a rarity.
// Rarity names match the `rarity` field of a card's data.
var sellPrices = map[string]uint{
	"common":    10,
	"rare":      50,
	"legendary": 250,
}

// foilSellMultiplier multiplies a card's value when it is foil
const foilSellMultiplier = 2

// SellPrice returns the coins the shop pays for one card of a rarity, twice as
// much for a foil. Unknown rarities are worth nothing and cannot be sold.
func SellPrice(rarity string, foil string) uint {
	price := sellPrices[rarity]
	if foil != "" {
		price *= foilSellMultiplier
	}
	return price
}

var (
	// errSellMissing is returned when the inventory holds fewer copies than
	// the sale asks for
	errSellMissing = errors.New("sold cards are missing")
	// errSellLocked is returned when the copies to sell are up in trades
	errSellLocked = errors.New("sold cards are up in trades")
	// errSellNone is returned when there is no duplicate to sell
	errSellNone = errors.New("no duplicate to sell")
)

// Redirect back to the cards of the inventory with what a sale did
func soldRedirect(c *gin.Context, count uint, coins uint) {
	c.Redirect(http.StatusSeeOther, "/inventory?tab=cards&sold="+strconv.FormatUint(uint64(count), 10)+
		"&coins="+strconv.FormatUint(uint64(coins), 10))
}

// Redirect back to the cards of the inventory with why a sale failed
func sellFailed(c *gin.Context, code string) {
	c.Redirect(http.StatusSeeOther, "/inventory?tab=cards&error=sell-"+code)
}

// Take `count` copies of a stack out of an inventory, the last acquired first
// so the oldest copy stays
func removeCopies(tx *gorm.DB, ownerID uint, cardID uint, foil string, count uint) error {
	var ids []uint
	if err := tx.Model(&models.Card{}).
		Where("owner_id = ? AND card_id = ? AND foil = ?", ownerID, cardID, foil).
		Order("acquired DESC, id DESC").
		Limit(int(count)).
		Pluck("id", &ids).Error; err != nil {
		return err
	}
	if uint(len(ids)) != count {
		return errSellMissing
	}
	deleted := tx.Where("id IN ? AND owner_id = ?", ids, ownerID).Delete(&models.Card{})
	if deleted.Error != nil {
		return deleted.Error
	}
	if deleted.RowsAffected != int64(count) {
		return errSellMissing
	}
	return nil
}

// Credit the coins of a sale
func creditSale(tx *gorm.DB, ownerID uint, coins uint) error {
	return tx.Model(&models.User{}).
		Where("id = ?", ownerID).
		Update("coins", gorm.Expr("coins + ?", coins)).Error
}

// ShopSellPost takes copies of one card out of the inventory and pays their
// value in coins. The reply is a redirect back to the inventory, so
// refreshing it cannot sell the cards twice.
func (sc *Controller) ShopSellPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// The card to sell and which of its foil variants, then how many copies
	id, err := strconv.Atoi(c.PostForm("card"))
	if err != nil || id <= 0 {
		sellFailed(c, "card")
		return
	}
	foil := c.PostForm("foil")
	card, _ := sc.Data.GetCard(id)
	if card == nil || SellPrice(card.Rarity, foil) == 0 {
		sellFailed(c, "card")
		return
	}
	quantity, err := strconv.Atoi(c.PostForm("quantity"))
	if err != nil || quantity < 1 {
		sellFailed(c, "quantity")
		return
	}

	// The unit price is fixed whatever the quantity
	total := SellPrice(card.Rarity, foil) * uint(quantity)
	err = sc.DB.Transaction(func(tx *gorm.DB) error {
		// Locked, so no offer can put up the copies while they are sold
		if err := LockUsers(tx, user.ID); err != nil {
			return err
		}
		var owned int64
		if err := tx.Model(&models.Card{}).
			Where("owner_id = ? AND card_id = ? AND foil = ?", user.ID, id, foil).
			Count(&owned).Error; err != nil {
			return err
		}
		if owned < int64(quantity) {
			return errSellMissing
		}
		// Copies up in pending trades leave with those trades
		committed, err := TradeCommitmentsOf(tx, user.ID, 0)
		if err != nil {
			return err
		}
		if owned-int64(committed.Copies(uint(id), foil)) < int64(quantity) {
			return errSellLocked
		}
		if err := removeCopies(tx, user.ID, uint(id), foil, uint(quantity)); err != nil {
			return err
		}
		return creditSale(tx, user.ID, total)
	})

	switch {
	case errors.Is(err, errSellMissing):
		sellFailed(c, "missing")
	case errors.Is(err, errSellLocked):
		sellFailed(c, "locked")
	case err != nil:
		log.Printf("WARN failed to sell %d copies of card %d for user '%s': %s", quantity, id, user.Username, err)
		sellFailed(c, "internal")
	default:
		soldRedirect(c, uint(quantity), total)
	}
}

// One stack of identical copies whose duplicates can be sold
type duplicateStack struct {
	CardID uint
	Foil   string
	// Copies sold, all but one of those not up in trades
	Count uint
}

// DuplicateSale is what selling the duplicates of an inventory takes and pays
type DuplicateSale struct {
	// Copies sold
	Count uint
	// Coins paid for them
	Coins  uint
	stacks []duplicateStack
}

// DuplicatesForSale lists the duplicates of an inventory: every copy of a
// stack (same card and foil) but one. Copies up in pending trades are kept,
// and the copy left is not one of them since they leave with those trades.
func DuplicatesForSale(db *gorm.DB, d *data.Data, userID uint) (DuplicateSale, error) {
	var sale DuplicateSale
	var rows []struct {
		CardID uint
		Foil   string
		Count  uint
	}
	if err := db.Model(&models.Card{}).
		Select("card_id, foil, COUNT(*) AS count").
		Where("owner_id = ?", userID).
		Group("card_id, foil").
		Having("COUNT(*) > 1").
		Order("card_id, foil").
		Scan(&rows).Error; err != nil {
		return sale, err
	}
	if len(rows) == 0 {
		return sale, nil
	}
	committed, err := TradeCommitmentsOf(db, userID, 0)
	if err != nil {
		return sale, err
	}

	for _, row := range rows {
		card, _ := d.GetCard(int(row.CardID))
		if card == nil {
			continue
		}
		price := SellPrice(card.Rarity, row.Foil)
		if price == 0 {
			continue
		}
		free := row.Count - min(row.Count, committed.Copies(row.CardID, row.Foil))
		if free <= 1 {
			continue
		}
		count := free - 1
		sale.stacks = append(sale.stacks, duplicateStack{CardID: row.CardID, Foil: row.Foil, Count: count})
		sale.Count += count
		sale.Coins += price * count
	}
	return sale, nil
}

// ShopSellDuplicatesPost sells every duplicate card of the inventory, keeping
// one copy of each stack
func (sc *Controller) ShopSellDuplicatesPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	var sale DuplicateSale
	err := sc.DB.Transaction(func(tx *gorm.DB) error {
		// Locked, so the duplicates cannot change while they are sold
		if err := LockUsers(tx, user.ID); err != nil {
			return err
		}
		var err error
		if sale, err = DuplicatesForSale(tx, sc.Data, user.ID); err != nil {
			return err
		}
		if sale.Count == 0 {
			return errSellNone
		}
		for _, stack := range sale.stacks {
			if err := removeCopies(tx, user.ID, stack.CardID, stack.Foil, stack.Count); err != nil {
				return err
			}
		}
		return creditSale(tx, user.ID, sale.Coins)
	})

	switch {
	case errors.Is(err, errSellNone):
		sellFailed(c, "none")
	case err != nil:
		log.Printf("WARN failed to sell the duplicates of user '%s': %s", user.Username, err)
		sellFailed(c, "internal")
	default:
		soldRedirect(c, sale.Count, sale.Coins)
	}
}
