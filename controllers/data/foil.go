package data

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"transcendance/controllers"
	"transcendance/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Foil roulette odds of one rarity, in basis points (1/100 of a percent)
type foilOdds struct {
	// Chance of a card of this rarity to come out foil on its own
	Base int `json:"base"`
	// Chance a sacrificed card of this rarity adds
	Bonus int `json:"bonus"`
}

// Foil roulette odds of each rarity
var foilRouletteOdds = map[string]foilOdds{
	"common":    {Base: 100, Bonus: 50},
	"rare":      {Base: 200, Bonus: 100},
	"legendary": {Base: 500, Bonus: 200},
}

// Highest chance a roll can reach, sacrifices or not, in basis points
const foilRouletteCap = 5000

// Rolls out of which the chance is drawn: 100% in basis points
const foilRouletteScale = 10000

// The most cards a roll can sacrifice. Plain commons on a common reach the
// cap with 98, anything past that is sent by hand.
const foilRouletteMaxSacrifices = 100

var (
	// A roll is refused whole when its cards can't be taken
	errFoilMissing = errors.New("foil roulette cards are missing")
	// Copies up in pending trades leave with those trades
	errFoilLocked = errors.New("foil roulette cards are up in trades")
)

// foilRouletteChance returns the chance, in basis points, for a card of
// `rarity` to come out foil when the cards of `sacrificed` rarities are given
// up for it, capped at foilRouletteCap. False when a rarity has no odds.
func foilRouletteChance(rarity string, sacrificed []string) (int, bool) {
	odds, ok := foilRouletteOdds[rarity]
	if !ok {
		return 0, false
	}
	chance := odds.Base
	for _, r := range sacrificed {
		odds, ok := foilRouletteOdds[r]
		if !ok {
			return 0, false
		}
		chance += odds.Bonus
	}
	return min(chance, foilRouletteCap), true
}

// rollFoilRoulette draws a roll at `chance` basis points, the foil pattern
// won or "" when the roll is lost
func rollFoilRoulette(rnd *rand.Rand, chance int) string {
	if rnd.N(foilRouletteScale) >= chance {
		return ""
	}
	return foilPatterns[rnd.N(len(foilPatterns))].Name
}

// One (card, foil) stack a roll takes copies of
type foilTake struct {
	CardID uint
	Foil   string
}

// Parse a `<id>` or `<id>:<foil>` form value
func parseFoilTake(value string) (foilTake, bool) {
	idPart, foil, _ := strings.Cut(value, ":")
	id, err := strconv.ParseUint(idPart, 10, 32)
	if err != nil || id == 0 {
		return foilTake{}, false
	}
	return foilTake{CardID: uint(id), Foil: foil}, true
}

