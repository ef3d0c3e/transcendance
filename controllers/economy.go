package controllers

import (
	"errors"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"transcendance/models"
)

// ErrNoPileBooster is returned when the pile holds no default booster.
var ErrNoPileBooster = errors.New("no default booster available")

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

// creditDailyReward credits the connection streak reward of the day to a
// user's balance, once per day, growing their streak. A missed day resets
// the streak to day 1, which caps out at the last reward day.
// `user.Coins` is updated to the new balance.
func creditDailyReward(db *gorm.DB, user *AuthenticatedUser, now time.Time) error {
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
			// The day is already credited
			return nil
		case last.Equal(yesterday):
			// The streak counter caps out, but keeps paying the last reward
			day = min(streak.Day+1, uint(len(dailyStreakRewards)))
		}
		// A wider gap resets the streak to day 1
	}

	reward := dailyStreakReward(day)
	return db.Transaction(func(tx *gorm.DB) error {
		// Credit only when no request credited this day yet, so two visits
		// the same day cannot pay it twice
		result := tx.Model(&models.DailyStreak{}).
			Where("user_id = ? AND (last_day IS NULL OR last_day < ?)", user.ID, today).
			Updates(map[string]any{"day": day, "last_day": today})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		if err := tx.Model(&models.User{}).
			Where("id = ?", user.ID).
			Update("coins", gorm.Expr("coins + ?", reward)).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.Notification{
			UserID: user.ID,
			Type:   models.NotificationDailyReward,
			Data:   datatypes.JSONMap{"amount": int(reward), "day": int(day)},
		}).Error; err != nil {
			return err
		}

		user.Coins += reward
		return nil
	})
}

// DailyRewardMiddleware credits the connection streak reward of the day,
// once per day, on the first request of the day (server time).
func DailyRewardMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if user := GetAuthenticatedUser(c); user != nil {
			if err := creditDailyReward(db, user, time.Now()); err != nil {
				log.Printf("WARN failed to credit the daily reward of user %d: %s", user.ID, err)
			}
		}
		c.Next()
	}
}
