package controllers

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"transcendance/models"
)

// ErrNoPileBooster is returned when the pile holds no default booster.
var ErrNoPileBooster = errors.New("no default booster available")

// ErrDailyRewardClaimed is returned when the bonus of the day is claimed already.
var ErrDailyRewardClaimed = errors.New("daily reward claimed already")

// ensureBoosterPile returns a user's pile of default boosters, creating it
// when the account predates the pile. A new pile starts refilling right away.
func ensureBoosterPile(db *gorm.DB, ownerID uint, now time.Time) (models.BoosterPile, error) {
	var pile models.BoosterPile
	err := db.Where("owner_id = ?", ownerID).First(&pile).Error
	if err == nil {
		return pile, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return pile, err
	}

	next := now.Add(boosterPileInterval)
	pile = models.BoosterPile{OwnerID: ownerID, NextAt: &next}
	if err := db.Create(&pile).Error; err != nil {
		// Another request created the pile first: use its row
		if err := db.Where("owner_id = ?", ownerID).First(&pile).Error; err != nil {
			return pile, err
		}
	}
	return pile, nil
}

// accrueBoosterPile adds the boosters that refilled until now, and reports
// whether the pile changed. While the pile is not full, one booster joins it
// every boosterPileInterval, counting from the moment it stopped being full.
func accrueBoosterPile(pile *models.BoosterPile, now time.Time) bool {
	if pile.Count >= boosterPileCap {
		changed := pile.Count != boosterPileCap || pile.NextAt != nil
		pile.Count = boosterPileCap
		pile.NextAt = nil
		return changed
	}
	if pile.NextAt == nil {
		// The pile just stopped being full (or is new): start the countdown
		next := now.Add(boosterPileInterval)
		pile.NextAt = &next
		return true
	}
	changed := false
	for pile.Count < boosterPileCap && !now.Before(*pile.NextAt) {
		pile.Count++
		next := pile.NextAt.Add(boosterPileInterval)
		pile.NextAt = &next
		changed = true
	}
	if pile.Count >= boosterPileCap {
		pile.NextAt = nil
	}
	return changed
}

// AccrueBoosterPile tops a user's pile up to now and returns its state.
func AccrueBoosterPile(db *gorm.DB, ownerID uint) (models.BoosterPile, error) {
	now := time.Now()
	pile, err := ensureBoosterPile(db, ownerID, now)
	if err != nil {
		return pile, err
	}
	if !accrueBoosterPile(&pile, now) {
		return pile, nil
	}
	if err := db.Save(&pile).Error; err != nil {
		return pile, err
	}
	return pile, nil
}

// ConsumePileBooster takes one default booster out of a user's pile inside a
// transaction, returning the pile it has left. Only a full pile stops being
// full, so taking one out of a full pile starts the countdown, while taking
// one out of a pile that already refills leaves the countdown alone.
// Returns ErrNoPileBooster when the pile is empty.
func ConsumePileBooster(tx *gorm.DB, ownerID uint) (models.BoosterPile, error) {
	now := time.Now()
	pile, err := AccrueBoosterPile(tx, ownerID)
	if err != nil {
		return pile, err
	}
	if pile.Count == 0 {
		return pile, ErrNoPileBooster
	}

	updates := map[string]any{"count": gorm.Expr("count - 1")}
	var nextAt *time.Time
	if pile.NextAt == nil {
		next := now.Add(boosterPileInterval)
		updates["next_at"] = next
		nextAt = &next
	}
	// Taking the last booster with a conditional update keeps concurrent
	// openings from digging into an empty pile
	result := tx.Model(&models.BoosterPile{}).
		Where("owner_id = ? AND count > 0", ownerID).
		Updates(updates)
	if result.Error != nil {
		return pile, result.Error
	}
	if result.RowsAffected == 0 {
		return pile, ErrNoPileBooster
	}

	pile.Count--
	if nextAt != nil {
		pile.NextAt = nextAt
	}
	return pile, nil
}

