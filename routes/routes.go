package routes

import (
	"transcendance/controllers"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)


func RegisterRoutes(
	router *gin.Engine,
	db *gorm.DB,
	renderer *views.Renderer,
) {
	authController := controllers.AuthController{
		DB:       db,
		Renderer: renderer,
	}

	router.Static("/frontend", "./frontend/dist")
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")

	router.GET("/register", authController.RegisterGet)
	router.POST("/register", authController.RegisterPost)

	router.GET("/login", authController.LoginGet)
	router.POST("/login", authController.LoginPost)
}
