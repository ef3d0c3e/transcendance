package config

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const testDatabase = "main_test"

func ConnectTestDatabase() (*gorm.DB, func(), error) {
	// The environment may come from CI instead of a .env file
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, fmt.Errorf("load .env: %w", err)
	}

	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	address := os.Getenv("MYSQL_ADDRESS")
	if address == "" {
		address = "127.0.0.1:3306"
	}

	serverDSN := fmt.Sprintf(
		"%s:%s@tcp(%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
		address,
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
		"%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
		address,
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

		_, _ = admin.Exec(
			"DROP DATABASE IF EXISTS `" + testDatabase + "`",
		)
		err = admin.Close()
		if err != nil {
			log.Fatal(err)
		}
	}

	return db, cleanup, nil
}
