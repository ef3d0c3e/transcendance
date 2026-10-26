package data

import (
	"math/rand/v2"
)

// A foil pattern a card can be obtained with
type foilPattern struct {
	// Stored on the card, also the mask file name in `data/assets`
	Name string
	// Size of the pattern tiles on the card
	Scale float64
}

// Foil patterns obtainable when opening a booster
var foilPatterns = []foilPattern{
	{Name: "foil1", Scale: 0.5},
	{Name: "foil2", Scale: 0.5},
	{Name: "foil3", Scale: 0.5},
	{Name: "foil4", Scale: 0.5},
	{Name: "foil5", Scale: 0.5},
	{Name: "foil6", Scale: 0.5},
}

// Chance for an obtained card to be foil, in percent
const foilChance = 5

// Foil color and opacity, shared by every pattern
const (
	foilColor   = "#ffaf5f"
	foilOpacity = 0.8
)

// Roll the foil of an obtained card, "" when the card isn't foil
func rollFoil(rnd *rand.Rand) string {
	if rnd.N(100) >= foilChance {
		return ""
	}
	return foilPatterns[rnd.N(len(foilPatterns))].Name
}

// Find a foil pattern by name
func getFoilPattern(name string) *foilPattern {
	for i := range foilPatterns {
		if foilPatterns[i].Name == name {
			return &foilPatterns[i]
		}
	}
	return nil
}

// Add the card's foil to its template data, unknown patterns are ignored
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
