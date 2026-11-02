package data

import (
	"math/rand/v2"
	"testing"

	"transcendance/data"
)

// A card of a tradeup pool, from the collection of the given ID
func tradeupCard(collectionID int, name string, rarity string) poolCard {
	return poolCard{card: testCard(name, rarity), collection: &data.Collection{ID: collectionID}}
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
	byCollection := map[int]map[string][]poolCard{
		1: {"rare": {tradeupCard(1, "a-rare", "rare")}},
		2: {"rare": {tradeupCard(2, "b-rare", "rare")}},
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

func TestTradeupOutcomesSkipsCollectionsWithoutTargetTier(t *testing.T) {
	// Only the first collection holds a rare card, so it takes the whole odds
	byCollection := map[int]map[string][]poolCard{
		1: {"rare": {tradeupCard(1, "a-rare", "rare")}},
		2: {"common": {tradeupCard(2, "b-common", "common")}},
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
	byCollection := map[int]map[string][]poolCard{
		1: {"common": {tradeupCard(1, "a-common", "common")}},
	}

	if outcomes := tradeupOutcomes(byCollection, "common", map[int]int{1: 10}); len(outcomes) != 0 {
		t.Errorf("got %d outcomes, want 0", len(outcomes))
	}
	if outcomes := tradeupOutcomes(byCollection, "legendary", map[int]int{1: 10}); len(outcomes) != 0 {
		t.Errorf("got %d outcomes for the highest rarity, want 0", len(outcomes))
	}
}

func TestDrawTradeupPicksTargetTierCards(t *testing.T) {
	wanted := []poolCard{tradeupCard(1, "rare-1", "rare"), tradeupCard(1, "rare-2", "rare")}
	byCollection := map[int]map[string][]poolCard{
		1: {"rare": wanted, "common": {tradeupCard(1, "common", "common")}},
	}
	outcomes := tradeupOutcomes(byCollection, "common", map[int]int{1: 10})

	rnd := rand.New(rand.NewPCG(1, 2))
	seen := map[string]bool{}
	for range 100 {
		drawn, ok := drawTradeup(rnd, outcomes)
		if !ok {
			t.Fatal("drawTradeup returned no card")
		}
		if drawn.collection.ID != 1 || drawn.card.Rarity != "rare" {
			t.Errorf("drew %q (%s) of collection %d, want a rare of collection 1", drawn.card.Name, drawn.card.Rarity, drawn.collection.ID)
		}
		seen[drawn.card.Name] = true
	}
	// Both cards of the tier must be reachable
	for _, entry := range wanted {
		if !seen[entry.card.Name] {
			t.Errorf("card %q was never drawn", entry.card.Name)
		}
	}
}

func TestDrawTradeupFollowsContributionWeights(t *testing.T) {
	byCollection := map[int]map[string][]poolCard{
		1: {"rare": {tradeupCard(1, "a-rare", "rare")}},
		2: {"rare": {tradeupCard(2, "b-rare", "rare")}},
	}
	outcomes := tradeupOutcomes(byCollection, "common", map[int]int{1: 7, 2: 3})

	rnd := rand.New(rand.NewPCG(1, 2))
	counts := map[int]int{}
	samples := 10_000
	for range samples {
		drawn, _ := drawTradeup(rnd, outcomes)
		counts[drawn.collection.ID]++
	}

	// 7 of the contract's cards come from collection 1, 3 from collection 2
	for id, ratio := range map[int]float64{1: 0.7, 2: 0.3} {
		got := float64(counts[id]) / float64(samples)
		if got < ratio-0.03 || got > ratio+0.03 {
			t.Errorf("collection %d: share %.3f, want %.2f ± 0.03", id, got, ratio)
		}
	}
}

func TestDrawTradeupNoOutcomes(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))

	if _, ok := drawTradeup(rnd, nil); ok {
		t.Error("drawTradeup(nil) drew a card")
	}
	// Outcomes that contribute nothing can't win either
	if _, ok := drawTradeup(rnd, []tradeupOutcome{{CollectionID: 1}}); ok {
		t.Error("drawTradeup with no weight drew a card")
	}
}
