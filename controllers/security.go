package controllers

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// What pages may load, see SecurityHeaders. Scripts only come from the site's own files, so an injected
// <script> or onclick= would not run. Styles allow inline ones: templates set colors in style attributes, and
// components add their own.
var contentSecurityPolicy = strings.Join([]string{
	"default-src 'self'",
	"script-src 'self'",
	"style-src 'self' 'unsafe-inline'",
	"img-src 'self' data: blob:",
	"font-src 'self'",
	"media-src 'self' data: blob:",
	"worker-src 'self' blob:",
	"object-src 'none'",
	"base-uri 'self'",
	"form-action 'self'",
	"frame-ancestors 'self'",
}, "; ")

// SecurityHeaders sends the Content-Security-Policy of the site with every response. Requests and the chat's
// WebSocket go to the site itself, its ws: and wss: addresses are named since older browsers don't count them as
// 'self'.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		if strings.ContainsAny(host, " ;'\"") {
			host = ""
		}
		connect := "connect-src 'self'"
		if host != "" {
			connect += " ws://" + host + " wss://" + host
		}
		c.Header("Content-Security-Policy", contentSecurityPolicy+"; "+connect)
		c.Next()
	}
}
