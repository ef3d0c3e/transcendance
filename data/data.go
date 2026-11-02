package data

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	fluent "github.com/hakastein/gofluent"
	toml "github.com/pelletier/go-toml"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"github.com/chai2010/webp"
	"golang.org/x/image/draw"
)

type Card struct {
	// ID in collection
	ID             int
	Name           string
	Locales        map[string]*fluent.Bundle
	Tags           []string
	ThumbnailPath  string
	ArtworkPath    string
	Rarity         string
	ColorPrimary   string
	ColorSecondary string
}

// Locale of the texts of cards, collections and tags missing in the requested one
const FallbackLocale = "en"

// localizedMessage falls back to FallbackLocale: a new interface language must
// not break the pages showing cards that are not translated yet
func localizedMessage(locales map[string]*fluent.Bundle, localeName string, id string) (*fluent.Bundle, *fluent.Message) {
	for _, name := range []string{localeName, FallbackLocale} {
		if loc := locales[name]; loc != nil {
			if msg, _ := loc.Message(id); msg != nil {
				return loc, msg
			}
		}
	}
	return nil, nil
}

func (c *Card) Translate(localeName string, id string) (string, error) {
	loc, msg := localizedMessage(c.Locales, localeName, id)
	if msg == nil {
		return "", fmt.Errorf("failed to find message '%s/%s' for card %s", localeName, id, c.Name)
	}
	value, err := loc.FormatPattern(msg.Value(), map[string]any{})
	if err != nil {
		return "", err
	}

	return value, nil
}

// Number of cards in a booster, unless the collection sets `booster-size`
const defaultBoosterSize = 5

type CollectionCard struct {
	Name string
}

type Collection struct {
	ID             int
	Name           string
	Locales        map[string]*fluent.Bundle
	Cards          []CollectionCard
	Tags           []string
	ColorPrimary   string
	ColorSecondary string
	ContectSheet   string
	// Cards per booster
	BoosterSize int
}

func (c *Collection) Translate(localeName string, id string) (string, error) {
	loc, msg := localizedMessage(c.Locales, localeName, id)
	if msg == nil {
		return "", fmt.Errorf("failed to find message '%s/%s' for collection %s", localeName, id, c.Name)
	}
	value, err := loc.FormatPattern(msg.Value(), map[string]any{})
	if err != nil {
		return "", err
	}

	return value, nil
}

func generateThumbnail(imagePath string, cardName string) (string, error) {
	outputPath := "data/dist/" + cardName + ".webp"
	if _, err := os.Stat(outputPath); err == nil {
		return outputPath, nil
	}

	srcFile, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = srcFile.Close() }()

	src, _, err := image.Decode(srcFile)
	if err != nil {
		return "", err
	}

	// Scale to the 330x192 card display size, keeping the aspect ratio
	thumbX := src.Bounds().Max.X
	thumbY := src.Bounds().Max.Y
	if thumbX > thumbY {
		thumbY = 192
		thumbX = (thumbY * thumbX) / src.Bounds().Max.Y
	} else {
		thumbX = 330
		thumbY = (thumbX * thumbY) / src.Bounds().Max.X
	}

	dst := image.NewRGBA(image.Rect(0, 0, thumbX, thumbY))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	if err := webp.Save(outputPath, dst, &webp.Options{Quality: 85}); err != nil {
		return "", err
	}
	return outputPath, nil
}

type Data struct {
	Cards       map[string]*Card
	Collections map[string]*Collection
	// Tag names, by locale
	Tags      map[string]*fluent.Bundle
	cardsById map[int]struct {
		*Card
		*Collection
	}
	collectionsById map[int]*Collection
}

