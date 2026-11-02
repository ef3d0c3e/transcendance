package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"transcendance/tests"

	"github.com/stretchr/testify/require"
)

func TestRobotsAndSitemap(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	req.Host = "example.test:8443"
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "Disallow: /admin/")
	require.Contains(t, rec.Body.String(), "Sitemap: https://example.test:8443/sitemap.xml")

	req = httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	req.Host = "example.test:8443"
	req.Header.Set("X-Forwarded-Proto", "https")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/xml"))
	require.Contains(t, rec.Body.String(), "<loc>https://example.test:8443/cards/search/</loc>")
	require.Contains(t, rec.Body.String(), "<loc>https://example.test:8443/collections/"+strconv.Itoa(collectionDev)+"/</loc>")
}

// Without JavaScript, opening a booster from its page shows the new cards in the inventory instead of JSON
func TestOpeningWithoutJavaScript(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)
	createAccount(t, router, "nojsopen1", "foo-bar-baz-qux", 0)
	giveBoosters(t, "nojsopen1", collectionDev, 5, 1)
	cookie := login(t, router, "nojsopen1", "foo-bar-baz-qux")
	booster := ownedBoosters(t, "nojsopen1")[0]

	form := url.Values{"booster": {strconv.FormatUint(uint64(booster.ID), 10)}}
	req := httptest.NewRequest(http.MethodPost, "/opening", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=cards&sort=recent", rec.Header().Get("Location"))
	require.Len(t, ownedCards(t, "nojsopen1"), 5)

	// The same booster again: it is gone
	req = httptest.NewRequest(http.MethodPost, "/opening", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/inventory?tab=boosters&error=opening", rec.Header().Get("Location"))
}
