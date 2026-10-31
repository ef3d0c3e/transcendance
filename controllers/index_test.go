package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"transcendance/tests"

	"github.com/stretchr/testify/require"
)

func TestIndexPage(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	body := getPage(t, router, "/")
	require.Contains(t, body, "<h1>Collect, trade and show off your cards</h1>")
	require.Contains(t, body, "<h2>Trade cards</h2>")
	// The stats count the cards of every collection, like the page does
	noun := func(count int, word string) string {
		if count == 1 {
			return fmt.Sprintf("%d %s", count, word)
		}
		return fmt.Sprintf("%d %ss", count, word)
	}
	cards, collections := 0, 0
	for _, collection := range tests.Data.Collections {
		collections++
		for _, colCard := range collection.Cards {
			if tests.Data.Cards[colCard.Name] != nil {
				cards++
			}
		}
	}
	require.Contains(t, body, fmt.Sprintf(`<p class="index-stats">%s · %s</p>`, noun(cards, "card"), noun(collections, "collection")))
	// Visitors are invited to register
	require.Contains(t, body, `<a class="index-action-primary" href="/register">Create an account</a>`)
	require.NotContains(t, body, `href="/inventory">Open your inventory</a>`)
	// The rarest cards are showcased first, and every collection is linked
	require.Contains(t, body, `<h2 class="card-title">Ford Model T</h2>`)
	require.NotContains(t, body, "COMMON")
	require.Contains(t, body, `<a class="catalog-collection" href="/collections/1/">`)
	require.Contains(t, body, `<a class="catalog-collection" href="/collections/2/">`)

	// Users are sent to their inventory instead
	createAccount(t, router, "indexuser", "123456789", 0)
	cookie := login(t, router, "indexuser", "123456789")
	body = getInventory(t, router, cookie, "/")
	require.Contains(t, body, `<a class="index-action-primary" href="/inventory">Open your inventory</a>`)
	require.NotContains(t, body, `href="/register">Create an account</a>`)
}

func TestIndexPageFrench(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "lang", Value: "fr"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "<h1>Collectionnez, troquez et exposez vos cartes</h1>")
	require.Contains(t, rec.Body.String(), "<h2>Troquez vos cartes</h2>")
	require.Contains(t, rec.Body.String(), "Créer un compte</a>")
}
