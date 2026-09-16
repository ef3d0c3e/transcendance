package controllers

import (
	"github.com/gin-gonic/gin"
	"transcendance/views"
	"net/http"
)

func (ac *AuthController) NotificationGet(c *gin.Context) {
	user := GetAuthenticatedUser(c);
	builder := views.PageBuilder("base", map[string]any{
		"Title": "notif",
		"User":  user,
	})
	builder.Add("notif", "Content", map[string]any{
	})
	ac.Renderer.Render(c, &builder)
}

func (ac *AuthController) ApiNotifGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"notifications": []map[string]any{
				{
					"icon": "",
					"title": "test greg",
					"description": "bah c'est une notif",
					"date": "2026-09-11",
				},
				{
					"icon": "",
					"title": "test greg 2",
					"description": "bah c'est une autre notif",
					"date": "1970-09-12",
				},
			},
		})
	}
}
