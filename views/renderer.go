package views

import (
	"bytes"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	fluentloc "github.com/hakastein/gofluent/localization"

	"transcendance/localization"
)

// Renderer holds parsed templates and localization state
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
	// Parameters
	Data any
	// Child templates
	Children map[string]template.HTML
	// Translator instance
	Translator
}

// Translator that is given to templates
type Translator struct {
	localizer *localization.Localizer
	loc       *fluentloc.Localization
}

// Translate string, available in templates
func (t Translator) T(id string, kv ...any) string {
	return t.localizer.Translate(t.loc, id, kv...)
}

// NewRenderer: parse all templates under templates/**/*.html
func NewRenderer(l *localization.Localizer) *Renderer {
	t := template.Must(
		template.New("").ParseGlob("templates/**/*.html"),
	)

	return &Renderer{
		templates: t,
		localizer: l,
	}
}

type page_builder struct {
	template_name string
	data          map[string]any
	children      map[string]page_builder
}

// Create a new page builder
func PageBuilder(template_name string, data map[string]any) page_builder {
	return page_builder{
		template_name: template_name,
		data:          data,
		children:      make(map[string]page_builder, 1),
	}
}

// Add a template (child) to the page builder
// - `template_name` Template name
// - `name` Name that will be available to the parent
// This method returns the added child, so you may add children to that child
func (b *page_builder) Add(template_name string, name string, data map[string]any) page_builder {
	child := PageBuilder(template_name, data)
	b.children[name] = child
	return child
}

func (r *Renderer) renderPage(
	loc *fluentloc.Localization,
	b *page_builder,
) (template.HTML, error) {
	children := make(map[string]template.HTML, len(b.children))

	// Render children first (DFS)
	for name, child := range b.children {
		content, err := r.renderPage(loc, &child)
		if err != nil {
			return "", err
		}

		children[name] = content
		b.data[name] = content
	}

	view := View{
		Data:     b.data,
		Children: children,
		Translator: Translator{
			localizer: r.localizer,
			loc:       loc,
		},
	}

	var content bytes.Buffer

	if err := r.templates.ExecuteTemplate(&content, b.template_name, view); err != nil {
		return "", err
	}

	return template.HTML(content.String()), nil
}

func (r *Renderer) Render2(c *gin.Context, b *page_builder) {
	if b == nil {
		log.Println("Rendered page is nil")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to render page"})
		return
	}

	loc := r.localizer.Localization(c)

	content, err := r.renderPage(loc, b)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Writer.Write([]byte(content))
}
