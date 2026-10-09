package controllers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"transcendance/data"
	"transcendance/views"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/analysis/char/asciifolding"
	"github.com/blevesearch/bleve/analysis/token/lowercase"
	"github.com/blevesearch/bleve/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/search/query"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DataController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
	Data     *data.Data
	// Per-locale search set
	Search map[string]*SearchSet
}

type CardPair struct {
	card       *data.Card
	collection *data.Collection
}

func getCardData(c *gin.Context, loc string, renderer *views.Renderer, collection *data.Collection, card *data.Card) map[string]any {
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
	return map[string]any{
		"CardContainerData":   template.HTMLAttr(`data-tilt`),
		"CardColorPrimary":    template.CSS(card.ColorPrimary),
		"CardColorSecondary":  template.CSS(card.ColorSecondary),
		"CardBadgeColor":      template.CSS(rarity_color),
		"CardArtwork":         template.URL(fmt.Sprint("/cards/", id, "/thumbnail")),
		"CardBadge":           strings.ToUpper(rarity),
		"CardTitle":           card.Translate(loc, "title"),
		"CardDescription":     card.Translate(loc, "description"),
		"CardCollectionTitle": collection.Translate(loc, "title"),
		"CardCollectionID":    collection.ID,
		"CardCount":           fmt.Sprintf(colFormat, card.ID, len(collection.Cards)),
	}
}

func (dc *DataController) CardGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	first := c.Param("FIRST")
	second := c.Param("SECOND")
	if first == "assets" {
		// Return assets
		c.File("data/assets/" + second)
		return
	} else if first == "search" {
		// Search cards
		query := c.Query("query")

		if query != "" {
			loc := dc.Renderer.Localizer.GetLocale(c)
			if dc.Search == nil {
				dc.Search = map[string]*SearchSet{}
			}

			if dc.Search[loc] == nil {
				ss, err := dc.BuildSearchSet(loc)
				if err != nil {
					log.Fatalf("Failed to build search set for locale '%s': %s", loc, err)
					c.JSON(http.StatusInternalServerError, gin.H{
						"message": dc.Renderer.T(c, "card-search-error-internal"),
					})
					return
				}
				dc.Search[loc] = ss
			}
			results, err := dc.Search[loc].Search(query, 20, 0)
			if err != nil {
				log.Fatalf("Search failed for locale: '%s', term '%s': %s", loc, query, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": dc.Renderer.T(c, "card-search-error-internal"),
				})
				return
			}

			cards := make([]map[string]any, len(results))
			for i, pair := range results {
				cards[i] = map[string]any{
					"Data": getCardData(c, loc, dc.Renderer, pair.collection, pair.card),
				}
			}

			builder := views.PageBuilder("base", map[string]any{
				"Title": "Login",
				"User":  user,
			})
			builder.Add("card-search", "Content", map[string]any{
				"Query": query,
				"Cards": cards,
			})
			dc.Renderer.Render(c, &builder)
			return
		}

		builder := views.PageBuilder("base", map[string]any{
			"Title": "Login",
			"User":  user,
		})
		builder.Add("card-search", "Content", map[string]any{})
		dc.Renderer.Render(c, &builder)
		return
	}

	id, err := strconv.Atoi(first)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "cards-error-not-found"),
		})
		return
	}

	card, collection := dc.Data.GetCard(id)
	if card == nil || collection == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "cards-error-not-found"),
		})
		return
	}

	switch second {
	case "/info":
		builder := views.PageBuilder("base", map[string]any{
			"Title": "Login",
			"User":  user,
		})
		builder.Add("card-info", "Content", map[string]any{
			"User":       user,
			"Card":       card,
			"Collection": collection,
			"ID":         id,
		})
		dc.Renderer.Render(c, &builder)
	case "/show":
		loc := dc.Renderer.Localizer.GetLocale(c)
		builder := views.PageBuilder("base", map[string]any{
			"Title": "Login",
			"User":  user,
		})
		builder.Add("card", "Content", getCardData(c, loc, dc.Renderer, collection, card))
		dc.Renderer.Render(c, &builder)
	case "/artwork":
		c.File(card.ArtworkPath)
	case "/thumbnail":
		c.File(card.ThumbnailPath)
	}
}

