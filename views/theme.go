package views

import (
	"github.com/gin-gonic/gin"
)

const themeContextKey string = "views.Themer"

type Themer struct {
	// Available themes
	Available []string

	// Theme GET parameter
	QueryParam string
	// Theme cookie name
	CookieName string
}

func NewThemer() (*Themer) {
	return &Themer{
		Available:  []string{"light", "dark"},
		QueryParam: "theme",
		CookieName: "theme",
	}
}

// Theme middleware. Negotiate theme based on request parameters and/or cookie
func (t *Themer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get(themeContextKey); !exists {
			// Negotiate and store
			theme := t.negotiate(c)
			c.Set(themeContextKey, theme)

			// Cookie is set by the client side javascript
		}
		c.Next()
	}
}

func (t *Themer) Theme(c *gin.Context) string {
	if theme, exists := c.Get(themeContextKey); exists {
		return theme.(string)
	}

	theme := t.negotiate(c)
	c.Set(themeContextKey, theme)
	return theme
}

// negotiate builds the requested locales list, using the cache, and returns a Localization
func (t *Themer) negotiate(c *gin.Context) string {
	requested := t.Available[0]
	if v, err := c.Cookie(t.CookieName); err == nil && v != "" {
		for _, theme := range t.Available {
			if theme == v {
				requested = theme
				break
			}
		}
	}

	if v := c.Query(t.QueryParam); v != "" {
		for _, theme := range t.Available {
			if theme == v {
				requested = theme
				break
			}
		}
	}

	return requested
}
