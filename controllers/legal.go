package controllers

import (
	"transcendance/views"

	"github.com/gin-gonic/gin"
)

// Public repository, used as the contact channel
const repositoryURL = "https://github.com/ef3d0c3e/transcendance"

// GitHub handles of the team behind the site
var teamMembers = []string{"ef3d0c3e", "cheldux", "SuddenLeeus", "zelie-delhotal"}

// Render a static informational page with the given template and title key
func (lc *Controller) renderInfoPage(c *gin.Context, page string, title string) {
	user := GetAuthenticatedUser(c)

	builder := views.PageBuilder("base", map[string]any{
		"Title": lc.Renderer.T(c, title),
		"User":  user,
	})
	builder.Add(page, "Content", map[string]any{
		"Repository": repositoryURL,
		"Team":       teamMembers,
	})
	lc.Renderer.Render(c, &builder)
}

// /terms GET: terms of service
func (lc *Controller) TermsGet(c *gin.Context) {
	lc.renderInfoPage(c, "terms", "terms-title")
}

// /legal GET: legal notice, privacy policy and cookies
func (lc *Controller) LegalGet(c *gin.Context) {
	lc.renderInfoPage(c, "legal", "legal-title")
}

// /contact GET: contact information
func (lc *Controller) ContactGet(c *gin.Context) {
	lc.renderInfoPage(c, "contact", "contact-title")
}