func (dc *DataController) CollectionGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	first := c.Param("FIRST")
	id, err := strconv.Atoi(first)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "cards-error-collection-not-found"),
		})
		return
	}

	collection := dc.Data.GetCollection(id)
	if collection == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "cards-error-collection-not-found"),
		})
		return
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": "Login",
		"User":  user,
	})

	loc := dc.Renderer.Localizer.GetLocale(c)
	cards := make([]map[string]any, len(collection.Cards))
	for i := range collection.Cards {
		card, _ := dc.Data.GetCard(collection.ID*1000 + i + 1)

		cards[i] = map[string]any{
			"Data": getCardData(c, loc, dc.Renderer, collection, card),
		}
	}
	for k, v := range cards {

		print(k, " : ", v, "\n")
	}

	builder.Add("card-collection", "Content", map[string]any{
		"CollectionTitle": collection.Translate(loc, "title"),
		"Cards":           cards,
	})

	dc.Renderer.Render(c, &builder)
}

// Per (locale, card) entry for bleve search
type IndexableCard struct {
	Name                  string
	Description           string
	Tags                  []string
	CollectionName        string
	CollectionDescription string
	CollectionTags        []string
	Rarity                string
}

func (dc *DataController) toIndexable(locale string, card *data.Card, collection *data.Collection) (*IndexableCard, error) {

	cardLoc := card.Locales[locale]
	collectionLoc := collection.Locales[locale]
	tagsLoc := dc.Data.Tags[locale]

	// Card data
	nameMsg, _ := cardLoc.Message("title")
	name, err := cardLoc.FormatPattern(nameMsg.Value(), map[string]any{})
	if err != nil {
		return nil, err
	}

	descMsg, _ := cardLoc.Message("description")
	desc, err := cardLoc.FormatPattern(descMsg.Value(), map[string]any{})
	if err != nil {
		return nil, err
	}

	tags := make([]string, 0)
	for _, tag := range card.Tags {
		tagMsg, _ := tagsLoc.Message(tag)
		tagString, err := cardLoc.FormatPattern(tagMsg.Value(), map[string]any{})
		if err != nil {
			return nil, err
		}
		tags = append(tags, tagString)
	}

	// Collection data
	collectionNameMsg, _ := collectionLoc.Message("title")
	collectionName, err := collectionLoc.FormatPattern(collectionNameMsg.Value(), map[string]any{})
	if err != nil {
		return nil, err
	}

	collectionDescMsg, _ := collectionLoc.Message("description")
	collectionDesc, err := collectionLoc.FormatPattern(collectionDescMsg.Value(), map[string]any{})
	if err != nil {
		return nil, err
	}

	collectionTags := make([]string, 0)
	for _, tag := range collection.Tags {
		tagMsg, _ := tagsLoc.Message(tag)
		collectionTagstring, err := collectionLoc.FormatPattern(tagMsg.Value(), map[string]any{})
		if err != nil {
			return nil, err
		}
		collectionTags = append(collectionTags, collectionTagstring)
	}

	rarity, err := dc.Renderer.Localizer.LocalizeFor(locale, "card-rarity-"+card.Rarity)
	if err != nil {
		return nil, err
	}

	return &IndexableCard{
		Name:                  name,
		Description:           desc,
		Tags:                  tags,
		CollectionName:        collectionName,
		CollectionDescription: collectionDesc,
		CollectionTags:        collectionTags,
		Rarity:                rarity,
	}, nil

}

