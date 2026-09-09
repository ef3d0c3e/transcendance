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

	router.Static("/static/frontend", "./frontend/dist")

	router.GET("/register", authController.ShowRegister)
	router.POST("/register", authController.Register)
}
