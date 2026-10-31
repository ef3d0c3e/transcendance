package data

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"transcendance/controllers"
	"transcendance/data"
	"transcendance/views"

	"github.com/gin-gonic/gin"
)

const (
	pageSize = 20
)

// Get template data for a card
func GetCardData(c *gin.Context, loc string, renderer *views.Renderer, collection *data.Collection, card *data.Card) (map[string]any, error) {
	rarity := renderer.T(c, "card-rarity-"+card.Rarity)
	var rarity_color string
	switch card.Rarity {
	case "common":
		rarity_color = "#407fff"
	case "rare":
		rarity_color = "#ff1440"
	case "legendary":
		rarity_color = "#ffdc50"
	}
	id := collection.ID*1000 + card.ID

	colFormat := "%d / %d"
	if len(collection.Cards) >= 100 {
		colFormat = "%03d / %03d"
	} else if len(collection.Cards) >= 10 {
		colFormat = "%02d / %02d"
	}
	title, err := card.Translate(loc, "title")
	if err != nil {
		return nil, err
	}
	description, err := card.Translate(loc, "description")
	if err != nil {
		return nil, err
	}
	collectionTitle, err := collection.Translate(loc, "title")
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"CardContainerData":   template.HTMLAttr(`data-tilt`),
		"CardColorPrimary":    template.CSS(card.ColorPrimary),
		"CardColorSecondary":  template.CSS(card.ColorSecondary),
		"CardBadgeColor":      template.CSS(rarity_color),
		"CardArtwork":         template.URL(fmt.Sprint("/cards/", id, "/thumbnail")),
		"CardBadge":           strings.ToUpper(rarity),
		"CardTitle":           title,
		"CardDescription":     description,
		"CardCollectionTitle": collectionTitle,
		"CardCollectionID":    collection.ID,
		"CardCount":           fmt.Sprintf(colFormat, card.ID, len(collection.Cards)),
	}, nil
}

// cards/* route
func (dc *DataController) CardsGet(c *gin.Context) {
	first := c.Param("FIRST")
	second := c.Param("SECOND")

	switch first {
	case "assets":
		c.File("data/assets/" + second)
		return
	case "search":
		dc.catalogCards(c)
		return
	}

	id, err := strconv.Atoi(first)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "card-error-not-found"),
		})
		return
	}

	card, collection := dc.Data.GetCard(id)
	if card == nil || collection == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "card-error-not-found"),
		})
		return
	}

	switch second {
	case "/artwork":
		c.File(card.ArtworkPath)
	case "/thumbnail":
		c.File(card.ThumbnailPath)
	}
}

// Get template data for a collection
func getCollectionData(c *gin.Context, loc string, renderer *views.Renderer, collection *data.Collection) (map[string]any, error) {
	title, err := collection.Translate(loc, "title")
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"CollectionContainerData": template.HTMLAttr(`data-tilt`),
		"CollectionArtwork":       "/collections/" + strconv.Itoa(collection.ID) + "/thumbnail",
		"CollectionTitle":         title,
		"ColorPrimary":            template.CSS(collection.ColorPrimary),
		"ColorSecondary":          template.CSS(collection.ColorSecondary),
		"CollectionDescription":   renderer.T(c, "collection-booster", "count", collection.BoosterSize),
		"CollectionCount":         renderer.T(c, "collection-count", "count", len(collection.Cards)),
		"CollectionID":            renderer.T(c, "collection-id", "id", fmt.Sprintf("%03d", collection.ID)),
	}, nil
}

// collections/* route
func (dc *DataController) CollectionsGet(c *gin.Context) {
	first := c.Param("FIRST")
	second := c.Param("SECOND")

	switch first {
	case "search":
		// Collections are listed in the inventory, with the user's progress
		if controllers.GetAuthenticatedUser(c) == nil {
			c.Redirect(http.StatusSeeOther, "/cards/search/")
		} else {
			c.Redirect(http.StatusSeeOther, "/inventory?tab=collections")
		}
		return
	}

	id, err := strconv.Atoi(first)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "collection-error-not-found"),
		})
		return
	}

	collection := dc.Data.GetCollection(id)
	if collection == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "collection-error-not-found"),
		})
		return
	}

	// Show cards in collection
	if second == "/" || second == "" {
		dc.collectionCards(c, collection)
		return
	}

	switch second {
	case "/thumbnail":
		c.File(collection.ContectSheet)
	}
}
