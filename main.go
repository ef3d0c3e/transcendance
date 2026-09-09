package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"transcendance/config"
	"transcendance/models"
	"transcendance/routes"
	"transcendance/views"
	"transcendance/localization"
)

func main() {
	db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	l10n, err := localization.New()
	if err != nil {
		log.Fatal(err)
	}
	// Register localization middleware on all routes
	router.Use(l10n.Middleware());

	renderer := views.NewRenderer(l10n)
	routes.RegisterRoutes(router, db, renderer)

	log.Println("Server running on http://localhost:8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