// Load cards and collections from the `data/` directory
func LoadCards() (*Data, error) {
	tags := make(map[string]*fluent.Bundle)
	cards := make(map[string]*Card)
	collections := make(map[string]*Collection)

	traverse_tags := func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == "data/tags" {
			return nil
		}

		components := strings.Split(path, "/")

		if file.IsDir() {
			return nil
		}

		name := components[len(components)-1]
		if strings.HasSuffix(name, ".ftl") {
			localeName, _, _ := strings.Cut(name, ".")
			tags[localeName] = fluent.NewBundle(localeName)
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			resource := fluent.NewResource(string(source))
			if resource == nil {
				return fmt.Errorf("failed to parse FTL from %s", path)
			}
			if err := tags[localeName].AddResource(resource); err != nil {
				return err
			}
		}

		return err
	}
	if err := filepath.Walk("data/tags", traverse_tags); err != nil {
		return nil, err
	}
	traverse_cards := func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == "data/cards" {
			return nil
		}

		components := strings.Split(path, "/")

		if file.IsDir() {
			cardName := components[len(components)-1]
			cards[cardName] = &Card{
				Name:    cardName,
				Locales: make(map[string]*fluent.Bundle),
				Tags:    make([]string, 0),
			}
		} else {
			cardName := components[len(components)-2]
			name := components[len(components)-1]
			card := cards[cardName]
			if strings.HasSuffix(name, ".ftl") {
				localeName, _, _ := strings.Cut(name, ".")
				card.Locales[localeName] = fluent.NewBundle(localeName)
				source, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				resource := fluent.NewResource(string(source))
				if resource == nil {
					return fmt.Errorf("failed to parse FTL from %s", path)
				}
				if err := card.Locales[localeName].AddResource(resource); err != nil {
					return err
				}

				if _, ok := card.Locales[localeName].Message("title"); !ok {
					return fmt.Errorf("missing 'title' local key in %s", path)
				}
				if _, ok := card.Locales[localeName].Message("description"); !ok {
					return fmt.Errorf("missing 'description' local key in %s", path)
				}
			} else if name == "card.toml" {
				config, err := toml.LoadFile(path)
				if err != nil {
					return err
				}
				if card.Tags, err = tomlStrings(config, path, "tags"); err != nil {
					return err
				}

				for localeName, locale := range tags {
					for _, tag := range card.Tags {
						if _, ok := locale.Message(tag); ok {
							continue
						}
						return fmt.Errorf("missing tag '%s' in '%s' tags locale", tag, localeName)
					}
				}

				id, err := tomlValue[int64](config, path, "id")
				if err != nil {
					return err
				}
				// Global IDs are the collection's ID times 1000 plus the card's
				if id < 1 || id > 999 {
					return fmt.Errorf("%s: `id` must be between 1 and 999", path)
				}
				card.ID = int(id)
				if card.Rarity, err = tomlValue[string](config, path, "rarity"); err != nil {
					return err
				}
				if !slices.Contains(Rarities, card.Rarity) {
					return fmt.Errorf("%s: unknown rarity %q, expected one of %v", path, card.Rarity, Rarities)
				}

				if config.Has("color-primary") {
					if card.ColorPrimary, err = tomlValue[string](config, path, "color-primary"); err != nil {
						return err
					}
				}
				if config.Has("color-secondary") {
					if card.ColorSecondary, err = tomlValue[string](config, path, "color-secondary"); err != nil {
						return err
					}
				}
			} else if strings.HasPrefix(name, "artwork.") {
				card.ArtworkPath = path
			}
			cards[cardName] = card
		}
		return err
	}
	if err := filepath.Walk("data/cards", traverse_cards); err != nil {
		return nil, err
	}

	traverse_collections := func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == "data/collections" {
			return nil
		}

		components := strings.Split(path, "/")

		if file.IsDir() {
			collectionName := components[len(components)-1]
			collections[collectionName] = &Collection{
				Name:        collectionName,
				Locales:     make(map[string]*fluent.Bundle),
				Cards:       make([]CollectionCard, 0),
				BoosterSize: defaultBoosterSize,
			}
		} else {
			collectionName := components[len(components)-2]
			name := components[len(components)-1]
			collection := collections[collectionName]
			if strings.HasSuffix(name, ".ftl") {
				localeName, _, _ := strings.Cut(name, ".")
				collection.Locales[localeName] = fluent.NewBundle(localeName)
				source, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				resource := fluent.NewResource(string(source))
				if resource == nil {
					return fmt.Errorf("failed to parse FTL from %s", path)
				}
				if err := collection.Locales[localeName].AddResource(resource); err != nil {
					return err
				}

				if _, ok := collection.Locales[localeName].Message("title"); !ok {
					return fmt.Errorf("missing 'title' local key in %s", path)
				}
				if _, ok := collection.Locales[localeName].Message("description"); !ok {
					return fmt.Errorf("missing 'description' local key in %s", path)
				}
			} else if name == "collection.toml" {
				config, err := toml.LoadFile(path)
				if err != nil {
					return err
				}
				if collection.Tags, err = tomlStrings(config, path, "tags"); err != nil {
					return err
				}

				for localeName, locale := range tags {
					for _, tag := range collection.Tags {
						if _, ok := locale.Message(tag); ok {
							continue
						}
						return fmt.Errorf("missing tag '%s' in '%s' tags locale", tag, localeName)
					}
				}

				id, err := tomlValue[int64](config, path, "id")
				if err != nil {
					return err
				}
				collection.ID = int(id)

				if collection.ColorPrimary, err = tomlValue[string](config, path, "color-primary"); err != nil {
					return err
				}
				if collection.ColorSecondary, err = tomlValue[string](config, path, "color-secondary"); err != nil {
					return err
				}

				if config.Has("booster-size") {
					size, ok := config.Get("booster-size").(int64)
					if !ok || size <= 0 {
						return fmt.Errorf("Collection %s has an invalid 'booster-size', expected a positive integer", collection.Name)
					}
					collection.BoosterSize = int(size)
				}

				cardList, err := tomlValue[[]*toml.Tree](config, path, "cards")
				if err != nil {
					return err
				}
				for _, entry := range cardList {
					name, err := tomlValue[string](entry, path, "cards.name")
					if err != nil {
						return err
					}
					collection.Cards = append(collection.Cards, CollectionCard{Name: name})
				}
			}
			collections[collectionName] = collection
		}
		return err
	}
	if err := filepath.Walk("data/collections", traverse_collections); err != nil {
		return nil, err
	}

	cardsInCollection := make(map[string]string)
	for _, collection := range collections {
		for _, collectionCard := range collection.Cards {
			card, ok := cards[collectionCard.Name]
			if !ok {
				return nil, fmt.Errorf("collection '%s' has card '%s' but it doesn't exist", collection.Name, collectionCard.Name)
			}
			// Cards inherit the collection's colors unless they set their own
			if card.ColorPrimary == "" {
				card.ColorPrimary = collection.ColorPrimary
			}
			if card.ColorSecondary == "" {
				card.ColorSecondary = collection.ColorSecondary
			}

			cardsInCollection[card.Name] = collection.Name
		}
	}

	cardsById := make(map[int]struct {
		*Card
		*Collection
	})
	for _, card := range cards {
		collectionName, ok := cardsInCollection[card.Name]
		if !ok {
			return nil, fmt.Errorf("card '%s' is not in a collection", card.Name)
		}

		collection := collections[collectionName]
		id := card.ID + collection.ID*1000
		_, ok = cardsById[id]
		if ok {
			return nil, fmt.Errorf("card '%s' has id #%d, which is already used", card.Name, id)
		}
		cardsById[id] = struct {
			*Card
			*Collection
		}{
			card,
			collection,
		}
	}

	collectionsById := make(map[int]*Collection)
	for _, collection := range collections {
		collectionsById[collection.ID] = collection
	}

	for _, card := range cards {
		thumbnail, err := generateThumbnail(card.ArtworkPath, card.Name)
		if err != nil {
			return nil, err
		}
		card.ThumbnailPath = thumbnail
		cards[card.Name] = card
	}

	for _, collection := range collections {
		contactSheet, err := generateCollectionSheet(collection, cards)
		if err != nil {
			return nil, err
		}
		collection.ContectSheet = contactSheet
	}

	log.Printf("Loaded %d cards in %d collections", len(cards), len(collections))
	log.Print("========================================")

	return &Data{
		Cards:           cards,
		Collections:     collections,
		Tags:            tags,
		cardsById:       cardsById,
		collectionsById: collectionsById,
	}, nil
}

func (dc *Data) GetCard(id int) (*Card, *Collection) {
	col := dc.cardsById[id]
	return col.Card, col.Collection
}

func (dc *Data) GetCollection(id int) *Collection {
	return dc.collectionsById[id]
}

// Rarities a card can have, from the most to the least common
var Rarities = []string{"common", "rare", "legendary"}

// tomlValue reads a required key, the whole dotted `key` only names it in the error
func tomlValue[T any](config *toml.Tree, path string, key string) (T, error) {
	name := key[strings.LastIndex(key, ".")+1:]
	value, ok := config.Get(name).(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("%s: `%s` is missing or is not a %T", path, key, zero)
	}
	return value, nil
}

func tomlStrings(config *toml.Tree, path string, key string) ([]string, error) {
	value, ok := config.GetArray(key).([]string)
	if !ok {
		return nil, fmt.Errorf("%s: `%s` is missing or is not an array of strings", path, key)
	}
	return value, nil
}

// TagLocale falls back to FallbackLocale when the locale has no tag names
func (dc *Data) TagLocale(localeName string) *fluent.Bundle {
	if loc := dc.Tags[localeName]; loc != nil {
		return loc
	}
	return dc.Tags[FallbackLocale]
}
