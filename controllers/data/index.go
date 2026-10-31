package data

import (
	"fmt"
	"log"
	"net/http"
	"transcendance/controllers"
	"transcendance/views"

	"github.com/gin-gonic/gin"
)

// Number of collections and cards showcased on the homepage
const (
	indexCollections = 4
	indexCards       = 4
)

// / GET: homepage, presents the site and showcases collections and cards
func (dc *DataController) IndexGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	loc := dc.Renderer.Localizer.GetLocale(c)

	collections := dc.sortedCollections()
	cardCount := 0
	var stacks []cardStack
	for _, collection := range collections {
		for _, colCard := range collection.Cards {
			card := dc.Data.Cards[colCard.Name]
			if card == nil {
				continue
			}
			cardCount++
			id := uint(collection.ID*1000 + card.ID)
			stacks = append(stacks, cardStack{CardID: id, card: card, collection: collection})
		}
	}

	// Rarest cards, in collection order
	sortCardStacks(stacks, inventorySortRarity, loc, nil)
	cards := make([]map[string]any, 0, indexCards)
	for _, stack := range stacks[:min(indexCards, len(stacks))] {
		data, err := GetCardData(c, loc, dc.Renderer, stack.collection, stack.card)
		if err != nil {
			log.Printf("WARN failed to build data for card %d: %s", stack.CardID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": dc.Renderer.T(c, "card-search-error-internal"),
			})
			return
		}
		cards = append(cards, map[string]any{"Data": data})
	}

	showcased := make([]map[string]any, 0, indexCollections)
	for _, collection := range collections[:min(indexCollections, len(collections))] {
		data, err := getCollectionData(c, loc, dc.Renderer, collection)
		if err != nil {
			log.Printf("WARN failed to build data for collection '%s': %s", collection.Name, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": dc.Renderer.T(c, "collection-search-error-internal"),
			})
			return
		}
		showcased = append(showcased, map[string]any{
			"Data": data,
			"Url":  fmt.Sprint("/collections/", collection.ID, "/"),
		})
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": dc.Renderer.T(c, "index-title"),
		"User":  user,
	})
	builder.Add("index", "Content", map[string]any{
		"User":            user,
		"CardCount":       dc.Renderer.T(c, "index-stat-cards", "count", cardCount),
		"CollectionCount": dc.Renderer.T(c, "index-stat-collections", "count", len(collections)),
		"Collections":     showcased,
		"Cards":           cards,
	})
	dc.Renderer.Render(c, &builder)
}
