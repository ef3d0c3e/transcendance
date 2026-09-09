package localization

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	fluentloc "github.com/hakastein/gofluent/localization"
	"golang.org/x/text/language"
)

//go:embed locales
var localesFS embed.FS

const localesDir = "locales"

// contextKey is a private type used to store the negotiated Localization
// in the Gin context.
type contextKey string

const localizationContextKey contextKey = "localization.Localizer"

type Localizer struct {
	// Available locales
	Available []string
	// List of individial .ftl files
	Resources []string
	Loader    fluentloc.ResourceLoader

	// Fallback locale: 'en'
	Fallback string
	// Locale GET parameter: 'lang'
	QueryParam string
	// Locale cookie name: 'lang'
	CookieName string

	// Cache of per-user locales
	Cache sync.Map
}

// Build localizer
func New() (*Localizer, error) {
	entries, err := fs.ReadDir(localesFS, localesDir)
	if err != nil {
		return nil, fmt.Errorf("localization: reading %q: %w", localesDir, err)
	}

	var available []string
	for _, e := range entries {
		if e.IsDir() {
			available = append(available, e.Name())
		}
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("localization: no locale directories found under %q", localesDir)
	}

	files, err := fs.ReadDir(localesFS, localesDir+"/"+available[0])
	if err != nil {
		return nil, fmt.Errorf("localization: reading locale %q: %w", available[0], err)
	}

	var resources []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".ftl") {
			resources = append(resources, strings.TrimSuffix(f.Name(), ".ftl"))
		}
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("localization: no .ftl resources found for locale %q", available[0])
	}

	l := &Localizer{
		Available:  available,
		Resources:  resources,
		Fallback:   "en",
		QueryParam: "lang",
		CookieName: "lang",
		Loader:     fluentloc.FSLoader(localesFS, localesDir+"/{locale}/{resource}.ftl"),
	}

	return l, nil
}

// Localization middleware. Negotiate locale based on request parameters and/or cookie
func (l *Localizer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get(localizationContextKey); !exists {
			// Negotiate and store
			loc := l.negotiate(c)
			c.Set(localizationContextKey, loc)

			// Store cookie
			requested := l.requestedLocales(c)
			chosen := l.chosenLocale(requested)
			c.SetCookie(l.CookieName, chosen, 3600*24*30, "/", "", false, true)
		}
		c.Next()
	}
}

// Return the negotiated locale. Locale is negotiated once, then cached
func (l *Localizer) Localization(c *gin.Context) *fluentloc.Localization {
	// Cached?
	if loc, exists := c.Get(localizationContextKey); exists {
		return loc.(*fluentloc.Localization)
	}

	// Negotiate and store
	loc := l.negotiate(c)
	c.Set(localizationContextKey, loc)
	return loc
}

// Translate formats a message using an already-obtained Localization
func (l *Localizer) Translate(loc *fluentloc.Localization, id string, kv ...any) string {
	val, _ := loc.FormatValue(id, argsToMap(kv))
	return val
}

// negotiate builds the requested locales list, using the cache, and returns a Localization
func (l *Localizer) negotiate(c *gin.Context) *fluentloc.Localization {
	requested := l.requestedLocales(c)
	cacheKey := strings.Join(requested, ",")

	if cached, ok := l.Cache.Load(cacheKey); ok {
		return cached.(*fluentloc.Localization)
	}

	loc, err := fluentloc.NewFromLocales(fluentloc.Config{
		Requested: requested,
		Available: l.Available,
		Default:   l.Fallback,
		Resources: l.Resources,
		Loader:    l.Loader,
	})
	if err != nil || loc == nil {
		// This should never happen
		loc = fluentloc.New()
	}

	l.Cache.Store(cacheKey, loc)
	return loc
}

// Attempt to get the requested locale using a fallback chain
func (l *Localizer) chosenLocale(requested []string) string {
	for _, r := range requested {
		for _, a := range l.Available {
			if r == a {
				return a
			}
		}
	}
	return l.Fallback
}

// Detect locale requested by user. Build a list of requested locale in ordered by priority
func (l *Localizer) requestedLocales(c *gin.Context) []string {
	var requested []string

	if l.QueryParam != "" {
		if v := c.Query(l.QueryParam); v != "" {
			requested = append(requested, v)
		}
	}

	if l.CookieName != "" {
		if v, err := c.Cookie(l.CookieName); err == nil && v != "" {
			requested = append(requested, v)
		}
	}

	if header := c.GetHeader("Accept-Language"); header != "" {
		if tags, _, err := language.ParseAcceptLanguage(header); err == nil {
			for _, t := range tags {
				requested = append(requested, t.String())
			}
		}
	}

	return requested
}

// Transforms a flat: "key1", val1, "key2", val2 list into map[string]any
func argsToMap(kv []any) map[string]any {
	if len(kv) == 0 {
		return nil
	}
	args := make(map[string]any, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		args[key] = kv[i+1]
	}
	return args
}
