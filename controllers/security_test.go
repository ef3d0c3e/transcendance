package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"transcendance/tests"

	"github.com/stretchr/testify/require"
)

var (
	scriptTagRe    = regexp.MustCompile(`<script\b[^>]*>`)
	eventHandlerRe = regexp.MustCompile(`\son[a-z]+\s*=`)
)

// The policy only lets the site's own script files run: pages must not rely on inline scripts or handlers
func TestContentSecurityPolicy(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "csp1", "foo-bar-baz-qux", 2)
	giveBoosters(t, "csp1", collectionDev, 5, 1)
	cookie := login(t, router, "csp1", "foo-bar-baz-qux")

	for _, target := range []string{"/", "/cards/search/", "/search", "/login", "/register", "/terms", "/legal",
		"/inventory", "/inventory?tab=boosters", "/inventory?tab=tradeup", "/trades", "/notifications", "/profile",
		"/admin/users", "/opening?booster=default"} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		policy := rec.Header().Get("Content-Security-Policy")
		require.Contains(t, policy, "script-src 'self'", target)
		require.NotContains(t, policy, "unsafe-eval", target)

		body := rec.Body.String()
		for _, tag := range scriptTagRe.FindAllString(body, -1) {
			if !strings.Contains(tag, `type="application/json"`) {
				require.Contains(t, tag, "src=", "%s: inline script %s", target, tag)
			}
		}
		require.NotRegexp(t, eventHandlerRe, body, target)
	}
}
