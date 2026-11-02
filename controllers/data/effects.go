package data

import (
	crand "crypto/rand"
	"math/rand/v2"

	"transcendance/models"
)

type foilPattern struct {
	// Stored on the card, also the mask file name in `data/assets`
	Name string
	// Size of the pattern tiles on the card
	Scale float64
}

var foilPatterns = func() []foilPattern {
	patterns := make([]foilPattern, len(models.FoilPatterns))
	for i, name := range models.FoilPatterns {
		patterns[i] = foilPattern{Name: name, Scale: 0.5}
	}
	return patterns
}()

// Chance for an obtained card to be foil, in percent
const foilChance = 5

const (
	foilColor   = "#ffaf5f"
	foilOpacity = 0.8
)

func rollFoil(rnd *rand.Rand) string {
	if rnd.N(100) >= foilChance {
		return ""
	}
	return foilPatterns[rnd.N(len(foilPatterns))].Name
}

func getFoilPattern(name string) *foilPattern {
	for i := range foilPatterns {
		if foilPatterns[i].Name == name {
			return &foilPatterns[i]
		}
	}
	return nil
}

func setCardFoil(cardData map[string]any, foil string) {
	pattern := getFoilPattern(foil)
	if pattern == nil {
		return
	}
	cardData["CardFoilMask"] = "/cards/assets/" + pattern.Name + ".png"
	cardData["CardFoilScale"] = pattern.Scale
	cardData["CardFoilColor"] = foilColor
	cardData["CardFoilOpacity"] = foilOpacity
}

// Seeded by the system, so two draws at the same time get different cards
func newRand() *rand.Rand {
	var seed [32]byte
	_, _ = crand.Read(seed[:])
	return rand.New(rand.NewChaCha8(seed))
}
