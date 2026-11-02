package data

import (
	"math/rand/v2"
	"testing"
)

func TestFoilRouletteChanceBaseOdds(t *testing.T) {
	cases := map[string]int{"common": 100, "rare": 200, "legendary": 500}
	for rarity, want := range cases {
		got, ok := foilRouletteChance(rarity, nil)
		if !ok || got != want {
			t.Errorf("foilRouletteChance(%q, nil) = %d %v, want %d", rarity, got, ok, want)
		}
	}
}

func TestFoilRouletteChanceAddsSacrifices(t *testing.T) {
	// 1% + 0.5% + 1% + 2%
	got, ok := foilRouletteChance("common", []string{"common", "rare", "legendary"})
	if !ok || got != 450 {
		t.Errorf("got %d %v, want 450", got, ok)
	}
}

func TestFoilRouletteChanceIsCapped(t *testing.T) {
	sacrificed := make([]string, 30)
	for i := range sacrificed {
		sacrificed[i] = "legendary"
	}
	got, ok := foilRouletteChance("legendary", sacrificed)
	if !ok || got != foilRouletteCap {
		t.Errorf("got %d %v, want %d", got, ok, foilRouletteCap)
	}
}

func TestFoilRouletteChanceRejectsUnknownRarities(t *testing.T) {
	if _, ok := foilRouletteChance("unknown", nil); ok {
		t.Error("unknown rolled rarity accepted")
	}
	if _, ok := foilRouletteChance("common", []string{"unknown"}); ok {
		t.Error("unknown sacrificed rarity accepted")
	}
}

func TestRollFoilRoulette(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))
	for range 100 {
		if got := rollFoilRoulette(rnd, 0); got != "" {
			t.Fatalf("won %q at 0%%", got)
		}
		got := rollFoilRoulette(rnd, foilRouletteScale)
		if getFoilPattern(got) == nil {
			t.Fatalf("won unknown pattern %q at 100%%", got)
		}
	}
}

func TestRollFoilRouletteFollowsTheChance(t *testing.T) {
	rnd := rand.New(rand.NewPCG(3, 4))
	won := 0
	const rolls = 20000
	for range rolls {
		if rollFoilRoulette(rnd, foilRouletteCap) != "" {
			won++
		}
	}
	// 50% expected, well within a few standard deviations
	if won < rolls*45/100 || won > rolls*55/100 {
		t.Errorf("won %d of %d rolls at 50%%", won, rolls)
	}
}
