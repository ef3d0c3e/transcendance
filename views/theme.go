package views

import (
	"github.com/gin-gonic/gin"
)

const themeContextKey string = "views.Themer"

type Themer struct {
	Available []string

	QueryParam string
	CookieName string
}

func NewThemer() (*Themer) {
	return &Themer{
		Available:  []string{"light", "dark"},
		QueryParam: "theme",
		CookieName: "theme",
	}
}

// Middleware negotiates the theme from the query parameter and/or cookie
func (t *Themer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get(themeContextKey); !exists {
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
