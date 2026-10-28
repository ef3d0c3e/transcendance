package views

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	fluentloc "github.com/hakastein/gofluent/localization"

	"transcendance/localization"
)

// Renderer holds parsed templates and localization state
type Renderer struct {
	templates *template.Template
	Localizer *localization.Localizer
	themer    *Themer
}

// T is a method to translate strings
func (r *Renderer) T(c *gin.Context, id string, kv ...any) string {
	loc := r.Localizer.Localization(c)
	return r.Localizer.Translate(loc, id, kv...)
}

// Data passed to templates
type View struct {
	// Parameters
	Data any
	// Theme
	Theme string
	// Authenticated user, nil for visitors (shown in the header)
	User any
	// Request path, to highlight the current page in the header
	Path string
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
func NewRenderer(l *localization.Localizer, themer *Themer) *Renderer {
	t := template.Must(
		template.New("").Funcs(template.FuncMap{
			"sub": func(a, b int) int { return a - b },
			"add": func(a, b int) int { return a + b },
			"today": func() string {
				return time.Now().UTC().Format("2006-01-02")
			},
			"hasPrefix": strings.HasPrefix,
		}).ParseGlob("templates/**/*.html"),
	)

	return &Renderer{
		templates: t,
		Localizer: l,
		themer:    themer,
	}
}

// RenderToString executes a named template and returns the result as a string.
// Useful to embed fragments (like cards) into JSON responses.
func (r *Renderer) RenderToString(name string, data any) (string, error) {
	var content bytes.Buffer

	if err := r.templates.ExecuteTemplate(&content, name, data); err != nil {
		return "", err
	}

	return content.String(), nil
}

type pageBuilder struct {
	templateName string
	data         map[string]any
	children     map[string]*pageBuilder
}

// Create a new page builder
func PageBuilder(templateName string, data map[string]any) pageBuilder {
	if data == nil {
		data = make(map[string]any, 1)
	}
	return pageBuilder{
		templateName: templateName,
		data:         data,
		children:     make(map[string]*pageBuilder, 1),
	}
}

// Add a template (child) to the page builder
// - `templateName` Template name
// - `name` Name that will be available to the parent
// This method returns the added child, so you may add children to that child
func (b *pageBuilder) Add(templateName string, name string, data map[string]any) *pageBuilder {
	if data == nil {
		data = make(map[string]any, 1)
	}
	child := &pageBuilder{
		templateName: templateName,
		data:         data,
		children:     make(map[string]*pageBuilder, 1),
	}
	b.children[name] = child
	return child
}

// Request-wide values given to every template
type pageContext struct {
	theme string
	user  any
	path  string
	loc   *fluentloc.Localization
}

func (r *Renderer) renderPage(ctx *pageContext, b *pageBuilder) (template.HTML, error) {
	children := make(map[string]template.HTML, len(b.children))

	// Render children first (DFS)
	for name, child := range b.children {
		content, err := r.renderPage(ctx, child)
		if err != nil {
			return "", err
		}

		children[name] = content
	}

	view := View{
		Data:     b.data,
		Theme:    ctx.theme,
		User:     ctx.user,
		Path:     ctx.path,
		Children: children,
		Translator: Translator{
			localizer: r.Localizer,
			loc:       ctx.loc,
		},
	}

	var content bytes.Buffer

	if err := r.templates.ExecuteTemplate(&content, b.templateName, view); err != nil {
		return "", err
	}

	return template.HTML(content.String()), nil
}

func (r *Renderer) Render(c *gin.Context, b *pageBuilder) {
	if b == nil {
		log.Println("Rendered page is nil")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to render page"})
		return
	}

	// Set by the auth middleware, a nil *AuthenticatedUser for visitors
	user, _ := c.Get("authenticatedUser")
	ctx := &pageContext{
		theme: r.themer.Theme(c),
		user:  user,
		path:  c.Request.URL.Path,
		loc:   r.Localizer.Localization(c),
	}

	content, err := r.renderPage(ctx, b)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	_, err = c.Writer.Write([]byte(content))
	if err != nil {
		log.Println("Failed to write bytes to client")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to render page"})
		return
	}
}

type ErrorPage struct {
	User       any
	Title      string
	Message    string
	StatusCode int
}

func (r *Renderer) RenderError(
	c *gin.Context,
	data ErrorPage,
) {
	if data.StatusCode != 0 {
		c.Status(data.StatusCode)
	}
	if data.Title == "" {
		if data.StatusCode != 0 {
			data.Title = strconv.Itoa(data.StatusCode) + " " + http.StatusText(data.StatusCode)
		} else {
			data.Title = r.T(c, "Error")
		}
	}
	builder := PageBuilder("base", gin.H{
		"Title": data.Title,
		"User":  data.User,
	})
	builder.Add("error", "Content", gin.H{
		"StatusCode":   data.StatusCode,
		"ErrorMessage": data.Message,
	})
	r.Render(c, &builder)
}
