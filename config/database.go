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
	if err := db.AutoMigrate(&models.User{}, &models.UserSession{}, &models.Card{}, &models.Booster{}); err != nil {
		return err
	}
	return nil
}
