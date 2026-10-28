package data

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"transcendance/controllers"

	"github.com/gin-gonic/gin"
)

// formatCountdown renders a number of seconds as M:SS, or H:MM:SS past an hour
func formatCountdown(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, (seconds/60)%60, seconds%60)
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

// streakTile is the template data of the daily streak section, consumed by
// the "daily-streak" partial
type streakTile struct {
	Title     string
	TierLabel string
	// Raw streak count, shown past the reward cap
	StreakLabel string
	BestLabel   string
	// Claim button: label and countdown once the bonus is claimed
	ClaimLabel    string
	ClaimDisabled bool
	TimerSeconds  int
	TimerValue    string
	// Form claiming the bonus of the day
	ClaimAction string
	// Tab the section is rendered on, kept when claiming
	Tab string
}

// streakTileData builds the template data of the daily streak section
func (dc *DataController) streakTileData(c *gin.Context, state *controllers.StreakState) *streakTile {
	tile := &streakTile{
		Title:       dc.Renderer.T(c, "daily-streak-title"),
		TierLabel:   dc.Renderer.T(c, "daily-streak-tier", "tier", int(state.Tier), "tiers", int(state.Tiers)),
		ClaimAction: "/inventory/claim",
	}
	// The streak count is only shown once it climbs past the reward cap
	if state.Day > state.Tiers {
		tile.StreakLabel = dc.Renderer.T(c, "daily-streak-count", "days", int(state.Day))
	}
	// The best streak shows as a number, or as "current" while the ongoing
	// streak holds the record
	var best any = dc.Renderer.T(c, "daily-streak-none")
	if state.Best > 0 {
		if state.Day > 0 && state.Day == state.Best {
			best = dc.Renderer.T(c, "daily-streak-best-current")
		} else {
			best = int(state.Best)
		}
	}
	tile.BestLabel = dc.Renderer.T(c, "daily-streak-best", "best", best)

	if state.Claimed {
		// The bonus of the day comes back at midnight, server time
		remaining := int(time.Until(state.NextAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		tile.ClaimLabel = dc.Renderer.T(c, "daily-streak-claimed")
		tile.ClaimDisabled = true
		tile.TimerSeconds = remaining
		tile.TimerValue = formatCountdown(remaining)
	} else {
		tile.ClaimLabel = dc.Renderer.T(c, "daily-streak-claim", "amount", int(state.Reward))
	}
	return tile
}

// InventoryClaimPost claims the connection streak reward of the day
func (dc *DataController) InventoryClaimPost(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)

	if user == nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	err := controllers.ClaimDailyReward(dc.DB, user, time.Now())
	if err != nil && !errors.Is(err, controllers.ErrDailyRewardClaimed) {
		log.Printf("WARN failed to claim the daily reward of user %d: %s", user.ID, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Back to the tab the bonus was claimed from
	target := "/inventory"
	if c.PostForm("tab") == inventoryTabBoosters {
		target += "?tab=" + inventoryTabBoosters
	}
	c.Redirect(http.StatusSeeOther, target)
}
