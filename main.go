package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"transcendance/config"
	"transcendance/localization"
	"transcendance/routes"
	"transcendance/views"
)

func run(db *gorm.DB) *gin.Engine {
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


func main() {
db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatal(err)
	}

	router := run(db)

	log.Println("Server running on http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
