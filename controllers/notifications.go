package controllers

import (
	"net/http"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (ac *AuthController) NotificationGet(c *gin.Context) {
	user := GetAuthenticatedUser(c);
	builder := views.PageBuilder("base", map[string]any{
		"Title": "notifications",
		"User":  user,
	})
	builder.Add("notifications", "Content", map[string]any{
	})
	ac.Renderer.Render(c, &builder)
}

func (ac *AuthController) GetANotifGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	notif := models.Notif{
		UserID: user.ID,
		Icon: "",
		Title: "Free Notification",
		Description: "You got a Notification and it cost you nothing!",
		Status: "unread",
		Action: "/logout",
	}
	if err := ac.DB.Create(&notif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "login-error-internal"),
		})
		return
	}
}

func (ac *AuthController) ApiNotifGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusOK, gin.H{
			"notifications": []map[string]any{
			},
		})
	} else {
	ctx := c.Request.Context()
	notifs, err := gorm.G[models.Notif](ac.DB).
		Where("user_id = ?", user.ID).
		Order("created_at DESC").
		Find(ctx)
	if err != nil {
	} else {
		c.JSON(http.StatusOK, gin.H{"notifications": notifs})
	}
	/*
		c.JSON(http.StatusOK, gin.H{
			"notificationz": []map[string]any{
				{
					"icon": "",
					"title": "test greg",
					"description": "bah c'est une notif",
					"date": "2026-09-11",
					"status": "unread",
				},
				{
					"icon": "",
					"title": "test greg 2",
					"description": "bah c'est une autre notif",
					"date": "1970-09-12",
					"status": "read",
				},
			},
		})
	*/
	}
}
