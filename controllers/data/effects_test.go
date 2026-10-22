package data

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRollFoilChance(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))

	const rolls = 100000
	foils := 0
	seen := map[string]int{}
	for range rolls {
		foil := rollFoil(rnd)
		if foil == "" {
			continue
		}
		foils++
		seen[foil]++
		require.NotNil(t, getFoilPattern(foil))
	}

	// About foilChance percent of cards are foil, with every pattern
	require.InDelta(t, float64(foilChance)/100, float64(foils)/rolls, 0.005)
	require.Len(t, seen, len(foilPatterns))
}

func TestSetCardFoil(t *testing.T) {
	cardData := map[string]any{}
	setCardFoil(cardData, "foil3")
	require.Equal(t, "/cards/assets/foil3.png", cardData["CardFoilMask"])
	require.Equal(t, getFoilPattern("foil3").Scale, cardData["CardFoilScale"])

	// No foil, or a pattern that no longer exists: plain card
	for _, foil := range []string{"", "foil-removed"} {
		cardData := map[string]any{}
		setCardFoil(cardData, foil)
		require.NotContains(t, cardData, "CardFoilMask")
	}
}
