package data

import (
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
	"transcendance/data"
	"transcendance/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Number of cards a tradeup consumes
const tradeupCost = 10

// A tradeup is refused whole when its cards can't be taken
var errTradeupMissing = errors.New("tradeup cards are missing")

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
	Cards []*data.Card
}

// tradeupOutcomes lists the collections that can pay out a contract of
// `inputRarity` cards. Each is weighted by how many of the contract's cards
// come from it, and only a collection holding at least one card of the
// rewarded rarity can win. `byCollection` maps a collection ID to its cards
// grouped by rarity, as returned by groupByRarity.
func tradeupOutcomes(byCollection map[int]map[string][]*data.Card, inputRarity string, contributions map[int]int) []tradeupOutcome {
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
// its cards of the rewarded rarity at random. Returns nil when no outcome can
// pay out.
func drawTradeup(rnd *rand.Rand, outcomes []tradeupOutcome) (*data.Card, int) {
	total := 0
	for _, outcome := range outcomes {
		total += outcome.Weight
	}
	if total == 0 {
		return nil, 0
	}

	n := rnd.N(total)
	for _, outcome := range outcomes {
		if n < outcome.Weight {
			return outcome.Cards[rnd.N(len(outcome.Cards))], outcome.CollectionID
		}
		n -= outcome.Weight
	}
	return nil, 0
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

	// Owning the cards is checked up front so the player gets a clear answer
	// rather than half a contract
	for take, count := range takes {
		var owned int64
		if err := dc.DB.Model(&models.Card{}).
			Where("owner_id = ? AND card_id = ? AND foil = ?", user.ID, take.CardID, take.Foil).
			Count(&owned).Error; err != nil {
			log.Printf("WARN failed to count cards of user '%s': %s", user.Username, err)
			fail("internal")
			return
		}
		if owned < int64(count) {
			fail("missing")
			return
		}
	}

	byCollection := make(map[int]map[string][]*data.Card, len(contributions))
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
	winner, collectionID := drawTradeup(rnd, outcomes)
	if winner == nil {
		fail("empty")
		return
	}

	// Take the contract's cards and award the reward at once. Deleting with
	// the owner check first means the same card can't be spent twice, even
	// when the request is sent twice.
	now := time.Now()
	reward := uint(collectionID*1000 + winner.ID)
	err := dc.DB.Transaction(func(tx *gorm.DB) error {
		for take, count := range takes {
			var ids []uint
			if err := tx.Model(&models.Card{}).
				Where("owner_id = ? AND card_id = ? AND foil = ?", user.ID, take.CardID, take.Foil).
				Order("id").
				Limit(count).
				Pluck("id", &ids).
				Error; err != nil {
				return err
			}
			if len(ids) != count {
				return errTradeupMissing
			}
			deleted := tx.Where("id IN ? AND owner_id = ?", ids, user.ID).Delete(&models.Card{})
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
	if errors.Is(err, errTradeupMissing) {
		fail("missing")
		return
	}
	if err != nil {
		log.Printf("WARN failed to trade up for user '%s': %s", user.Username, err)
		fail("internal")
		return
	}

	c.Redirect(http.StatusSeeOther,
		fmt.Sprintf("/inventory?tab=tradeup&result=%d", reward))
}
