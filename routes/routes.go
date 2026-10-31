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
	controller := controllers.Controller{
		DB:       db,
		Renderer: renderer,
		Data:     data,
	}

	// Static data
	router.Static("/frontend", "./frontend/dist")
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")

	// health.go
	router.GET("/healthz", controller.HealthzGet)

	// errors.go
	router.NoRoute(controller.NotFound)

	// auth.go
	router.GET("/register", controller.RegisterGet)
	router.POST("/register", controller.RegisterPost)
	router.GET("/login", controller.LoginGet)
	router.POST("/login", controller.LoginPost)
	router.GET("/api/logout", controller.LogoutGet)

	// profile.go
	router.GET("/profile", controller.ProfileGet)
	router.GET("/profile/:username", controller.ProfileGet)
	router.POST("/profile", controller.ProfilePost)
	router.POST("/profile/avatar", controller.AvatarPost)
	router.POST("/profile/avatar/delete", controller.AvatarDelete)
	router.POST("/profiles/:id", controller.ProfilePost)
	router.POST("/profiles/:id/delete", controller.ProfileDeletePost)
	router.POST("/profiles/:id/avatar", controller.AvatarPost)
	router.POST("/profiles/:id/avatar/delete", controller.AvatarDelete)
	router.GET("/avatars/:id", controller.AvatarGet)
	router.HEAD("/avatars/:id", controller.AvatarGet)

	// legal.go
	router.GET("/terms", controller.TermsGet)
	router.GET("/legal", controller.LegalGet)
	router.GET("/contact", controller.ContactGet)

	// user_search.go
	router.GET("/search", controller.SearchUsersGet)

	// notifications.go
	router.GET("/notifications", controller.NotificationsGet)
	router.GET("/notification/:ID", controller.NotificationGet)
	router.GET("/api/notifications", controller.ApiNotificationsGet)
	router.POST("/api/notifications/read_all", controller.ApiNotificationsReadAllPost)
	router.POST("/api/notifications/:ID/read", controller.ApiNotificationReadPost)
	router.POST("/api/notifications/:ID/unread", controller.ApiNotificationUnreadPost)
	router.POST("/api/notifications/:ID/delete", controller.ApiNotificationDeletePost)

	// DEBUG: send a notification to the authenticated user
	router.GET("/add_notification", controller.AddNotificationGet)
	router.POST("/add_notification", controller.AddNotificationPost)

	// shoutbox.go
	shoutboxController := controllers.NewShoutboxController(db, renderer)
	router.GET("/api/shoutbox", shoutboxController.ApiShoutboxGet)

	// Data
	dataController := ctrldata.DataController{
		DB:       db,
		Renderer: renderer,
		Data:     data,
	}
	router.GET("/", dataController.IndexGet)
	router.GET("/cards/:FIRST/*SECOND", dataController.CardsGet)
	router.GET("/collections/:FIRST/*SECOND", dataController.CollectionsGet)

	router.GET("/inventory", dataController.InventoryGet)
	router.GET("/add", dataController.InventoryAddGet)

	router.GET("/opening", dataController.OpeningGet)
	router.POST("/opening", dataController.OpeningPost)
	router.POST("/opening/buy", dataController.OpeningBuyPost)

	// Debug: add a booster to the authenticated user's inventory
	router.GET("/add_booster", dataController.AddBoosterGet)
	router.POST("/add_booster", dataController.AddBoosterPost)

	// shop.go
	router.GET("/shop", controller.ShopGet)
	router.POST("/shop/topup", controller.ShopTopupPost)
	router.POST("/shop/buy", controller.ShopBuyPost)
}
