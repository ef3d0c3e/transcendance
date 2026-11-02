package tests

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"transcendance/config"
	"transcendance/controllers"
	"transcendance/data"
	"transcendance/localization"
	"transcendance/routes"
	"transcendance/views"
)

var (
	DB      *gorm.DB
	Router  *gin.Engine
	Data    *data.Data
	cleanup func()
)

func Run(data *data.Data, db *gorm.DB) *gin.Engine {
	if err := config.MigrateDatabase(db); err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	l10n, err := localization.New()
	if err != nil {
		log.Fatal(err)
	}
	themer := views.NewThemer()

	router.Use(l10n.Middleware())
	router.Use(controllers.AuthMiddleware(db))
	router.Use(themer.Middleware())

	renderer := views.NewRenderer(l10n, themer)
	routes.RegisterRoutes(router, data, db, renderer)

	return router
}

func Init() error {
	var err error

	// Test accounts need no strong hash, and the default cost makes every
	// registration and login slow under the race detector
	controllers.PasswordHashCost = bcrypt.MinCost

	data, err := data.LoadCards()
	if err != nil {
		return err
	}
	Data = data

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
