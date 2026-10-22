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

	// Notifications
	notificationController := controllers.NotificationController{
		DB:       db,
		Renderer: renderer,
	}
	router.GET("/notifications", notificationController.NotificationsGet)
	router.GET("/notification/:ID", notificationController.NotificationGet)
	router.GET("/api/notifications", notificationController.ApiNotificationsGet)
	router.POST("/api/notifications/read_all", notificationController.ApiNotificationsReadAllPost)
	router.POST("/api/notifications/:ID/read", notificationController.ApiNotificationReadPost)
	router.POST("/api/notifications/:ID/unread", notificationController.ApiNotificationUnreadPost)
	router.POST("/api/notifications/:ID/delete", notificationController.ApiNotificationDeletePost)

	// Debug: send a notification to the authenticated user
	router.GET("/add_notification", notificationController.AddNotificationGet)
	router.POST("/add_notification", notificationController.AddNotificationPost)
}
