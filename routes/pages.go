package routes

import (
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Pages struct {
	DB       *gorm.DB
	Renderer *views.Renderer
}

func (p *Pages) IndexGet(c *gin.Context) {
	builder := views.PageBuilder("base", map[string]any{
		"Title": "Cards",
	})
	builder.Add("cards", "Content", map[string]any{
	})
	p.Renderer.Render(c, &builder)
}
