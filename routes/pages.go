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
	_ = user

	builder := views.PageBuilder("base", map[string]any {
		"Title": "CHANGE ME", //use project final name instead
	})
	builder.Add("index", "Content", map[string]any {
		"User": user,
	})

	p.Renderer.Render(c, &builder)
}

func (p *Pages) TosGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	_ = user

	builder := views.PageBuilder("base", map[string]any {
		"Title": "Terms of service",
	})
	builder.Add("tos", "Content", map[string]any {
	})

	p.Renderer.Render(c, &builder)
}

func (p *Pages) ContactGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	_ = user

	builder := views.PageBuilder("base", map[string]any {
		"Title": "Contact",
	})
	builder.Add("contact", "Content", map[string]any {
	})

	p.Renderer.Render(c, &builder)
}

func (p *Pages) PrivacyGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	_ = user

	builder := views.PageBuilder("base", map[string]any {
		"Title": "Privacy",
	})
	builder.Add("privacy", "Content", map[string]any {
	})

	p.Renderer.Render(c, &builder)
}
