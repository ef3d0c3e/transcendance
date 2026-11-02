package data

import (
	"cmp"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"transcendance/controllers"
	"transcendance/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

// Number of cards a tradeup consumes
const tradeupCost = 10

var (
	// A tradeup is refused whole when its cards can't be taken
	errTradeupMissing = errors.New("tradeup cards are missing")
	// Copies up in pending trades leave with those trades
	errTradeupLocked = errors.New("tradeup cards are up in trades")
)

// tierAbove returns the rarity one tier above the given one, "" for the
// highest tier and for unknown rarities. rarityTiers runs from highest to
// lowest, so the tier above is the entry before it.
func tierAbove(rarity string) string {
	i := indexOfTier(rarity)
	if i <= 0 {
		return ""
	}
	return rarityTiers[i-1]
}

// One collection a tradeup can draw its reward from
type tradeupOutcome struct {
	CollectionID int
	// How many of the contract's cards come from this collection, its share
	// of the odds
	Weight int
	// Cards of the rewarded rarity the collection holds
	Cards []poolCard
}

// tradeupOutcomes lists the collections that can pay out a contract of
// `inputRarity` cards. Each is weighted by how many of the contract's cards
// come from it, and only a collection holding at least one card of the
// rewarded rarity can win. `byCollection` maps a collection ID to its cards
// grouped by rarity, as returned by groupByRarity.
func tradeupOutcomes(byCollection map[int]map[string][]poolCard, inputRarity string, contributions map[int]int) []tradeupOutcome {
	target := tierAbove(inputRarity)
	if target == "" {
		return nil
	}

	// Sorted so a seeded draw is reproducible in tests
	ids := make([]int, 0, len(contributions))
	for id := range contributions {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	outcomes := make([]tradeupOutcome, 0, len(ids))
	for _, id := range ids {
		weight := contributions[id]
		if weight <= 0 {
			continue
		}
		cards := byCollection[id][target]
		if len(cards) == 0 {
			continue
		}
		outcomes = append(outcomes, tradeupOutcome{CollectionID: id, Weight: weight, Cards: cards})
	}
	return outcomes
}

// drawTradeup picks a contract's reward: first a collection, with odds
// proportional to how many of the contract's cards come from it, then one of
// its cards of the rewarded rarity at random. Returns false when no outcome
// can pay out.
func drawTradeup(rnd *rand.Rand, outcomes []tradeupOutcome) (poolCard, bool) {
	total := 0
	for _, outcome := range outcomes {
		total += outcome.Weight
	}
	if total == 0 {
		return poolCard{}, false
	}

	n := rnd.N(total)
	for _, outcome := range outcomes {
		if n < outcome.Weight {
			return outcome.Cards[rnd.N(len(outcome.Cards))], true
		}
		n -= outcome.Weight
	}
	return poolCard{}, false
}

// One (card, foil) stack a contract takes copies of
type tradeupTake struct {
	CardID uint
	Foil   string
}

// /tradeup POST: trade `tradeupCost` cards of one rarity for a random card of
// the tier above, then show the tradeup tab with what came out
func (dc *DataController) TradeupPost(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	fail := func(code string) {
		c.Redirect(http.StatusSeeOther, "/inventory?tab=tradeup&error="+code)
	}

	// One `card` field per card taken, `<id>` or `<id>:<foil>`
	values := c.PostFormArray("card")
	if len(values) != tradeupCost {
		fail("count")
		return
	}
	takes := make(map[tradeupTake]int, tradeupCost)
	for _, value := range values {
		idPart, foil, _ := strings.Cut(value, ":")
		id, err := strconv.ParseUint(idPart, 10, 32)
		if err != nil || id == 0 {
			fail("missing")
			return
		}
		takes[tradeupTake{CardID: uint(id), Foil: foil}]++
	}

	// Every taken stack must exist and share one rarity that has a tier above
	rarities := make(map[string]bool)
	contributions := make(map[int]int)
	for take, count := range takes {
		card, collection := dc.Data.GetCard(int(take.CardID))
		if card == nil || collection == nil {
			fail("missing")
			return
		}
		rarities[card.Rarity] = true
		contributions[collection.ID] += count
	}
	if len(rarities) != 1 {
		fail("rarity")
		return
	}
	var inputRarity string
	for rarity := range rarities {
		inputRarity = rarity
	}
	if tierAbove(inputRarity) == "" {
		fail("tier")
		return
	}

	byCollection := make(map[int]map[string][]poolCard, len(contributions))
	for id := range contributions {
		collection := dc.Data.GetCollection(id)
		if collection == nil {
			fail("missing")
			return
		}
		byCollection[id] = groupByRarity(dc.Data.Cards, collection)
	}
	outcomes := tradeupOutcomes(byCollection, inputRarity, contributions)
	if len(outcomes) == 0 {
		fail("empty")
		return
	}

	rnd := rand.New(rand.NewPCG(
		uint64(time.Now().UnixNano()),
		uint64(time.Now().UnixNano()),
	))
	winner, ok := drawTradeup(rnd, outcomes)
	if !ok {
		fail("empty")
		return
	}

	// Take the contract's cards and award the reward at once. Deleting with
	// the owner check means the same card can't be spent twice, even when the
	// request is sent twice.
	now := time.Now()
	reward := uint(winner.collection.ID*1000 + winner.card.ID)
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
				return errTradeupMissing
			}
			if len(ids)-int(committed.Copies(take.CardID, take.Foil)) < count {
				return errTradeupLocked
			}
			// The last acquired copies go first, the oldest stays
			deleted := tx.Where("id IN ? AND owner_id = ?", ids[:count], user.ID).Delete(&models.Card{})
			if deleted.Error != nil {
				return deleted.Error
			}
			if deleted.RowsAffected != int64(count) {
				return errTradeupMissing
			}
		}

		return tx.Create(&models.Card{
			CardID:   reward,
			Acquired: now,
			Foil:     rollFoil(rnd),
			OwnerID:  user.ID,
		}).Error
	})
	switch {
	case errors.Is(err, errTradeupMissing):
		fail("missing")
		return
	case errors.Is(err, errTradeupLocked):
		fail("locked")
		return
	case err != nil:
		log.Printf("WARN failed to trade up for user '%s': %s", user.Username, err)
		fail("internal")
		return
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/inventory?tab=tradeup&result=%d", reward))
}

