package data

import (
	"math/rand/v2"
	"testing"

	"transcendance/data"
)

// One collection holding the given cards, indexed by ID as tradeupOutcomes expects
func testCollection(id int, cards ...*data.Card) map[int]map[string][]*data.Card {
	byRarity := make(map[string][]*data.Card)
	for _, card := range cards {
		byRarity[card.Rarity] = append(byRarity[card.Rarity], card)
	}
	return map[int]map[string][]*data.Card{id: byRarity}
}

func TestTierAbove(t *testing.T) {
	cases := map[string]string{
		"common":    "rare",
		"rare":      "legendary",
		"legendary": "",
		"":          "",
		"unknown":   "",
	}
	for rarity, want := range cases {
		if got := tierAbove(rarity); got != want {
			t.Errorf("tierAbove(%q) = %q, want %q", rarity, got, want)
		}
	}
}

func TestTradeupOutcomesWeightsByContribution(t *testing.T) {
	byCollection := map[int]map[string][]*data.Card{
		1: {"rare": {testCard("a-rare", "rare")}},
		2: {"rare": {testCard("b-rare", "rare")}},
	}

	outcomes := tradeupOutcomes(byCollection, "common", map[int]int{1: 7, 2: 3})

	if len(outcomes) != 2 {
		t.Fatalf("got %d outcomes, want 2", len(outcomes))
	}
	if outcomes[0].CollectionID != 1 || outcomes[0].Weight != 7 {
		t.Errorf("outcome 0 = {%d %d}, want {1 7}", outcomes[0].CollectionID, outcomes[0].Weight)
	}
	if outcomes[1].CollectionID != 2 || outcomes[1].Weight != 3 {
		t.Errorf("outcome 1 = {%d %d}, want {2 3}", outcomes[1].CollectionID, outcomes[1].Weight)
	}
}

func TestTradeupOutcomesSingleCollection(t *testing.T) {
	byCollection := testCollection(1, testCard("rare", "rare"))

	outcomes := tradeupOutcomes(byCollection, "common", map[int]int{1: 20})

	if len(outcomes) != 1 {
		t.Fatalf("got %d outcomes, want 1", len(outcomes))
	}
	if outcomes[0].Weight != 20 || len(outcomes[0].Cards) != 1 {
		t.Errorf("outcome = {%d cards:%d}, want {20 cards:1}", outcomes[0].Weight, len(outcomes[0].Cards))
	}
}

func TestTradeupOutcomesSkipsCollectionsWithoutTargetTier(t *testing.T) {
	// Only the first collection holds a rare card, so it takes the whole odds
	byCollection := map[int]map[string][]*data.Card{
		1: {"rare": {testCard("a-rare", "rare")}},
		2: {"common": {testCard("b-common", "common")}},
	}

	outcomes := tradeupOutcomes(byCollection, "common", map[int]int{1: 4, 2: 6})

	if len(outcomes) != 1 {
		t.Fatalf("got %d outcomes, want 1", len(outcomes))
	}
	if outcomes[0].CollectionID != 1 || outcomes[0].Weight != 4 {
		t.Errorf("outcome = {%d %d}, want {1 4}", outcomes[0].CollectionID, outcomes[0].Weight)
	}
}

func TestTradeupOutcomesEmptyWhenNothingCanPayOut(t *testing.T) {
	byCollection := map[int]map[string][]*data.Card{
		1: {"common": {testCard("a-common", "common")}},
		2: {"common": {testCard("b-common", "common")}},
	}

	if outcomes := tradeupOutcomes(byCollection, "common", map[int]int{1: 10, 2: 10}); len(outcomes) != 0 {
		t.Errorf("got %d outcomes, want 0", len(outcomes))
	}
}

func TestTradeupOutcomesNoneForHighestRarity(t *testing.T) {
	byCollection := testCollection(1, testCard("legendary", "legendary"))

	if outcomes := tradeupOutcomes(byCollection, "legendary", map[int]int{1: 20}); len(outcomes) != 0 {
		t.Errorf("got %d outcomes, want 0", len(outcomes))
	}
}

func TestDrawTradeupPicksTargetTierCards(t *testing.T) {
	wanted := []*data.Card{
		testCard("rare-1", "rare"),
		testCard("rare-2", "rare"),
	}
	byCollection := map[int]map[string][]*data.Card{
		1: {"rare": wanted, "common": {testCard("common", "common")}},
	}
	outcomes := tradeupOutcomes(byCollection, "common", map[int]int{1: 20})
	if len(outcomes) != 1 {
		t.Fatalf("got %d outcomes, want 1", len(outcomes))
	}

	rnd := rand.New(rand.NewPCG(1, 2))
	seen := map[string]bool{}
	for range 100 {
		card, collectionID := drawTradeup(rnd, outcomes)
		if card == nil {
			t.Fatal("drawTradeup returned no card")
		}
		if collectionID != 1 {
			t.Errorf("drew from collection %d, want 1", collectionID)
		}
		if card.Rarity != "rare" {
			t.Errorf("drew %q (%s), want a rare card", card.Name, card.Rarity)
		}
		seen[card.Name] = true
	}
	// Both cards of the tier must be reachable
	for _, card := range wanted {
		if !seen[card.Name] {
			t.Errorf("card %q was never drawn", card.Name)
		}
	}
}

func TestDrawTradeupFollowsContributionWeights(t *testing.T) {
	byCollection := map[int]map[string][]*data.Card{
		1: {"rare": {testCard("a-rare", "rare")}},
		2: {"rare": {testCard("b-rare", "rare")}},
	}
	outcomes := tradeupOutcomes(byCollection, "common", map[int]int{1: 7, 2: 3})

	rnd := rand.New(rand.NewPCG(1, 2))
	counts := map[int]int{}
	samples := 10_000
	for range samples {
		_, collectionID := drawTradeup(rnd, outcomes)
		counts[collectionID]++
	}

	// 7 of the contract's cards come from collection 1, 3 from collection 2
	expected := map[int]float64{1: 0.7, 2: 0.3}
	for id, ratio := range expected {
		got := float64(counts[id]) / float64(samples)
		if got < ratio-0.03 || got > ratio+0.03 {
			t.Errorf("collection %d: share %.3f, want %.2f ± 0.03", id, got, ratio)
		}
	}
}

func TestDrawTradeupNoOutcomes(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))

	if card, collectionID := drawTradeup(rnd, nil); card != nil || collectionID != 0 {
		t.Errorf("drawTradeup(nil) = (%v, %d), want (nil, 0)", card, collectionID)
	}
	// Outcomes that contribute nothing can't win either
	if card, _ := drawTradeup(rnd, []tradeupOutcome{{CollectionID: 1}}); card != nil {
		t.Errorf("drawTradeup with no weight drew %v, want nil", card)
	}
}