// ensureDailyStreak returns a user's connection streak, creating it when the
// account predates the streaks.
func ensureDailyStreak(db *gorm.DB, userID uint) (models.DailyStreak, error) {
	var streak models.DailyStreak
	err := db.Where("user_id = ?", userID).First(&streak).Error
	if err == nil {
		return streak, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return streak, err
	}

	streak = models.DailyStreak{UserID: userID}
	if err := db.Create(&streak).Error; err != nil {
		// Another request created the streak first: use its row
		if err := db.Where("user_id = ?", userID).First(&streak).Error; err != nil {
			return streak, err
		}
	}
	return streak, nil
}

// startOfDay returns the start of a day in the server's timezone
func startOfDay(t time.Time) time.Time {
	local := t.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
}

// ClaimDailyReward credits the connection streak reward of the day to a
// user's balance, once per day, growing their streak. A missed day resets
// the streak to day 1. `user.Coins` is updated to the new balance.
// Returns ErrDailyRewardClaimed when the bonus of the day is claimed already.
func ClaimDailyReward(db *gorm.DB, user *AuthenticatedUser, now time.Time) error {
	today := startOfDay(now)
	yesterday := startOfDay(now.AddDate(0, 0, -1))

	streak, err := ensureDailyStreak(db, user.ID)
	if err != nil {
		return err
	}

	day := uint(1)
	if streak.LastDay != nil {
		last := startOfDay(*streak.LastDay)
		switch {
		case last.Equal(today):
			// The day is claimed already
			return ErrDailyRewardClaimed
		case last.Equal(yesterday):
			// Consecutive day: the streak keeps growing, past the reward cap
			day = streak.Day + 1
		}
		// A wider gap resets the streak to day 1
	}

	reward := dailyStreakReward(day)
	return db.Transaction(func(tx *gorm.DB) error {
		// Claim only when no request claimed this day yet, so two claims
		// the same day cannot pay it twice
		result := tx.Model(&models.DailyStreak{}).
			Where("user_id = ? AND (last_day IS NULL OR last_day < ?)", user.ID, today).
			Updates(map[string]any{
				"day":      day,
				"last_day": today,
				"best":     max(bestStreak(&streak), day),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrDailyRewardClaimed
		}

		if err := tx.Model(&models.User{}).
			Where("id = ?", user.ID).
			Update("coins", gorm.Expr("coins + ?", reward)).Error; err != nil {
			return err
		}

		user.Coins += reward
		return nil
	})
}

// bestStreak returns the best streak recorded for a user, counting the
// streaks that predate the best-streak tracking
func bestStreak(streak *models.DailyStreak) uint {
	return max(streak.Best, streak.Day)
}

// DailyStreakState describes a user's connection streak for display
type StreakState struct {
	// Consecutive days claimed in a row, 0 when the streak is over
	Day uint
	// Tier of the bonus of the day, 1 to Tiers
	Tier uint
	// Number of tiers of the reward ladder
	Tiers uint
	// Coins the bonus of the day pays
	Reward uint
	// Best streak the user has done
	Best uint
	// Whether the bonus of the day is claimed already
	Claimed bool
	// Start of the next day, server time, when the bonus comes back
	NextAt time.Time
}

// DailyStreakState returns the state of a user's connection streak at `now`
func DailyStreakState(db *gorm.DB, userID uint, now time.Time) (StreakState, error) {
	streak, err := ensureDailyStreak(db, userID)
	if err != nil {
		return StreakState{}, err
	}

	today := startOfDay(now)
	yesterday := startOfDay(now.AddDate(0, 0, -1))
	state := StreakState{
		Tiers:  uint(len(dailyStreakRewards)),
		Best:   bestStreak(&streak),
		NextAt: today.AddDate(0, 0, 1),
	}
	// The bonus of the day is the one the streak is on, or the one that
	// restarts it after a missed day
	next := uint(1)
	if streak.LastDay != nil {
		last := startOfDay(*streak.LastDay)
		switch {
		case last.Equal(today):
			state.Claimed = true
			state.Day = streak.Day
			next = streak.Day
		case last.Equal(yesterday):
			state.Day = streak.Day
			next = streak.Day + 1
		}
		// A wider gap leaves the streak over: the bonus of the day is day 1
	}
	state.Tier = min(next, state.Tiers)
	state.Reward = dailyStreakReward(state.Tier)
	return state, nil
}