// Highest rarity first, unknown rarities last
func rarityRank(s cardStack) int {
	if i := indexOfTier(s.card.Rarity); i >= 0 {
		return i
	}
	return len(rarityTiers)
}

// Tradeup cards stand grouped by collection, then by rarity, then by how many
// copies there are to spend
func sortTradeupStacks(stacks []cardStack, loc string) {
	titles := make(map[uint]string, len(stacks))
	for _, stack := range stacks {
		title, err := stack.collection.Translate(loc, "title")
		if err != nil {
			log.Printf("WARN collection of card %d has no title: %s", stack.CardID, err)
		}
		titles[stack.CardID] = title
	}
	collator := collate.New(language.Make(loc), collate.IgnoreCase)

	slices.SortStableFunc(stacks, func(a, b cardStack) int {
		return cmp.Or(
			collator.CompareString(titles[a.CardID], titles[b.CardID]),
			cmp.Compare(rarityRank(a), rarityRank(b)),
			cmp.Compare(b.Count, a.Count),
			b.LastAcquired.Compare(a.LastAcquired),
			cmp.Compare(a.CardID, b.CardID),
			cmp.Compare(a.Foil, b.Foil),
		)
	})
}

// Message for a tradeup error code, "" when the code is unknown
func (dc *DataController) tradeupErrorMessage(c *gin.Context, code string) string {
	switch code {
	case "count":
		return dc.Renderer.T(c, "tradeup-error-count", "count", tradeupCost)
	case "rarity", "tier", "missing", "locked", "empty", "internal":
		return dc.Renderer.T(c, "tradeup-error-"+code)
	}
	return ""
}

// Fill the view with every card that can go into a tradeup, and the outcome of
// the last one if it was just completed. Copies up in pending trades cannot
// be spent, so they are left out.
func (dc *DataController) inventoryTradeup(c *gin.Context, userID uint, v *inventoryView) error {
	loc := dc.Renderer.Localizer.GetLocale(c)

	if message := dc.tradeupErrorMessage(c, c.Query("error")); message != "" {
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
		// The highest rarity has no tier above to trade into
		if tierAbove(stack.card.Rarity) == "" {
			continue
		}
		stack.Count -= min(stack.Count, int64(committed.Copies(stack.CardID, stack.Foil)))
		if stack.Count == 0 {
			continue
		}
		stacks = append(stacks, stack)
		v.Count += stack.Count
	}
	// Grouped by collection, then rarity, then how many copies to spend
	sortTradeupStacks(stacks, loc)

	// Every eligible stack is listed at once: a contract is built across
	// stacks, and paging away would drop the selection
	v.PageItems = int64(len(stacks))
	v.Items = make([]map[string]any, 0, len(stacks))
	// The filter offers every collection the grid holds, plus all of them
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

	// The card the last tradeup awarded, with the foil it rolled
	if id, err := strconv.Atoi(c.Query("result")); err == nil {
		card, collection := dc.Data.GetCard(id)
		if card != nil && collection != nil {
			result, err := GetCardData(c, loc, dc.Renderer, collection, card)
			if err != nil {
				return fmt.Errorf("result card %d: %w", id, err)
			}
			var won models.Card
			if err := dc.DB.
				Where("owner_id = ? AND card_id = ?", userID, id).
				Order("id DESC").
				First(&won).Error; err == nil {
				setCardFoil(result, won.Foil)
			}
			v.Result = map[string]any{"Data": result}
		}
	}

	return nil
}
