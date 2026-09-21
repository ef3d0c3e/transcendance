package controllers

import (
	"net/http"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/datatypes"
	"encoding/json"
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

// Only for testing purposes to add notifications in real time
func (ac *AuthController) GetANotifGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	userJson, _ := json.Marshal(user)
	c.String(http.StatusOK, string(userJson))
	var data datatypes.JSONMap
	err := data.UnmarshalJSON(userJson)
	if (err != nil) {
		c.String(http.StatusInternalServerError, err.Error())
	}
	c.JSON(http.StatusOK, gin.H{"user": data})
	notif := models.Notif{
		UserID: user.ID,
		EmmiterID: user.ID,
		Icon: "",
		Type: "FriendRequest",
		Data: data,
		Status: "unread",
		Action: "/friendRequest",
	}
	if err := ac.DB.Create(&notif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, err.Error()),
		})
		return
	}
}

func (ac *AuthController) NotificationGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	notifID := c.Param("ID")
	act := c.Param("action")
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
			c.Redirect(http.StatusFound, notif.Action + act)
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
			Where("user_id = ? and status = ?", user.ID, "unread").
			Order("created_at DESC").
			Find(ctx)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.JSON(http.StatusOK, gin.H{"notifications": notifs})
		}
	}
}
