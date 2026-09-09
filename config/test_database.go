package config

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const testDatabase = "main_test"

func ConnectTestDatabase() (*gorm.DB, func(), error) {
	if err := godotenv.Load(); err != nil {
		return nil, nil, fmt.Errorf("load .env: %w", err)
	}

	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")

	serverDSN := fmt.Sprintf(
		"%s:%s@tcp(127.0.0.1:3306)/?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
	)

	// Connect without selecting a database so we can recreate the test DB.
	serverDB, err := gorm.Open(mysql.Open(serverDSN), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("connect to MariaDB: %w", err)
	}

	if err := serverDB.Exec(
		"DROP DATABASE IF EXISTS `" + testDatabase + "`",
	).Error; err != nil {
		return nil, nil, fmt.Errorf("drop test database: %w", err)
	}

	if err := serverDB.Exec(
		"CREATE DATABASE `" + testDatabase + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
	).Error; err != nil {
		return nil, nil, fmt.Errorf("create test database: %w", err)
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(127.0.0.1:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
		testDatabase,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("connect to test database: %w", err)
	}

	cleanup := func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}

		if sqlDB, err := serverDB.DB(); err == nil {
			_ = sqlDB.Close()
		}

		// Reconnect because serverDB may be closed above.
		admin, err := sql.Open("mysql", serverDSN)
		if err != nil {
			return
		}
		defer admin.Close()

		_, _ = admin.Exec(
			"DROP DATABASE IF EXISTS `" + testDatabase + "`",
		)
	}

	return db, cleanup, nil
}
