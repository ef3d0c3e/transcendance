package controllers

import (
	"net/http"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (ac *AuthController) NotificationsGet(c *gin.Context) {
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

func (ac *AuthController) NotificationGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	notifID := c.Param("ID")
	if user != nil {
		ctx := c.Request.Context()
		_, err := gorm.G[models.Notif](ac.DB).
			Where("id = ? and user_id = ?", notifID, user.ID).
			Update(ctx, "Status", "read")
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		}
		notif, err := gorm.G[models.Notif](ac.DB).
			Where("id = ? and user_id = ?", notifID, user.ID).
			First(ctx)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.Redirect(http.StatusFound, notif.Action)
		}
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
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.JSON(http.StatusOK, gin.H{"notifications": notifs})
		}
	}
}
