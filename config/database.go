package config

import (
	"fmt"
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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return nil, err
	}

	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	database := os.Getenv("MYSQL_DATABASE")

	dsn := fmt.Sprintf(
		"%s:%s@tcp(127.0.0.1:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
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
	return db.AutoMigrate(&models.User{}, &models.UserSession{})
}
