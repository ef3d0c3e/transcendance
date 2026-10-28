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

// streakTier is one tier of the reward ladder in the streak meter
type streakTier struct {
	// Tier number and coin value of the tier, e.g "3 · 500"
	Label string
	// Whether the tier is the one of the bonus of the day
	Current bool
}

// streakTile is the template data of the daily streak section, consumed by
// the "daily-streak" partial
type streakTile struct {
	Title string
	// Reward ladder, one cell per tier, the current one highlighted
	Tiers []streakTier
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
		ClaimAction: "/inventory/claim",
	}
	// The meter shows the tier number and the coin value of every tier,
	// the one of the bonus of the day standing out
	tile.Tiers = make([]streakTier, 0, len(state.Rewards))
	for i, amount := range state.Rewards {
		tier := uint(i + 1)
		tile.Tiers = append(tile.Tiers, streakTier{
			Label:   dc.Renderer.T(c, "daily-streak-meter", "tier", int(tier), "amount", int(amount)),
			Current: tier == state.Tier,
		})
	}
	// The streak count is only shown once it climbs past the reward cap
	if state.Day > uint(len(state.Rewards)) {
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
