package data

import (
	"fmt"
	"html/template"
	"log"
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
	user := controllers.GetAuthenticatedUser(c)
	first := c.Param("FIRST")
	second := c.Param("SECOND")

	switch first {
	case "assets":
		c.File("data/assets/" + second)
		return
	case "search":
		query := c.Query("q")
		page, err := strconv.Atoi(c.Query("p"))
		if err != nil || page < 1 {
			page = 1
		}

		if query != "" {
			loc := dc.Renderer.Localizer.GetLocale(c)
			ss, err := dc.GetCardSearchSet(loc)
			if err != nil {
				log.Printf("[WARN] Failed to get card search set for locale '%s': %s", loc, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": dc.Renderer.T(c, "card-search-error-internal"),
				})
				return
			}
			results, total, err := ss.Search(query, pageSize, (page-1)*pageSize)
			if err != nil {
				log.Printf("[WARN] Search failed for locale: '%s', term '%s': %s", loc, query, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": dc.Renderer.T(c, "card-search-error-internal"),
				})
				return
			}

			cards := make([]map[string]any, len(results))
			for i, pair := range results {
				data, err := GetCardData(c, loc, dc.Renderer, pair.collection, pair.card)
				if err != nil {
					log.Printf("[WARN] Failed to build data for card '%s': %s", pair.card.Name, err)
					c.JSON(http.StatusInternalServerError, gin.H{
						"message": dc.Renderer.T(c, "card-search-error-internal"),
					})
					return
				}
				cards[i] = map[string]any{
					"Data": data,
				}
			}

			builder := views.PageBuilder("base", map[string]any{
				"Title": "Login",
				"User":  user,
			})
			content := builder.Add("card-search", "Content", map[string]any{
				"Query": query,
				"Cards": cards,
			})
			content.Add("paginator", "Paginator", map[string]any{
				"Current": page,
				"Total":   int(total / uint64(pageSize)),
				"Url":     "?q=" + query + "&p=",
			})
			dc.Renderer.Render(c, &builder)
			return
		}

		builder := views.PageBuilder("base", map[string]any{
			"Title": dc.Renderer.T(c, "card-search-title"),
			"User":  user,
		})
		builder.Add("card-search", "Content", map[string]any{
			"Query": query,
			"Cards": []map[string]any{},
		})
		dc.Renderer.Render(c, &builder)
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
		c.File(card.ArtworkPath)
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
	user := controllers.GetAuthenticatedUser(c)
	first := c.Param("FIRST")
	second := c.Param("SECOND")

	switch first {
	case "search":
		query := c.Query("q")
		page, err := strconv.Atoi(c.Query("p"))
		if err != nil || page < 1 {
			page = 1
		}

		if query != "" {
			loc := dc.Renderer.Localizer.GetLocale(c)
			ss, err := dc.GetCollectionSearchSet(loc)
			if err != nil {
				log.Printf("[WARN] Failed to get collection search set for locale '%s': %s", loc, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": dc.Renderer.T(c, "collection-search-error-internal"),
				})
				return
			}
			results, total, err := ss.Search(query, pageSize, (page-1)*pageSize)
			if err != nil {
				log.Printf("[WARN] Search failed for locale: '%s', term '%s': %s", loc, query, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": dc.Renderer.T(c, "collection-search-error-internal"),
				})
				return
			}

			collections := make([]map[string]any, len(results))
			for i, collection := range results {
				data, err := getCollectionData(c, loc, dc.Renderer, collection)
				if err != nil {
					log.Printf("[WARN] Failed to build data for collection '%s': %s", collection.Name, err)
					c.JSON(http.StatusInternalServerError, gin.H{
						"message": dc.Renderer.T(c, "collection-search-error-internal"),
					})
					return
				}
				collections[i] = map[string]any{
					"Data": data,
				}
			}

			builder := views.PageBuilder("base", map[string]any{
				"Title": "Login",
				"User":  user,
			})
			content := builder.Add("collection-search", "Content", map[string]any{
				"Query":       query,
				"Collections": collections,
			})
			content.Add("paginator", "Paginator", map[string]any{
				"Current": page,
				"Total":   int(total / uint64(pageSize)),
				"Url":     "?q=" + query + "&p=",
			})
			dc.Renderer.Render(c, &builder)
			return
		}

		builder := views.PageBuilder("base", map[string]any{
			"Title": dc.Renderer.T(c, "collection-search-title"),
			"User":  user,
		})
		builder.Add("collection-search", "Content", map[string]any{
			"Query":       query,
			"Collections": []map[string]any{},
		})
		dc.Renderer.Render(c, &builder)
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
		loc := dc.Renderer.Localizer.GetLocale(c)
		title, err := collection.Translate(loc, "title")
		if err != nil {
			log.Printf("[WARN] Failed to build data for collection '%s': %s", collection.Name, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": dc.Renderer.T(c, "collection-search-error-internal"),
			})
			return
		}
		builder := views.PageBuilder("base", map[string]any{
			"Title": dc.Renderer.T(c, "collection-title", "collection", title),
			"User":  user,
		})

		cards := make([]map[string]any, len(collection.Cards))
		for i := range collection.Cards {
			card, _ := dc.Data.GetCard(collection.ID*1000 + i + 1)
			data, err := GetCardData(c, loc, dc.Renderer, collection, card)
			if err != nil {
				log.Printf("[WARN] Failed to build data for collection '%s': %s", collection.Name, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": dc.Renderer.T(c, "collection-search-error-internal"),
				})
			}

			cards[i] = map[string]any{
				"Data": data,
			}
		}

		builder.Add("card-collection", "Content", map[string]any{
			"CollectionTitle": title,
			"Cards":           cards,
		})

		dc.Renderer.Render(c, &builder)
		return
	}

	switch second {
	case "/thumbnail":
		c.File(collection.ContectSheet)
	}
}
