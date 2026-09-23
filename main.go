package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"transcendance/config"
	"transcendance/controllers"
	"transcendance/data"
	"transcendance/localization"
	"transcendance/routes"
	"transcendance/views"
)

func run(data *data.Data, db *gorm.DB) *gin.Engine {
	if err := config.MigrateDatabase(db); err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	l10n, err := localization.New()
	if err != nil {
		log.Fatal(err)
	}

	router.Use(l10n.Middleware())
	router.Use(controllers.AuthMiddleware(db))

	renderer := views.NewRenderer(l10n)
	routes.RegisterRoutes(router, data, db, renderer)

	return router
}

func main() {
	data, err := data.LoadCards()
	if err != nil {
		log.Fatal(err)
		return
	}

	db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatal(err)
	}

	router := run(data, db)

	log.Println("Server running on http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
