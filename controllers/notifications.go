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
	dataJson, _ := json.Marshal(gin.H{"UserID": 555, "Username": "foobar"})
	var data datatypes.JSONMap
	if err := data.UnmarshalJSON(dataJson); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
	notif := models.Notif{
		UserID: user.ID,
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
			Limit(15).
			Find(ctx)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			// Only expose what's required + expose localized data
			result := make([]map[string]any, 0)
			for _, notif := range notifs {
				data := make(map[string]any)
				data["ID"] = notif.ID
				data["Type"] = notif.Type
				data["Status"] = notif.Status
				data["Data"] = notif.Data
				data["CreatedAt"] = notif.CreatedAt
				data["Icon"] = notif.Icon
				data["Action"] = notif.Action
				locale := make(map[string]string)
				if notif.Type == "FriendRequest" {
					locale["Title"] = ac.Renderer.T(c, "notification-friend-request-title")
					locale["Desc"] = ac.Renderer.T(c, "notification-friend-request-desc", "username", notif.Data["Username"])
					locale["Accept"] = ac.Renderer.T(c, "notification-friend-request-accept")
					locale["Deny"] = ac.Renderer.T(c, "notification-friend-request-deny")
				}
				data["Locale"] = locale
				result = append(result, data)
			}
			c.JSON(http.StatusOK, gin.H{"notifications": result})
		}
	}
}
