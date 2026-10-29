package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"transcendance/tests"

	"github.com/stretchr/testify/require"
)

func TestInfoPages(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	pages := map[string]string{
		"/terms":   "<h1>Terms of Service</h1>",
		"/legal":   "<h1>Legal notice &amp; privacy</h1>",
		"/contact": "<h1>Contact</h1>",
	}
	for path, title := range pages {
		body := getPage(t, router, path)
		require.Contains(t, body, title, path)
		// The footer links every page, and marks the current one
		require.Contains(t, body, `<footer class="site-footer">`, path)
		require.Contains(t, body, `<a href="`+path+`" aria-current="page">`, path)
	}

	body := getPage(t, router, "/legal")
	require.Contains(t, body, `<code>session_token</code>`)
	require.Contains(t, body, `href="https://github.com/ef3d0c3e"`)
}

func TestInfoPagesFrench(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	req := httptest.NewRequest(http.MethodGet, "/legal", nil)
	req.AddCookie(&http.Cookie{Name: "lang", Value: "fr"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "<h1>Mentions légales et confidentialité</h1>")
}

// Pages load no resource from a third party, which would learn about the visit
func TestNoThirdPartyResources(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	body := getPage(t, router, "/terms")
	require.NotRegexp(t, `<(script|link|img)[^>]+(src|href)="(https?:)?//`, body)
	require.Contains(t, body, `<script src="/static/vendor/e-layout.min.js">`)
}

func TestRegisterLinksTerms(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	body := getPage(t, router, "/register")
	require.Contains(t, body, `<a href="/terms" target="_blank" rel="noopener">Terms of Service</a>`)
}

func TestAccountDeleteButton(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "delbtn1", "123456789", 0)
	createAccount(t, router, "delbtn2", "123456789", 0)
	cookie := login(t, router, "delbtn1", "123456789")

	// Only offered on the user's own profile editor
	body := getInventory(t, router, cookie, "/profile")
	require.Contains(t, body, `data-account-delete="/profiles/`+fmt.Sprint(userID(t, "delbtn1"))+`/delete"`)
	body = getInventory(t, router, cookie, "/profile/delbtn2")
	require.NotContains(t, body, "data-account-delete")

	// Deleting no longer works with a GET request, which other sites could trigger
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/profiles/%d/delete", userID(t, "delbtn1")), nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}
