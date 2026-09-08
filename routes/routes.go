package routes

import (
	"bytes"
	"html/template"
	"net/http"
	"transcendance/controllers"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var templates = template.Must(
	template.ParseGlob("templates/**/*.html"),
)

// Render template `page` using the base template.
func renderPage(c *gin.Context, page string, data any) {
	var content bytes.Buffer

	err := templates.ExecuteTemplate(&content, page, data)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	viewData := map[string]any{
		"Title":   data.(map[string]any)["Title"],
		"Content": template.HTML(content.String()),
	}

	err = templates.ExecuteTemplate(c.Writer, "base", viewData)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
}


func RegisterRoutes(
	router *gin.Engine,
	db *gorm.DB,
	renderer *views.Renderer,
) {
	authController := controllers.AuthController{
		DB:       db,
		Renderer: renderer,
	}

	router.GET("/register", authController.ShowRegister)
	router.POST("/register", authController.Register)
}
