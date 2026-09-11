package views

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	fluentloc "github.com/hakastein/gofluent/localization"

	"transcendance/localization"
)

// Hold data for template rendering
type Renderer struct {
	templates *template.Template
	localizer *localization.Localizer
}

// T is a method to translate strings
func (r *Renderer) T(c *gin.Context, id string, kv ...any) string {
	loc := r.localizer.Localization(c)
	return r.localizer.Translate(loc, id, kv...)
}

// Data passed to templates
type View struct {
	Data any
	Translator
}

// Translator available in templates
type Translator struct {
	localizer *localization.Localizer
	loc       *fluentloc.Localization
}

// Translate string, available in templates
func (t Translator) T(id string, kv ...any) string {
	return t.localizer.Translate(t.loc, id, kv...)
}

// NewRenderer: parse all templates under templates/**/*.html and return a Renderer
func NewRenderer(l *localization.Localizer) *Renderer {
	t := template.Must(
		template.New("").ParseGlob("templates/**/*.html"),
	)

	return &Renderer{
		templates: t,
		localizer: l,
	}
}

// FIXME: This always use the base.html layout.
func (r *Renderer) Render(c *gin.Context, page string, data any) {
	// Build view for user
	loc := r.localizer.Localization(c)
	view := View{
		Data: data,
			localizer: r.localizer,
			loc:       loc,
	}

	var content bytes.Buffer
	if err := r.templates.ExecuteTemplate(&content, page, view); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Set the data for the outer/base template
	view.Data = struct {
		Title   string
		Content template.HTML
	}{
		Title:   data.(map[string]any)["Title"].(string),
		Content: template.HTML(content.String()),
	}

	if err := r.templates.ExecuteTemplate(c.Writer, "base", view); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
}