func buildMapping() *mapping.IndexMappingImpl {

	im := bleve.NewIndexMapping()
	im.DefaultAnalyzer = "standard"
	if err := im.AddCustomCharFilter("ascii_fold", map[string]interface{}{
		"type": asciifolding.Name,
	}); err != nil {
		log.Fatal("Failed to create ascii_fold filter: ", err)
	}

	if err := im.AddCustomAnalyzer("accent_insensitive", map[string]interface{}{
		"type":          custom.Name,
		"tokenizer":     unicode.Name,
		"char_filters":  []string{"ascii_fold"},
		"token_filters": []string{lowercase.Name},
	}); err != nil {
		log.Fatal("Failed to create accent_insensitive analyzer: ", err)
	}

	card := bleve.NewDocumentMapping()
	fm := bleve.NewTextFieldMapping()
	fm.Analyzer = "accent_insensitive"

	card.AddFieldMappingsAt("Name", fm)
	card.AddFieldMappingsAt("Tags", fm)
	card.AddFieldMappingsAt("Description", fm)
	card.AddFieldMappingsAt("CollectionName", fm)
	card.AddFieldMappingsAt("CollectionTags", fm)
	card.AddFieldMappingsAt("CollectionDescription", fm)
	card.AddFieldMappingsAt("Rarity", fm)

	im.AddDocumentMapping("card", card)
	im.DefaultMapping = card
	return im
}

type SearchSet struct {
	locale string
	idx    bleve.Index
	byID   map[string]CardPair
}

func (dc *DataController) BuildSearchSet(locale string) (*SearchSet, error) {
	idx, err := bleve.NewMemOnly(buildMapping())
	if err != nil {
		return nil, fmt.Errorf("creating index: %w", err)
	}

	ss := &SearchSet{
		locale: locale,
		idx:    idx,
		byID:   make(map[string]CardPair, len(dc.Data.Cards)),
	}

	batch := idx.NewBatch()
	for _, collection := range dc.Data.Collections {
		for _, colCard := range collection.Cards {
			card := dc.Data.Cards[colCard.Name]

			ic, err := dc.toIndexable(locale, card, collection)
			id := card.ID + collection.ID*1000
			if err != nil {
				return nil, fmt.Errorf("indexing card %d: %w", id, err)
			}
			if err := batch.Index(card.Name, ic); err != nil {
				return nil, err
			}
			ss.byID[card.Name] = CardPair{
				card:       card,
				collection: collection,
			}

		}
	}

	if err := idx.Batch(batch); err != nil {
		return nil, fmt.Errorf("committing batch: %w", err)
	}

	return ss, nil
}

func fieldQuery(field, term string, boost float64) query.Query {
	mq := bleve.NewMatchQuery(term)
	mq.SetField(field)
	mq.SetBoost(boost)
	return mq
}

func (ss *SearchSet) Search(search string, limit int, offset int) ([]CardPair, error) {
	terms := strings.Split(search, " ")

	termQueries := make([]query.Query, len(terms))
	for i, term := range terms {
		nameM := bleve.NewMatchQuery(term)
		nameM.SetField("Name")
		nameM.SetBoost(8)

		nameP := bleve.NewPrefixQuery(term)
		nameP.SetField("Name")
		nameP.SetBoost(6)

		nameF := bleve.NewFuzzyQuery(term)
		nameF.SetField("Name")
		nameF.SetBoost(6)
		nameF.SetFuzziness(2)
		nameF.SetPrefix(1)

		tagsM := bleve.NewMatchQuery(term)
		tagsM.SetField("Tags")
		tagsM.SetBoost(4)

		collectionM := bleve.NewMatchQuery(term)
		collectionM.SetField("CollectionName")
		collectionM.SetBoost(3)

		rarityM := bleve.NewMatchQuery(term)
		rarityM.SetField("Rarity")
		rarityM.SetBoost(2)

		descriptionM := bleve.NewMatchQuery(term)
		descriptionM.SetField("Description")
		descriptionM.SetBoost(1)

		termQueries[i] = bleve.NewDisjunctionQuery(
			nameM, nameP, nameF, tagsM, collectionM, rarityM, descriptionM,
		)
	}

	q := bleve.NewConjunctionQuery(
		termQueries...
	)

	req := bleve.NewSearchRequestOptions(q, limit, offset, false)

	result, err := ss.idx.Search(req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	cards := make([]CardPair, 0, len(result.Hits))
	for _, hit := range result.Hits {
		if c, ok := ss.byID[hit.ID]; ok {
			cards = append(cards, c)
		}
	}
	return cards, nil
}
