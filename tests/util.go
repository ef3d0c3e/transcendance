package tests

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"transcendance/config"
	"transcendance/localization"
	"transcendance/routes"
	"transcendance/views"
)

var (
	DB     *gorm.DB
	Router *gin.Engine
	cleanup func()
)

func Run(db *gorm.DB) *gin.Engine {
	if err := config.MigrateDatabase(db); err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	l10n, err := localization.New()
	if err != nil {
		log.Fatal(err)
	}

	router.Use(l10n.Middleware())

	renderer := views.NewRenderer(l10n)
	routes.RegisterRoutes(router, db, renderer)

	return router
}

func Init() error {
	var err error

	DB, cleanup, err = config.ConnectTestDatabase()
	if err != nil {
		return err
	}

	Router = gin.Default()

	if err := config.MigrateDatabase(DB); err != nil {
		cleanup()
		return fmt.Errorf("migrate test database: %w", err)
	}

	return nil
}

func Cleanup() {
	if cleanup != nil {
		cleanup()
	}
}
