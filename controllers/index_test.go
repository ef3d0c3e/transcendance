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
	// Boosters are bought from the inventory, there is no shop to point to
	require.NotContains(t, body, `class="index-action" href="/shop"`)
	require.NotContains(t, body, "in the shop")
	// Players learn how to add the site to their home screen
	require.Contains(t, body, "<h2>Install the app</h2>")
	require.Contains(t, body, "Add to Home Screen")
	// Counts follow the card data, which grows with every new collection
	cards := 0
	for _, collection := range tests.Data.Collections {
		cards += len(collection.Cards)
	}
	require.Contains(t, body, fmt.Sprintf(`<p class="index-stats">%d cards · %d collections</p>`, cards, len(tests.Data.Collections)))
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
	require.Contains(t, body, `<a class="index-action" href="/cards/search/">Browse the cards</a>`)
	require.NotContains(t, body, `class="index-action" href="/shop"`)
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
	require.Contains(t, rec.Body.String(), "<h2>Installer l’application</h2>")
}
