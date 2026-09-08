package views

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Renderer struct {
	templates *template.Template
}

func NewRenderer() *Renderer {
	t := template.Must(
		template.ParseGlob("templates/**/*.html"),
	)

	return &Renderer{
		templates: t,
	}
}

func (r *Renderer) Render(
	c *gin.Context,
	page string,
	data any,
) {
	var content bytes.Buffer

	if err := r.templates.ExecuteTemplate(&content, page, data); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	view := struct {
		Title   string
		Content template.HTML
	}{
		Content: template.HTML(content.String()),
	}

	if err := r.templates.ExecuteTemplate(c.Writer, "base", view); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
}
