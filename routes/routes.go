package routes

import (
	"transcendance/controllers"
	"transcendance/data"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)


func RegisterRoutes(
	router *gin.Engine,
	data *data.Data,
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

	router.GET("/api/logout", authController.LogoutGet)

	accountController := controllers.AccountController{
		DB:       db,
		Renderer: renderer,
	}
	router.GET("/api/account_delete", accountController.AccountDeleteGet)

	pages := Pages{
		DB:       db,
		Renderer: renderer,
	}
	router.GET("/", pages.IndexGet)
	router.GET("/terms", pages.TosGet)
	router.GET("/contact", pages.ContactGet)
	router.GET("/privacy", pages.PrivacyGet)

	searchController := controllers.SearchController{
		DB:       db,
		Renderer: renderer,
	}
	router.GET("/search", searchController.SearchUsersGet)

	// Data
	dataController := controllers.DataController{
		DB:       db,
		Renderer: renderer,
		Data: data,
	}
	router.GET("/cards/:FIRST/:SECOND", dataController.CardGet)
	router.GET("/collections/:FIRST", dataController.CollectionGet)

	router.GET("/search_cards", dataController.CardSearchGet)
}
