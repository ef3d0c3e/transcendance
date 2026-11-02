package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Request header of the pages' fetch helper (frontend/src/api.js) asking for soft errors
const softErrorsHeader = "X-Soft-Errors"

// Response header holding the real status of a softened error
const softStatusHeader = "X-Status"

// SoftErrors answers client errors (4xx) with 200 and the real status in `X-Status`, for requests sending
// `X-Soft-Errors: 1`. Browsers log every failed request in the console, so a wrong password typed in a form
// submitted with fetch would show an error there; the page reads the status from the header instead. The body
// is unchanged, and logs and metrics still see the real status.
func SoftErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader(softErrorsHeader) == "1" {
			c.Writer = &softErrorWriter{ResponseWriter: c.Writer}
		}
		c.Next()
	}
}

type softErrorWriter struct {
	gin.ResponseWriter
	// Real status of a softened error, 0 otherwise
	status int
}

func (w *softErrorWriter) WriteHeader(code int) {
	if code >= 400 && code < 500 && !w.Written() {
		w.status = code
		w.Header().Set(softStatusHeader, strconv.Itoa(code))
		code = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *softErrorWriter) Status() int {
	if w.status != 0 {
		return w.status
	}
	return w.ResponseWriter.Status()
}

// wantsJSON reports whether the request comes from a page script expecting JSON rather than a plain form
// submission, which gets a redirect
func wantsJSON(c *gin.Context) bool {
	return c.GetHeader(softErrorsHeader) == "1" || c.NegotiateFormat(gin.MIMEJSON, gin.MIMEHTML) == gin.MIMEJSON
}
