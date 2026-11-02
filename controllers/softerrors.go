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

// FormFallback lets a handler answering JSON to page scripts serve plain form submissions too, for browsers
// without JavaScript: its answer is replaced by a redirect to `success`, or to `failure` when it failed.
func FormFallback(handler gin.HandlerFunc, success string, failure string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if wantsJSON(c) {
			handler(c)
			return
		}
		discard := &discardWriter{ResponseWriter: c.Writer}
		c.Writer = discard
		handler(c)
		c.Writer = discard.ResponseWriter
		target := success
		if discard.status >= 300 {
			target = failure
		}
		c.Redirect(http.StatusSeeOther, target)
	}
}

// discardWriter records the status of a response and drops it
type discardWriter struct {
	gin.ResponseWriter
	status int
}

func (w *discardWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
}

func (w *discardWriter) WriteHeaderNow() {}

func (w *discardWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return len(data), nil
}

func (w *discardWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *discardWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *discardWriter) Written() bool {
	return w.status != 0
}
