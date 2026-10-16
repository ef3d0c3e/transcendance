package data

import (
	"fmt"
	"strings"
	"transcendance/data"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/analysis/char/asciifolding"
	"github.com/blevesearch/bleve/analysis/token/lowercase"
	"github.com/blevesearch/bleve/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/search/query"
)

type CardPair struct {
	card       *data.Card
	collection *data.Collection
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

// Per (locale, collection) entry for bleve search
type IndexableCollection struct {
	Name        string
	Description string
	Tags        []string
}

// Search set for cards
type CardSearchSet struct {
	locale string
	idx    bleve.Index
	byName map[string]CardPair
}

// Search set for collections
type CollectionSearchSet struct {
	locale string
	idx    bleve.Index
	byName map[string]*data.Collection
}

// Build IndexableCard for a (card, collection) pair
func (dc *DataController) toIndexableCard(locale string, card *data.Card, collection *data.Collection) (*IndexableCard, error) {
	tagsLoc := dc.Data.Tags[locale]

	// Card data
	name, err := card.Translate(locale, "title")
	if err != nil {
		return nil, err
	}
	desc, err := card.Translate(locale, "description")
	if err != nil {
		return nil, err
	}

	// Tags
	tags := make([]string, 0)
	for _, tag := range card.Tags {
		tagMsg, _ := tagsLoc.Message(tag)
		tagString, err := tagsLoc.FormatPattern(tagMsg.Value(), map[string]any{})
		if err != nil {
			return nil, err
		}
		tags = append(tags, tagString)
	}

	// Collection data
	collectionName, err := collection.Translate(locale, "title")
	if err != nil {
		return nil, err
	}
	collectionDesc, err := collection.Translate(locale, "description")
	if err != nil {
		return nil, err
	}

	// Tags
	collectionTags := make([]string, 0)
	for _, tag := range collection.Tags {
		tagMsg, _ := tagsLoc.Message(tag)
		collectionTagstring, err := tagsLoc.FormatPattern(tagMsg.Value(), map[string]any{})
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

// Build IndexableCollection for a collection
func (dc *DataController) toIndexableCollection(locale string, collection *data.Collection) (*IndexableCollection, error) {
	tagsLoc := dc.Data.Tags[locale]

	name, err := collection.Translate(locale, "title")
	if err != nil {
		return nil, err
	}
	desc, err := collection.Translate(locale, "description")
	if err != nil {
		return nil, err
	}

	// Tags
	tags := make([]string, 0)
	for _, tag := range collection.Tags {
		tagMsg, _ := tagsLoc.Message(tag)
		tagString, err := tagsLoc.FormatPattern(tagMsg.Value(), map[string]any{})
		if err != nil {
			return nil, err
		}
		tags = append(tags, tagString)
	}

	return &IndexableCollection{
		Name:        name,
		Description: desc,
		Tags:        tags,
	}, nil
}

// search mapping for cards
func buildCardMapping() (*mapping.IndexMappingImpl, error) {
	im := bleve.NewIndexMapping()
	im.DefaultAnalyzer = "standard"
	if err := im.AddCustomCharFilter("ascii_fold", map[string]interface{}{
		"type": asciifolding.Name,
	}); err != nil {
		return nil, fmt.Errorf("failed to create ascii_fold filter: %w", err)
	}

	if err := im.AddCustomAnalyzer("accent_insensitive", map[string]interface{}{
		"type":          custom.Name,
		"tokenizer":     unicode.Name,
		"char_filters":  []string{"ascii_fold"},
		"token_filters": []string{lowercase.Name},
	}); err != nil {
		return nil, fmt.Errorf("failed to create accent_insensitive analyzer: %w", err)
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
	return im, nil
}

// search mapping for collections
func buildCollectionMapping() (*mapping.IndexMappingImpl, error) {

	im := bleve.NewIndexMapping()
	im.DefaultAnalyzer = "standard"
	if err := im.AddCustomCharFilter("ascii_fold", map[string]interface{}{
		"type": asciifolding.Name,
	}); err != nil {
		return nil, fmt.Errorf("failed to create ascii_fold filter: %w", err)
	}

	if err := im.AddCustomAnalyzer("accent_insensitive", map[string]interface{}{
		"type":          custom.Name,
		"tokenizer":     unicode.Name,
		"char_filters":  []string{"ascii_fold"},
		"token_filters": []string{lowercase.Name},
	}); err != nil {
		return nil, fmt.Errorf("failed to create accent_insensitive analyzer: %w", err)
	}

	collection := bleve.NewDocumentMapping()
	fm := bleve.NewTextFieldMapping()
	fm.Analyzer = "accent_insensitive"

	collection.AddFieldMappingsAt("Name", fm)
	collection.AddFieldMappingsAt("Tags", fm)
	collection.AddFieldMappingsAt("Description", fm)

	im.AddDocumentMapping("collection", collection)
	im.DefaultMapping = collection
	return im, nil
}

// Build the search set for cards
func (dc *DataController) BuildCardSearchSet(locale string) (*CardSearchSet, error) {
	im, err := buildCardMapping()
	if err != nil {
		return nil, err
	}
	idx, err := bleve.NewMemOnly(im)
	if err != nil {
		return nil, fmt.Errorf("creating index: %w", err)
	}

	ss := &CardSearchSet{
		locale: locale,
		idx:    idx,
		byName: make(map[string]CardPair, len(dc.Data.Cards)),
	}

	batch := idx.NewBatch()
	for _, collection := range dc.Data.Collections {
		for _, colCard := range collection.Cards {
			card := dc.Data.Cards[colCard.Name]

			ic, err := dc.toIndexableCard(locale, card, collection)
			id := card.ID + collection.ID*1000
			if err != nil {
				return nil, fmt.Errorf("indexing card %d: %w", id, err)
			}
			if err := batch.Index(card.Name, ic); err != nil {
				return nil, err
			}
			ss.byName[card.Name] = CardPair{
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

// Build the search set for collections
func (dc *DataController) BuildCollectionSearchSet(locale string) (*CollectionSearchSet, error) {
	im, err := buildCollectionMapping()
	if err != nil {
		return nil, err
	}
	idx, err := bleve.NewMemOnly(im)
	if err != nil {
		return nil, fmt.Errorf("creating index: %w", err)
	}

	ss := &CollectionSearchSet{
		locale: locale,
		idx:    idx,
		byName: make(map[string]*data.Collection, len(dc.Data.Collections)),
	}

	batch := idx.NewBatch()
	for _, collection := range dc.Data.Collections {
		ic, err := dc.toIndexableCollection(locale, collection)
		if err != nil {
			return nil, fmt.Errorf("while indexing collection %d: %w", collection.ID, err)
		}
		if err := batch.Index(collection.Name, ic); err != nil {
			return nil, fmt.Errorf("while indexing collection %d: %w", collection.ID, err)
		}
		ss.byName[collection.Name] = collection
	}

	if err := idx.Batch(batch); err != nil {
		return nil, fmt.Errorf("committing batch: %w", err)
	}

	return ss, nil
}

// Search function for cards
func (ss *CardSearchSet) Search(search string, limit int, offset int) ([]CardPair, uint64, error) {
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
		termQueries...,
	)

	req := bleve.NewSearchRequestOptions(q, limit, offset, false)

	result, err := ss.idx.Search(req)
	if err != nil {
		return nil, 0, fmt.Errorf("search: %w", err)
	}

	cards := make([]CardPair, 0, len(result.Hits))
	for _, hit := range result.Hits {
		if c, ok := ss.byName[hit.ID]; ok {
			cards = append(cards, c)
		}
	}
	return cards, result.Total, nil
}

// Search function for collections
func (ss *CollectionSearchSet) Search(search string, limit int, offset int) ([]*data.Collection, uint64, error) {
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
		nameF.SetField("nName")
		nameF.SetBoost(6)
		nameF.SetFuzziness(2)
		nameF.SetPrefix(1)

		tagsM := bleve.NewMatchQuery(term)
		tagsM.SetField("Tags")
		tagsM.SetBoost(4)

		descriptionM := bleve.NewMatchQuery(term)
		descriptionM.SetField("Description")
		descriptionM.SetBoost(1)

		termQueries[i] = bleve.NewDisjunctionQuery(
			nameM, nameP, nameF, tagsM, descriptionM,
		)
	}

	q := bleve.NewConjunctionQuery(
		termQueries...,
	)

	req := bleve.NewSearchRequestOptions(q, limit, offset, false)
	req.AddFacet("collections", bleve.NewFacetRequest("collection_id", 50))

	result, err := ss.idx.Search(req)
	if err != nil {
		return nil, 0, fmt.Errorf("search: %w", err)
	}

	collections := make([]*data.Collection, 0, len(result.Hits))
	for _, hit := range result.Hits {
		if c, ok := ss.byName[hit.ID]; ok {
			collections = append(collections, c)
		}
	}
	return collections, result.Total, nil
}

func (dc *DataController) GetCardSearchSet(locale string) (*CardSearchSet, error) {
	if dc.CardSearch == nil {
		dc.CardSearch = map[string]*CardSearchSet{}
	}

	if dc.CardSearch[locale] == nil {
		ss, err := dc.BuildCardSearchSet(locale)
		if err != nil {
			return nil, fmt.Errorf("failed to build search set for locale '%s': %w", locale, err)
		}
		dc.CardSearch[locale] = ss
	}
	return dc.CardSearch[locale], nil
}

func (dc *DataController) GetCollectionSearchSet(locale string) (*CollectionSearchSet, error) {
	if dc.CollectionSearch == nil {
		dc.CollectionSearch = map[string]*CollectionSearchSet{}
	}

	if dc.CollectionSearch[locale] == nil {
		ss, err := dc.BuildCollectionSearchSet(locale)
		if err != nil {
			return nil, fmt.Errorf("failed to build search set for locale '%s': %w", locale, err)
		}
		dc.CollectionSearch[locale] = ss
	}
	return dc.CollectionSearch[locale], nil
}
