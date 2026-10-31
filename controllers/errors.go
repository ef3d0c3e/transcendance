package controllers

import (
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
