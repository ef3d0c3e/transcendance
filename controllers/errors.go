package controllers

import (
	"log"
	"net/http"

	"transcendance/views"

	"github.com/gin-gonic/gin"
)

// Unknown pages: the site's error page, with its header and footer
func (ctrl *Controller) NotFound(c *gin.Context) {
	ctrl.Renderer.RenderError(c, views.ErrorPage{
		User:       GetAuthenticatedUser(c),
		StatusCode: http.StatusNotFound,
		Message:    ctrl.Renderer.T(c, "error-not-found"),
	})
}

// InternalError logs an unexpected error and answers a generic message: database errors name tables and columns,
// which clients have no use for
func InternalError(c *gin.Context, renderer *views.Renderer, err error) {
	log.Printf("ERROR %s %s: %s", c.Request.Method, c.Request.URL.Path, err)
	c.String(http.StatusInternalServerError, renderer.T(c, "error-unexpected"))
}