// /foil POST: roll a plain card for a foil, sacrificing cards for better
// odds. Won, the card turns foil with a random pattern; lost, it is
// destroyed. The sacrifices are spent either way.
func (dc *DataController) FoilRoulettePost(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	fail := func(code string) {
		c.Redirect(http.StatusSeeOther, "/inventory?tab=foil&error="+code)
	}

	// The rolled card is a plain copy: a foil one has nothing to win
	target, ok := parseFoilTake(c.PostForm("target"))
	if !ok {
		fail("missing")
		return
	}
	if target.Foil != "" {
		fail("foil")
		return
	}
	targetCard, targetCollection := dc.Data.GetCard(int(target.CardID))
	if targetCard == nil || targetCollection == nil {
		fail("missing")
		return
	}

	// One `card` field per sacrificed copy
	values := c.PostFormArray("card")
	if len(values) > foilRouletteMaxSacrifices {
		fail("count")
		return
	}
	// The rolled copy is taken along with the sacrifices
	takes := map[foilTake]int{target: 1}
	sacrificed := make([]string, 0, len(values))
	for _, value := range values {
		take, ok := parseFoilTake(value)
		if !ok {
			fail("missing")
			return
		}
		card, collection := dc.Data.GetCard(int(take.CardID))
		if card == nil || collection == nil {
			fail("missing")
			return
		}
		takes[take]++
		sacrificed = append(sacrificed, card.Rarity)
	}
	chance, ok := foilRouletteChance(targetCard.Rarity, sacrificed)
	if !ok {
		fail("rarity")
		return
	}

	rnd := rand.New(rand.NewPCG(
		uint64(time.Now().UnixNano()),
		uint64(time.Now().UnixNano()),
	))
	won := rollFoilRoulette(rnd, chance)

	// Take the cards and settle the roll at once. Deleting with the owner
	// check means the same card can't be spent twice, even when the request
	// is sent twice.
	err := dc.DB.Transaction(func(tx *gorm.DB) error {
		// Locked, so no offer can put up the cards while they are spent
		if err := controllers.LockUsers(tx, user.ID); err != nil {
			return err
		}
		committed, err := controllers.TradeCommitmentsOf(tx, user.ID, 0)
		if err != nil {
			return err
		}
		for take, count := range takes {
			var ids []uint
			if err := tx.Model(&models.Card{}).
				Where("owner_id = ? AND card_id = ? AND foil = ?", user.ID, take.CardID, take.Foil).
				Order("acquired DESC, id DESC").
				Pluck("id", &ids).
				Error; err != nil {
				return err
			}
			if len(ids) < count {
				return errFoilMissing
			}
			if len(ids)-int(committed.Copies(take.CardID, take.Foil)) < count {
				return errFoilLocked
			}

			// The last acquired copies go first, the oldest stays. The
			// rolled copy is the first of its stack, and survives a win.
			spent := ids[:count]
			if take == target && won != "" {
				updated := tx.Model(&models.Card{}).
					Where("id = ? AND owner_id = ?", spent[0], user.ID).
					Update("foil", won)
				if updated.Error != nil {
					return updated.Error
				}
				if updated.RowsAffected != 1 {
					return errFoilMissing
				}
				spent = spent[1:]
			}
			if len(spent) == 0 {
				continue
			}
			deleted := tx.Where("id IN ? AND owner_id = ?", spent, user.ID).Delete(&models.Card{})
			if deleted.Error != nil {
				return deleted.Error
			}
			if deleted.RowsAffected != int64(len(spent)) {
				return errFoilMissing
			}
		}
		return nil
	})
	switch {
	case errors.Is(err, errFoilMissing):
		fail("missing")
		return
	case errors.Is(err, errFoilLocked):
		fail("locked")
		return
	case err != nil:
		log.Printf("WARN failed to roll a foil for user '%s': %s", user.Username, err)
		fail("internal")
		return
	}

	params := url.Values{
		"tab":    {inventoryTabFoil},
		"result": {strconv.FormatUint(uint64(target.CardID), 10)},
	}
	if won != "" {
		params.Set("foil", won)
	} else {
		params.Set("lost", "1")
	}
	c.Redirect(http.StatusSeeOther, "/inventory?"+params.Encode())
}

// Message for a foil roulette error code, "" when the code is unknown
func (dc *DataController) foilErrorMessage(c *gin.Context, code string) string {
	switch code {
	case "count":
		return dc.Renderer.T(c, "foil-error-count", "count", foilRouletteMaxSacrifices)
	case "foil", "rarity", "missing", "locked", "internal":
		return dc.Renderer.T(c, "foil-error-"+code)
	}
	return ""
}

// Sorts the foil roulette grid offers, applied in the page. The grid comes in
// collection order.
var foilSorts = []string{
	catalogSortNumber,
	inventorySortName,
	inventorySortNameDesc,
	inventorySortRarity,
	inventorySortCount,
}

