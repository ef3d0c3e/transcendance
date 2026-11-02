package controllers

import (
	"encoding/xml"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Address of the site as the browser reached it, e.g "https://example.org:8443"
func siteURL(c *gin.Context) string {
	scheme := "http"
	if secureRequest(c) {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}

// /robots.txt GET: crawlers index the public pages, not the personal ones
func (sc *Controller) RobotsGet(c *gin.Context) {
	lines := []string{"User-agent: *"}
	for _, path := range []string{"/admin/", "/api/", "/inventory", "/opening", "/trades", "/notifications", "/friends", "/shop/"} {
		lines = append(lines, "Disallow: "+path)
	}
	lines = append(lines, "", "Sitemap: "+siteURL(c)+"/sitemap.xml", "")
	c.String(http.StatusOK, strings.Join(lines, "\n"))
}

type sitemapURL struct {
	Loc string `xml:"loc"`
}

type sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// /sitemap.xml GET: the public pages, the catalog and every collection
func (sc *Controller) SitemapGet(c *gin.Context) {
	base := siteURL(c)
	paths := []string{"/", "/cards/search/", "/search", "/register", "/login", "/terms", "/legal", "/contact"}
	ids := make([]int, 0, len(sc.Data.Collections))
	for _, collection := range sc.Data.Collections {
		ids = append(ids, collection.ID)
	}
	slices.Sort(ids)
	for _, id := range ids {
		paths = append(paths, "/collections/"+strconv.Itoa(id)+"/")
	}

	urls := make([]sitemapURL, len(paths))
	for i, path := range paths {
		urls[i] = sitemapURL{Loc: base + path}
	}
	body, err := xml.Marshal(sitemap{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: urls})
	if err != nil {
		InternalError(c, sc.Renderer, err)
		return
	}
	c.Data(http.StatusOK, "application/xml; charset=utf-8", append([]byte(xml.Header), body...))
}
