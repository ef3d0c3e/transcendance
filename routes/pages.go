package routes

import (
	"net/http"
	"transcendance/controllers"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Pages struct {
	DB       *gorm.DB
	Renderer *views.Renderer
}

func (p *Pages) IndexGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	_ = user

	builder := views.PageBuilder("base2", map[string]any {
		"Title": "TEST",
	})
	builder.Add("header", "Header", map[string]any {
		"Hello": "foobar",
	})

	p.Renderer.Render(c, &builder)
}

func (p* Pages) NotificationApiGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"notifications": []map[string]any{
			{
				"title": "foo",
				"description": "...",
				"date": "2026-09-15T23:40:00Z",
				"icon": "data/svg:...",
				"status": "unread",
			},
			{
				"title": "bar",
				"description": "...",
				"date": "2026-09-15T23:20:00Z",
				"icon": "data/svg:...",
				"status": "read",
			},
		},
	})
}

func (p* Pages) NotificationGet(c *gin.Context) {
	builder := views.PageBuilder("base", map[string]any{
		"Title": "Notifications",
	})
	builder.Add("notifications", "Content", map[string]any {})
	p.Renderer.Render(c, &builder)
}
