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

type Renderer struct {
	templates *template.Template
	Localizer *localization.Localizer
	themer    *Themer
}

func (r *Renderer) T(c *gin.Context, id string, kv ...any) string {
	loc := r.Localizer.Localization(c)
	return r.Localizer.Translate(loc, id, kv...)
}

type View struct {
	Data  any
	Theme string
	// Negotiated locale, e.g. "en", for the page's lang attribute
	Lang string
	// Authenticated user, nil for visitors (shown in the header)
	User any
	// Request path, to highlight the current page in the header
	Path     string
	Children map[string]template.HTML
	Translator
}

type Translator struct {
	localizer *localization.Localizer
	loc       *fluentloc.Localization
}

func (t Translator) T(id string, kv ...any) string {
	return t.localizer.Translate(t.loc, id, kv...)
}

func NewRenderer(l *localization.Localizer, themer *Themer) *Renderer {
	t := template.Must(
		template.New("").Funcs(template.FuncMap{
			"sub": func(a, b int) int { return a - b },
			"add": func(a, b int) int { return a + b },
			"today": func() string {
				return LatestToday().Format("2006-01-02")
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

// RenderToString renders fragments (like cards) to embed in JSON responses.
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

// Add adds a child template, available to the parent as name, and returns it
// so it can get children of its own
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

// AddPaginator adds the page links as "Paginator", and a copy for the bottom of
// lists as "PaginatorBottom": landmarks repeated on a page need distinct names
func (b *pageBuilder) AddPaginator(current int, total int, url string) {
	for name, bottom := range map[string]bool{"Paginator": false, "PaginatorBottom": true} {
		b.Add("paginator", name, map[string]any{
			"Current": current,
			"Total":   total,
			"Url":     url,
			"Bottom":  bottom,
		})
	}
}

type pageContext struct {
	theme string
	user  any
	path  string
	lang  string
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
		Lang:     ctx.lang,
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
		lang:  r.Localizer.GetLocale(c),
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

// Current date in the timezone furthest ahead (UTC+14), so no user's today is in the future
func LatestToday() time.Time {
	now := time.Now().UTC().Add(14 * time.Hour)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
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
		data.Title = r.T(c, "error-title")
		if data.StatusCode != 0 {
			data.Title += " " + strconv.Itoa(data.StatusCode)
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
