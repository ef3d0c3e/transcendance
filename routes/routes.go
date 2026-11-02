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
		Logins:   controllers.NewLoginLimits(),
	}

	// Before the routes, so every route gets them
	router.Use(controllers.SecurityHeaders())
	router.Use(controllers.SoftErrors())
	// Debug tools are for admins, and never part of a release build
	debug := gin.Mode() != gin.ReleaseMode

	// Static data
	router.Static("/frontend", "./frontend/dist")
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")

	// seo.go
	router.GET("/robots.txt", controller.RobotsGet)
	router.GET("/sitemap.xml", controller.SitemapGet)

	// health.go
	router.GET("/healthz", controller.HealthzGet)

	// errors.go
	router.NoRoute(controller.NotFound)

	// auth.go
	router.GET("/register", controller.RegisterGet)
	router.POST("/register", controller.RegisterPost)
	router.GET("/login", controller.LoginGet)
	router.POST("/login", controller.LoginPost)
	router.POST("/api/logout", controller.LogoutPost)

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

	// admin.go
	router.GET("/admin/users", controller.AdminUsersGet)
	router.POST("/admin/users/:id/rank", controller.AdminUserRankPost)
	router.POST("/admin/users/:id/password", controller.AdminUserPasswordPost)
	router.POST("/admin/users/:id/delete", controller.AdminUserDeletePost)
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

	// Debug: send a notification to the authenticated admin
	if debug {
		router.GET("/add_notification", controller.AddNotificationGet)
		router.POST("/add_notification", controller.AddNotificationPost)
	}

	// shoutbox.go
	shoutboxController := controllers.NewShoutboxController(db, renderer)
	router.GET("/api/shoutbox", shoutboxController.ApiShoutboxGet)
	//friends
	router.POST("/friendaction", controller.FriendsPost)
	router.GET("/friends", controller.FriendsGet)

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
	router.POST("/inventory/claim", dataController.InventoryClaimPost)
	// Debug: add a card to the authenticated admin's inventory
	if debug {
		router.POST("/add", dataController.InventoryAddPost)
	}

	router.GET("/opening", dataController.OpeningGet)
	// Without JavaScript, the opened cards are shown in the inventory, newest first
	openedTarget, openFailedTarget := "/inventory?tab=cards&sort=recent", "/inventory?tab=boosters&error=opening"
	router.POST("/opening", controllers.FormFallback(dataController.OpeningPost, openedTarget, openFailedTarget))
	router.POST("/opening/buy", controllers.FormFallback(dataController.OpeningBuyPost, openedTarget, openFailedTarget))

	// tradeup.go
	router.POST("/tradeup", dataController.TradeupPost)
	// foil.go
	router.POST("/foil", dataController.FoilRoulettePost)

	// Debug: add a booster to the authenticated admin's inventory
	if debug {
		router.GET("/add_booster", dataController.AddBoosterGet)
		router.POST("/add_booster", dataController.AddBoosterPost)
	}

	// trade.go
	router.GET("/trades", controller.TradesGet)
	router.GET("/trades/new", controller.TradeMenuGet)
	router.POST("/trades", controller.TradePost)
	router.POST("/trades/:ID/accept", controller.TradeAcceptPost)
	router.POST("/trades/:ID/decline", controller.TradeDeclinePost)
	router.POST("/trades/:ID/cancel", controller.TradeCancelPost)
	router.GET("/api/trade/stacks", dataController.TradeStacksGet)

	// shop.go
	router.GET("/shop", controller.ShopGet)
	router.POST("/shop/topup", controller.ShopTopupPost)
	router.POST("/shop/buy", controller.ShopBuyPost)
	router.POST("/shop/sell", controller.ShopSellPost)
	router.POST("/shop/sell-duplicates", controller.ShopSellDuplicatesPost)
}
