package routes

import (
	"transcendance/controllers"
	ctrldata "transcendance/controllers/data"
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

	searchController := controllers.SearchController{
		DB:       db,
		Renderer: renderer,
	}
	router.GET("/search", searchController.SearchUsersGet)

	// Data
	dataController := ctrldata.DataController{
		DB:       db,
		Renderer: renderer,
		Data:     data,
	}
	router.GET("/cards/:FIRST/*SECOND", dataController.CardsGet)
	router.GET("/collections/:FIRST/*SECOND", dataController.CollectionsGet)

	router.GET("/inventory", dataController.InventoryGet)
	router.GET("/add", dataController.InventoryAddGet)

	router.GET("/opening", dataController.OpeningGet)
	router.POST("/opening", dataController.OpeningPost)

	// Debug: add a booster to the authenticated user's inventory
	router.GET("/add_booster", dataController.AddBoosterGet)
	router.POST("/add_booster", dataController.AddBoosterPost)

	// Shop
	shopController := controllers.ShopController{
		DB:       db,
		Renderer: renderer,
		Data:     data,
	}
	router.GET("/shop", shopController.ShopGet)
	router.POST("/shop/topup", shopController.ShopTopupPost)
	router.POST("/shop/buy", shopController.ShopBuyPost)
	router.POST("/shop/sell", shopController.ShopSellPost)
}
