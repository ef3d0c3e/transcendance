package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"transcendance/data"
	"transcendance/views"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/search/query"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DataController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
	Data     *data.Data
}

func (dc *DataController) CardGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	id, err := strconv.Atoi(c.Param("ID"))
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

	method := c.Param("METHOD")
	switch method {
	case "info":
		builder := views.PageBuilder("base", map[string]any{
			"Title": "Login",
			"User":  user,
		})
		builder.Add("card", "Content", map[string]any{
			"User":       user,
			"Card":       card,
			"Collection": collection,
			"ID":         id,
		})
		dc.Renderer.Render(c, &builder)
	case "artwork":
		c.File(card.ArtworkPath)
	case "thumbnail":
		c.File(card.ThumbnailPath)
	}
}

// Per (locale, card) entry for bleve search
type IndexableCard struct {
	Name                  string
	Description           string
	Tags                  []string
	CollectionName        string
	CollectionDescription string
	CollectionTags        []string
}

func (dc *DataController) toIndexable(locale string, card *data.Card, collection *data.Collection) (*IndexableCard, error) {

	cardLoc := card.Locales[locale]
	collectionLoc := collection.Locales[locale]
	tagsLoc := dc.Data.Tags[locale]

	// Card data
	nameMsg, _ := cardLoc.Message("name")
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
	collectionNameMsg, _ := collectionLoc.Message("name")
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

	return &IndexableCard{
		Name:                  name,
		Description:           desc,
		Tags:                  tags,
		CollectionName:        collectionName,
		CollectionDescription: collectionDesc,
		CollectionTags:        collectionTags,
	}, nil

}

func buildMapping() *mapping.IndexMappingImpl {

	im := bleve.NewIndexMapping()
	im.DefaultAnalyzer = "standard"

	card := bleve.NewDocumentMapping()

	fm := bleve.NewTextFieldMapping()

	card.AddFieldMappingsAt("Name", fm)
	card.AddFieldMappingsAt("Tags", fm)
	card.AddFieldMappingsAt("Description", fm)
	card.AddFieldMappingsAt("CollectionName", fm)
	card.AddFieldMappingsAt("CollectionTags", fm)
	card.AddFieldMappingsAt("CollectionDescription", fm)

	im.AddDocumentMapping("card", card)
	im.DefaultMapping = card
	return im
}

type SearchSet struct {
	locale string
	idx    bleve.Index
	byID   map[string]*data.Card
}

func (dc *DataController) BuildSearchSet(locale string) (*SearchSet, error) {
	idx, err := bleve.NewMemOnly(buildMapping())
	if err != nil {
		return nil, fmt.Errorf("creating index: %w", err)
	}

	ss := &SearchSet{
		locale: locale,
		idx:    idx,
		byID:   make(map[string]*data.Card, len(dc.Data.Cards)),
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
			ss.byID[card.Name] = card

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

func (ss *SearchSet) Search(term string, limit int) ([]*data.Card, error) {
	q := bleve.NewDisjunctionQuery(
		fieldQuery("Name", term, 8),
		fieldQuery("Tags", term, 4),
		fieldQuery("Description", term, 2),
		fieldQuery("CollectionName", term, 1),
		fieldQuery("CollectionTags", term, 0.5),
		fieldQuery("CollectionDescription", term, 0.25),
	)

	req := bleve.NewSearchRequestOptions(q, limit, 0, false)

	result, err := ss.idx.Search(req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	cards := make([]*data.Card, 0, len(result.Hits))
	for _, hit := range result.Hits {
		if c, ok := ss.byID[hit.ID]; ok {
			cards = append(cards, c)
		}
	}
	return cards, nil
}

func (dc *DataController) CardSearchGet(c *gin.Context) {
	q := c.Query("search")

	ss, err := dc.BuildSearchSet("fr")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	results, err := ss.Search(q, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": results,
	})
}
