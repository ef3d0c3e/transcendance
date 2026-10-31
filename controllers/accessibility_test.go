package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"transcendance/tests"

	"github.com/stretchr/testify/require"
)

// Pages are in the negotiated language, for screen readers' pronunciation
func TestPageLanguage(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	require.Contains(t, getPage(t, router, "/terms"), `<html lang="en" `)
	require.Contains(t, getPage(t, router, "/terms?lang=fr"), `<html lang="fr" `)
}

// The skip link comes first, and leads past the header to the content
func TestSkipLink(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	body := getPage(t, router, "/cards/search/")
	link := strings.Index(body, `<a class="skip-link" href="#main">Skip to content</a>`)
	require.NotEqual(t, -1, link)
	require.Less(t, link, strings.Index(body, "<header>"))
	require.Contains(t, body, `<main id="main" tabindex="-1">`)
}

// The site name is not a heading: each page has its title as its only h1
func TestOneTitlePerPage(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	for _, target := range []string{"/", "/cards/search/", "/search", "/terms", "/legal", "/contact", "/login", "/register"} {
		require.Equal(t, 1, strings.Count(getPage(t, router, target), "<h1"), target)
	}
}

// Paginators at the top and bottom of a list are told apart
func TestPaginatorLabels(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	body := getPage(t, router, "/cards/search/")
	require.Equal(t, 1, strings.Count(body, `<nav class="paginator" aria-label="Pagination">`))
	require.Equal(t, 1, strings.Count(body, `<nav class="paginator" aria-label="Pagination, bottom of the list">`))
}

// Unknown pages show the site's error page, not a bare text
func TestNotFoundPage(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)

	body := rec.Body.String()
	require.Contains(t, body, "<title>Error 404</title>")
	require.Contains(t, body, `<h1 id="error-title">Error 404</h1>`)
	require.Contains(t, body, "This page does not exist")
}
