package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"transcendance/models"
)

func ConnectDatabase() (*gorm.DB, error) {
	// Containers get their environment from compose instead of a .env file
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	database := os.Getenv("MYSQL_DATABASE")
	address := os.Getenv("MYSQL_ADDRESS")
	if address == "" {
		address = "127.0.0.1:3306"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
		address,
		database,
	)

	gormLogger := logger.New(
		slogWriter{},
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
		},
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	// Bounded, so a burst of requests waits for a connection instead of exhausting the database's
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

// slogWriter logs GORM's slow queries and errors as one warning line each
type slogWriter struct{}

func (slogWriter) Printf(format string, args ...any) {
	slog.Warn(strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func MigrateDatabase(db *gorm.DB) error {
	// Early accounts have no email: add the column with unique placeholders first, so
	// the unique index can be built below (AutoMigrate would fail on the empty duplicates)
	if db.Migrator().HasTable(&models.User{}) {
		if !db.Migrator().HasColumn(&models.User{}, "Email") {
			if err := db.Exec("ALTER TABLE users ADD COLUMN email varchar(254) NOT NULL").Error; err != nil {
				return err
			}
		}
		if err := db.Exec(`UPDATE users SET email = CONCAT('user_', id, '@example.invalid') WHERE email = ''`).Error; err != nil {
			return err
		}
	}
	// Sessions are stored hashed since their token got an index: the older plain tokens can never match, and
	// expired sessions are useless
	if db.Migrator().HasTable(&models.UserSession{}) {
		if err := db.Exec("DELETE FROM user_sessions WHERE CHAR_LENGTH(token) <> 64 OR expires_at <= NOW()").Error; err != nil {
			return err
		}
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserSession{}, &models.UserProfile{}, &models.Relation{}, &models.Notification{}, &models.ShoutboxMessage{}, &models.PrivateMessage{}, &models.UserIgnore{}, &models.ShoutboxMute{}, &models.Card{}, &models.Booster{}, &models.BoosterPile{}, &models.DailyStreak{}, &models.Trade{}, &models.TradeLine{}); err != nil {
		return err
	}
	// The stacks of offers were JSON lists in the trades table before they got their own
	if db.Migrator().HasColumn(&models.Trade{}, "give") {
		if err := migrateTradeItems(db); err != nil {
			return fmt.Errorf("move trade stacks to trade_items: %w", err)
		}
	}
	// Foil values compare byte for byte, see models.FoilColumnType: older tables have the default collation
	for _, table := range []string{"cards", "trade_items"} {
		var collation string
		if err := db.Raw(`SELECT COLLATION_NAME FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = 'foil'`, table).
			Scan(&collation).Error; err != nil {
			return err
		}
		if collation != "utf8mb4_bin" {
			if err := db.Exec("ALTER TABLE `" + table + "` MODIFY foil " + models.FoilColumnType + " NOT NULL DEFAULT ''").Error; err != nil {
				return err
			}
		}
	}
	// Erasing users that were soft deleted before account deletion became permanent
	if err := db.Exec("DELETE FROM users WHERE deleted_at IS NOT NULL").Error; err != nil {
		return err
	}
	// Updating user entries that were created before user profile was introduced
	return db.Exec(`
		INSERT INTO user_profiles (user_id)
		SELECT users.id FROM users
		LEFT JOIN user_profiles ON user_profiles.user_id = users.id
		WHERE users.deleted_at IS NULL AND user_profiles.user_id IS NULL
	`).Error
}

// Copy the stacks of every offer from the former JSON columns of trades to trade_items, then drop the columns
func migrateTradeItems(db *gorm.DB) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		var rows []struct {
			ID   uint
			Give []byte
			Want []byte
		}
		if err := tx.Table("trades").Select("id, give, want").Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			sides := make([][]models.TradeItem, 2)
			for i, raw := range [][]byte{row.Give, row.Want} {
				var items []models.TradeItem
				if len(raw) > 0 {
					if err := json.Unmarshal(raw, &items); err != nil {
						return fmt.Errorf("trade %d: %w", row.ID, err)
					}
				}
				sides[i] = mergeTradeItems(items)
			}
			if err := models.SaveTradeItems(tx, row.ID, sides[0], sides[1]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Outside of the transaction: MariaDB commits before changing a table
	for _, column := range []string{"give", "want"} {
		if err := db.Migrator().DropColumn(&models.Trade{}, column); err != nil {
			return err
		}
	}
	return nil
}

// One item per stack, adding up the counts of repeated ones
func mergeTradeItems(items []models.TradeItem) []models.TradeItem {
	merged := make([]models.TradeItem, 0, len(items))
	index := make(map[models.TradeItem]int, len(items))
	for _, item := range items {
		key := models.TradeItem{CardID: item.CardID, Foil: item.Foil}
		if i, ok := index[key]; ok {
			merged[i].Count += item.Count
			continue
		}
		index[key] = len(merged)
		merged = append(merged, item)
	}
	return merged
}
