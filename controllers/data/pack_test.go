package data

import (
	"math/rand/v2"
	"testing"

	"transcendance/data"
)

// Size of the packs drawn in tests
const testPackSize = 5

func testCard(name string, rarity string) *data.Card {
	return &data.Card{Name: name, Rarity: rarity}
}

// Collection of every card of the test pools
var testCollection = &data.Collection{}

func testPoolCard(name string, rarity string) poolCard {
	return poolCard{card: testCard(name, rarity), collection: testCollection}
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

func TestDrawtestPackSize(t *testing.T) {
	byRarity := map[string][]poolCard{
		"common":    {testPoolCard("a", "common"), testPoolCard("b", "common")},
		"rare":      {testPoolCard("c", "rare")},
		"legendary": {testPoolCard("d", "legendary")},
	}

	for _, seed := range []uint64{1, 2, 42, 1000} {
		rnd := rand.New(rand.NewPCG(seed, seed))
		pack := drawPack(rnd, byRarity, testPackSize)

		if pack == nil {
			t.Fatalf("seed %d: drawPack returned nil", seed)
		}
		if len(pack) != testPackSize {
			t.Fatalf("seed %d: drawPack returned %d cards, want %d", seed, len(pack), testPackSize)
		}
	}
}

func TestDrawPackOnlyYieldsCardsFromCollection(t *testing.T) {
	pool := map[string]bool{"a": true, "b": true, "c": true}
	byRarity := map[string][]poolCard{
		"common":    {testPoolCard("a", "common")},
		"rare":      {testPoolCard("b", "rare")},
		"legendary": {testPoolCard("c", "legendary")},
	}

	rnd := rand.New(rand.NewPCG(7, 7))
	for i := 0; i < 500; i++ {
		for _, drawn := range drawPack(rnd, byRarity, testPackSize) {
			if !pool[drawn.card.Name] {
				t.Fatalf("drew card %q which is not in the collection", drawn.card.Name)
			}
		}
	}
}

func TestDrawPackRespectsSingleRarityPool(t *testing.T) {
	byRarity := map[string][]poolCard{
		"common": {testPoolCard("only", "common")},
	}

	rnd := rand.New(rand.NewPCG(3, 3))
	for i := 0; i < 100; i++ {
		for _, drawn := range drawPack(rnd, byRarity, testPackSize) {
			if drawn.card.Rarity != "common" {
				t.Fatalf("drew rarity %q from a common-only collection", drawn.card.Rarity)
			}
		}
	}
}

func TestDrawPackEmptyCollection(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))
	if pack := drawPack(rnd, map[string][]poolCard{}, testPackSize); pack != nil {
		t.Fatalf("drawPack on an empty collection = %v, want nil", pack)
	}
}

func TestBuildReelPlacesWinnerInWindow(t *testing.T) {
	byRarity := map[string][]poolCard{
		"common":    {testPoolCard("a", "common"), testPoolCard("b", "common")},
		"rare":      {testPoolCard("c", "rare")},
		"legendary": {testPoolCard("d", "legendary")},
	}
	winner := testPoolCard("w", "legendary")

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
	if got := strip[reelAbove].Card; got != winner.card {
		t.Errorf("strip[%d] = %q, want winner %q", reelAbove, got.Name, winner.card.Name)
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
	winner := testPoolCard("w", "common")

	if strip := buildReel(rnd, nil, winner); strip != nil {
		t.Fatalf("buildReel with an empty pool = %v, want nil", strip)
	}
}

func TestPackReelsHoldOneWinnerEach(t *testing.T) {
	byRarity := map[string][]poolCard{
		"common":    {testPoolCard("a", "common"), testPoolCard("b", "common")},
		"rare":      {testPoolCard("c", "rare")},
		"legendary": {testPoolCard("d", "legendary")},
	}

	rnd := rand.New(rand.NewPCG(21, 21))
	pack := drawPack(rnd, byRarity, testPackSize)
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
		if got := strip[reelAbove].Card; got != winner.card {
			t.Errorf("reel %d stops on %q, want winner %q", i, got.Name, winner.card.Name)
		}
	}
}

func TestDrawDefaultPackSize(t *testing.T) {
	pool := []poolCard{
		testPoolCard("a", "common"),
		testPoolCard("b", "rare"),
	}

	rnd := rand.New(rand.NewPCG(1, 2))
	pack := drawDefaultPack(rnd, pool, testPackSize)
	if pack == nil {
		t.Fatal("drawDefaultPack returned nil")
	}
	if len(pack) != testPackSize {
		t.Fatalf("drawDefaultPack returned %d cards, want %d", len(pack), testPackSize)
	}
}

func TestDrawDefaultPackEmptyPool(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))
	if pack := drawDefaultPack(rnd, nil, testPackSize); pack != nil {
		t.Fatalf("drawDefaultPack on an empty pool = %v, want nil", pack)
	}
}

func TestDrawDefaultPackIsUniformAcrossCollections(t *testing.T) {
	// A collection with one card and a collection with three: every card
	// must be drawn as often as the others, whatever its collection
	small := &data.Collection{ID: 1}
	big := &data.Collection{ID: 2}
	pool := []poolCard{
		{card: testCard("s", "common"), collection: small},
		{card: testCard("x", "common"), collection: big},
		{card: testCard("y", "rare"), collection: big},
		{card: testCard("z", "legendary"), collection: big},
	}

	rnd := rand.New(rand.NewPCG(3, 4))
	counts := map[string]int{}
	samples := 100_000
	for i := 0; i < samples; i++ {
		for _, drawn := range drawDefaultPack(rnd, pool, 1) {
			counts[drawn.card.Name]++
		}
	}

	for _, drawn := range pool {
		share := float64(counts[drawn.card.Name]) / float64(samples)
		if share < 0.25-0.02 || share > 0.25+0.02 {
			t.Errorf("card %s: share %.3f, want 0.25 ± 0.02", drawn.card.Name, share)
		}
	}
}

func TestDrawDefaultPackKeepsCardCollections(t *testing.T) {
	small := &data.Collection{ID: 1}
	big := &data.Collection{ID: 2}
	pool := []poolCard{
		{card: testCard("s", "common"), collection: small},
		{card: testCard("x", "common"), collection: big},
	}

	rnd := rand.New(rand.NewPCG(5, 6))
	for _, drawn := range drawDefaultPack(rnd, pool, 50) {
		if drawn.collection == nil {
			t.Fatalf("drew card %q with no collection", drawn.card.Name)
		}
		if drawn.collection != small && drawn.collection != big {
			t.Fatalf("drew card %q with an unknown collection", drawn.card.Name)
		}
	}
}
