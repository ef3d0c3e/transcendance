package data

import (
	"fmt"
	"html/template"
	"time"

	"transcendance/models"

	"github.com/gin-gonic/gin"
)

// The ?booster= value opening a default booster out of the user's pile
const defaultBoosterRef = "default"

const defaultBoosterSize = 5

// Gold of the default boosters, matching the coins
const (
	defaultBoosterColorPrimary   = "#e8b64c"
	defaultBoosterColorSecondary = "#8a5f1a"
)

// Template data of the "booster-pile" partial and the tile around it
type pileTile struct {
	PileTitle       string
	PileDescription string
	// "×N" pill of the top left corner
	PileCountLabel string
	// Countdown pill of the top right corner, hidden while the pile is full
	PileTimerLabel   string
	PileTimerSeconds int
	// Whether the pile holds no booster: the art fades out
	PileEmpty bool
	// Page opening one default booster out of the pile
	PileOpen string
	// Accessible name of the tile, which opens one
	PileLabel string

	ColorPrimary   template.CSS
	ColorSecondary template.CSS
}

func (dc *DataController) pileTileData(c *gin.Context, pile *models.BoosterPile) *pileTile {
	// Countdown to the next booster, none while the pile is full
	timerLabel := ""
	timerSeconds := 0
	if pile.NextAt != nil {
		remaining := int(time.Until(*pile.NextAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		timerSeconds = remaining
		timerLabel = formatCountdown(remaining)
	}

	return &pileTile{
		PileTitle:        dc.Renderer.T(c, "booster-pile-title"),
		PileDescription:  dc.Renderer.T(c, "booster-pile-desc", "count", defaultBoosterSize),
		PileCountLabel:   fmt.Sprintf("×%d", pile.Count),
		PileTimerLabel:   timerLabel,
		PileTimerSeconds: timerSeconds,
		PileEmpty:        pile.Count == 0,
		PileOpen:         "/opening?booster=" + defaultBoosterRef,
		PileLabel:        dc.Renderer.T(c, "inventory-booster-open", "name", dc.Renderer.T(c, "booster-pile-title")),
		ColorPrimary:     template.CSS(defaultBoosterColorPrimary),
		ColorSecondary:   template.CSS(defaultBoosterColorSecondary),
	}
}
