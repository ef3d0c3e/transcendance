package routes

import (
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

	builder := views.PageBuilder("base2", map[string]any{
		"Title": "TEST",
		"User":  user,
	})
	builder.Add("header", "Header", map[string]any{
		"Hello": "foobar",
	})

	p.Renderer.Render(c, &builder)
}
