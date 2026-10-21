package data

import (
	"math/rand/v2"
	"testing"

	"transcendance/data"
)

func testCard(name string, rarity string) *data.Card {
	return &data.Card{Name: name, Rarity: rarity}
}

func TestRollRarityDistribution(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))
	counts := map[string]int{}
	samples := 100_000

	for i := 0; i < samples; i++ {
		counts[rollRarity(rnd)]++
	}

	// Drop table is common 80%, rare 15%, legendary 5%
	expected := map[string]float64{
		"common":    0.80,
		"rare":      0.15,
		"legendary": 0.05,
	}
	for rarity, ratio := range expected {
		got := float64(counts[rarity]) / float64(samples)
		if got < ratio-0.02 || got > ratio+0.02 {
			t.Errorf("rarity %s: share %.3f, want %.2f ± 0.02", rarity, got, ratio)
		}
	}
}

func TestPickTierKeepsRolledTier(t *testing.T) {
	available := map[string]bool{"common": true, "rare": true, "legendary": true}

	for _, rolled := range rarityTiers {
		if tier := pickTier(rolled, available); tier != rolled {
			t.Errorf("pickTier(%s) = %q, want %q", rolled, tier, rolled)
		}
	}
}

func TestPickTierFallsBackToLower(t *testing.T) {
	cases := []struct {
		name      string
		rolled    string
		available map[string]bool
		want      string
	}{
		{
			name:      "legendary falls back to rare",
			rolled:    "legendary",
			available: map[string]bool{"common": true, "rare": true},
			want:      "rare",
		},
		{
			name:      "legendary falls back to common",
			rolled:    "legendary",
			available: map[string]bool{"common": true},
			want:      "common",
		},
		{
			name:      "rare falls back to common",
			rolled:    "rare",
			available: map[string]bool{"common": true},
			want:      "common",
		},
		{
			name:      "legendary skips empty rare",
			rolled:    "legendary",
			available: map[string]bool{"common": true, "rare": false, "legendary": false},
			want:      "common",
		},
		{
			name:      "lower tier only is used when higher rolled",
			rolled:    "rare",
			available: map[string]bool{"legendary": true},
			want:      "legendary",
		},
		{
			name:      "common falls back up when it has no card",
			rolled:    "common",
			available: map[string]bool{"legendary": true},
			want:      "legendary",
		},
		{
			name:      "empty collection yields nothing",
			rolled:    "legendary",
			available: map[string]bool{},
			want:      "",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := pickTier(testCase.rolled, testCase.available)
			if got != testCase.want {
				t.Errorf("pickTier(%s, %v) = %q, want %q", testCase.rolled, testCase.available, got, testCase.want)
			}
		})
	}
}

func TestDrawPackSize(t *testing.T) {
	byRarity := map[string][]*data.Card{
		"common":    {testCard("a", "common"), testCard("b", "common")},
		"rare":      {testCard("c", "rare")},
		"legendary": {testCard("d", "legendary")},
	}

	for _, seed := range []uint64{1, 2, 42, 1000} {
		rnd := rand.New(rand.NewPCG(seed, seed))
		pack := drawPack(rnd, byRarity)

		if pack == nil {
			t.Fatalf("seed %d: drawPack returned nil", seed)
		}
		if len(pack) != PackSize {
			t.Fatalf("seed %d: drawPack returned %d cards, want %d", seed, len(pack), PackSize)
		}
	}
}

func TestDrawPackOnlyYieldsCardsFromCollection(t *testing.T) {
	pool := map[string]bool{"a": true, "b": true, "c": true}
	byRarity := map[string][]*data.Card{
		"common":    {testCard("a", "common")},
		"rare":      {testCard("b", "rare")},
		"legendary": {testCard("c", "legendary")},
	}

	rnd := rand.New(rand.NewPCG(7, 7))
	for i := 0; i < 500; i++ {
		for _, card := range drawPack(rnd, byRarity) {
			if !pool[card.Name] {
				t.Fatalf("drew card %q which is not in the collection", card.Name)
			}
		}
	}
}

func TestDrawPackRespectsSingleRarityPool(t *testing.T) {
	byRarity := map[string][]*data.Card{
		"common": {testCard("only", "common")},
	}

	rnd := rand.New(rand.NewPCG(3, 3))
	for i := 0; i < 100; i++ {
		for _, card := range drawPack(rnd, byRarity) {
			if card.Rarity != "common" {
				t.Fatalf("drew rarity %q from a common-only collection", card.Rarity)
			}
		}
	}
}

func TestDrawPackEmptyCollection(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))
	if pack := drawPack(rnd, map[string][]*data.Card{}); pack != nil {
		t.Fatalf("drawPack on an empty collection = %v, want nil", pack)
	}
}

func TestBuildReelPlacesWinnerInWindow(t *testing.T) {
	byRarity := map[string][]*data.Card{
		"common":    {testCard("a", "common"), testCard("b", "common")},
		"rare":      {testCard("c", "rare")},
		"legendary": {testCard("d", "legendary")},
	}
	winner := testCard("w", "legendary")

	rnd := rand.New(rand.NewPCG(11, 11))
	strip := buildReel(rnd, flattenByRarity(byRarity), winner)

	want := reelAbove + 1 + reelBelow
	if len(strip) != want {
		t.Fatalf("strip length = %d, want %d", len(strip), want)
	}

	for i, entry := range strip {
		shouldBeWin := i == reelAbove
		if entry.Win != shouldBeWin {
			t.Errorf("strip[%d].Win = %v, want %v", i, entry.Win, shouldBeWin)
		}
	}

	// The won card must survive the trip through the strip
	if got := strip[reelAbove].Card; got != winner {
		t.Errorf("strip[%d] = %q, want winner %q", reelAbove, got.Name, winner.Name)
	}

	// The cards beneath the winner are the ones that run past as the reel
	// drops: they have to be there or the spin is too short
	if reelBelow < 8 {
		t.Errorf("reelBelow = %d, want at least 8 cards of spin", reelBelow)
	}

	// Fillers must come from the collection pool
	allowed := map[string]bool{"a": true, "b": true, "c": true, "d": true}
	for i, entry := range strip {
		if entry.Win {
			continue
		}
		if !allowed[entry.Card.Name] {
			t.Errorf("strip[%d] = %q, which is not in the collection", i, entry.Card.Name)
		}
	}
}

func TestBuildReelEmptyPool(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))
	winner := testCard("w", "common")

	if strip := buildReel(rnd, nil, winner); strip != nil {
		t.Fatalf("buildReel with an empty pool = %v, want nil", strip)
	}
}

func TestPackReelsHoldOneWinnerEach(t *testing.T) {
	byRarity := map[string][]*data.Card{
		"common":    {testCard("a", "common"), testCard("b", "common")},
		"rare":      {testCard("c", "rare")},
		"legendary": {testCard("d", "legendary")},
	}

	rnd := rand.New(rand.NewPCG(21, 21))
	pack := drawPack(rnd, byRarity)
	if pack == nil {
		t.Fatal("drawPack returned nil")
	}

	pool := flattenByRarity(byRarity)
	for i, winner := range pack {
		strip := buildReel(rnd, pool, winner)

		wins := 0
		for _, entry := range strip {
			if entry.Win {
				wins++
			}
		}
		if wins != 1 {
			t.Fatalf("reel %d holds %d winners, want exactly 1", i, wins)
		}
		if got := strip[reelAbove].Card; got != winner {
			t.Errorf("reel %d stops on %q, want winner %q", i, got.Name, winner.Name)
		}
	}
}
