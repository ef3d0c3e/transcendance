package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
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
		log.New(os.Stdout, "\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func MigrateDatabase(db *gorm.DB) error {
	// User emails were introduced after early accounts were registered: add the
	// column and fill it with unique placeholders first, so the unique index on
	// it can be built below (AutoMigrate would fail on the empty duplicates)
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
	if err := db.AutoMigrate(&models.User{}, &models.UserSession{}, &models.UserProfile{}, &models.Notification{}, &models.Card{}, &models.Booster{}, &models.BoosterPile{}, &models.DailyStreak{}, &models.Trade{}); err != nil {
		return err
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