// Fill the view with every card that can be rolled or sacrificed, and the
// outcome of the last roll if one just happened. Copies up in pending trades
// cannot be spent, so they are left out.
func (dc *DataController) inventoryFoil(c *gin.Context, userID uint, v *inventoryView) error {
	loc := dc.Renderer.Localizer.GetLocale(c)

	if message := dc.foilErrorMessage(c, c.Query("error")); message != "" {
		v.Message = message
	}

	var all []cardStack
	if err := dc.DB.Model(&models.Card{}).
		Where("owner_id = ?", userID).
		Select("card_id, foil, COUNT(*) AS count, MIN(acquired) AS first_acquired, MAX(acquired) AS last_acquired").
		Group("card_id, foil").
		Scan(&all).Error; err != nil {
		return err
	}
	committed, err := controllers.TradeCommitmentsOf(dc.DB, userID, 0)
	if err != nil {
		return err
	}

	stacks := all[:0]
	for _, stack := range all {
		stack.card, stack.collection = dc.Data.GetCard(int(stack.CardID))
		if stack.card == nil || stack.collection == nil {
			log.Printf("WARN user %d owns unknown card %d", userID, stack.CardID)
			continue
		}
		if _, ok := foilRouletteOdds[stack.card.Rarity]; !ok {
			continue
		}
		stack.Count -= min(stack.Count, int64(committed.Copies(stack.CardID, stack.Foil)))
		if stack.Count == 0 {
			continue
		}
		stacks = append(stacks, stack)
		v.Count += stack.Count
	}
	// Collection order, plain copies before the foils
	slices.SortFunc(stacks, func(a, b cardStack) int {
		return cmp.Or(cmp.Compare(a.CardID, b.CardID), cmp.Compare(a.Foil, b.Foil))
	})

	// Every stack is listed at once: a roll is built across stacks, and paging
	// away would drop the selection
	v.PageItems = int64(len(stacks))
	v.Items = make([]map[string]any, 0, len(stacks))
	v.Collections = []controllers.SelectOption{{
		Label: dc.Renderer.T(c, "inventory-filter-collection-all"),
	}}
	collections := make(map[int]bool)
	for _, stack := range stacks {
		data, err := GetCardData(c, loc, dc.Renderer, stack.collection, stack.card)
		if err != nil {
			return fmt.Errorf("card %d: %w", stack.CardID, err)
		}
		data["CardCopies"] = stack.Count
		setCardFoil(data, stack.Foil)
		v.Items = append(v.Items, map[string]any{
			"Data":   data,
			"CardID": stack.CardID,
			"Foil":   stack.Foil,
			"Count":  stack.Count,
			"Rarity": stack.card.Rarity,
		})

		if collections[stack.collection.ID] {
			continue
		}
		collections[stack.collection.ID] = true
		title, err := stack.collection.Translate(loc, "title")
		if err != nil {
			log.Printf("WARN collection %d has no title: %s", stack.collection.ID, err)
		}
		v.Collections = append(v.Collections, controllers.SelectOption{
			Value: strconv.Itoa(stack.collection.ID),
			Label: title,
		})
	}

	// The card the last roll was for: foil when won, destroyed when lost
	if id, err := strconv.Atoi(c.Query("result")); err == nil {
		card, collection := dc.Data.GetCard(id)
		foil := getFoilPattern(c.Query("foil"))
		lost := c.Query("lost") != ""
		if card != nil && collection != nil && (foil != nil || lost) {
			result, err := GetCardData(c, loc, dc.Renderer, collection, card)
			if err != nil {
				return fmt.Errorf("result card %d: %w", id, err)
			}
			if foil != nil {
				setCardFoil(result, foil.Name)
			}
			v.Result = map[string]any{"Data": result}
			v.ResultLost = foil == nil
		}
	}

	return nil
}

// Fill the foil roulette fields of the inventory page: its rarity and sort
// filters, and the odds the page counts the chance with
func (dc *DataController) foilFields(c *gin.Context, v *inventoryView, fields map[string]any) {
	fields["FoilCollections"] = v.Collections
	fields["FoilRarityOptions"] = controllers.SelectOptions(append([]string{""}, inventoryRarities...), "", func(value string) string {
		if value == "" {
			return dc.Renderer.T(c, "inventory-filter-rarity-all")
		}
		return dc.Renderer.T(c, "card-rarity-"+value)
	})
	fields["FoilSortOptions"] = controllers.SelectOptions(foilSorts, catalogSortNumber, func(value string) string {
		if value == catalogSortNumber {
			return dc.Renderer.T(c, "catalog-sort-number")
		}
		return dc.Renderer.T(c, "inventory-sort-"+value)
	})
	odds, err := json.Marshal(foilRouletteOdds)
	if err != nil {
		log.Printf("WARN failed to encode the foil roulette odds: %s", err)
	}
	fields["FoilOdds"] = string(odds)
	// Lowest rarity first, as the filter lists them
	type rarityName struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	names := make([]rarityName, 0, len(inventoryRarities))
	for _, rarity := range inventoryRarities {
		names = append(names, rarityName{Value: rarity, Label: dc.Renderer.T(c, "card-rarity-"+rarity)})
	}
	rarities, err := json.Marshal(names)
	if err != nil {
		log.Printf("WARN failed to encode the rarity names: %s", err)
	}
	fields["FoilRarities"] = string(rarities)
	// Filled in by the page, which knows the card and the chance
	fields["FoilConfirmText"] = dc.Renderer.T(c, "foil-confirm-text", "title", "{title}", "chance", "{chance}")
	fields["FoilConfirmSacrifices"] = dc.Renderer.T(c, "foil-confirm-sacrifices", "count", "{count}")
	fields["FoilRules"] = dc.Renderer.T(c, "foil-rules", "cap", foilRouletteCap/100)
	fields["FoilCap"] = foilRouletteCap
	fields["FoilScale"] = foilRouletteScale
	fields["FoilMaxSacrifices"] = foilRouletteMaxSacrifices
}
