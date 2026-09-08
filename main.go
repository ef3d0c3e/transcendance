package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"transcendance/config"
	"transcendance/models"
	"transcendance/routes"
	"transcendance/views"
)

func main() {
	db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatal(err)
	}

	renderer := views.NewRenderer()
	router := gin.Default()
	routes.RegisterRoutes(router, db, renderer)

	log.Println("Server running on http://localhost:8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
