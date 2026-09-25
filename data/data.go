package data

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
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
	ID            int
	Name          string
	Locales       map[string]*fluent.Bundle
	Tags          []string
	ThumbnailPath string
	ArtworkPath   string
}

type CollectionCard struct {
	Name string
}

type Collection struct {
	ID          int
	Name        string
	Locales     map[string]*fluent.Bundle
	Cards       []CollectionCard
	Tags        []string
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

	dst := image.NewRGBA(image.Rect(0, 0, 600, 400))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	if err := webp.Save(outputPath, dst, &webp.Options{Quality: 75}); err != nil {
		return "", err
	}
	return outputPath, nil
}

type Data struct {
	Cards       map[string]*Card
	Collections map[string]*Collection
	// Tags locales
	Tags      map[string]*fluent.Bundle
	cardsById map[int]struct {
		*Card
		*Collection
	}
}

// Load cards and collections from the `data/` directory
func LoadCards() (*Data, error) {
	cards := make(map[string]*Card)
	collections := make(map[string]*Collection)
	tags := make(map[string]*fluent.Bundle)

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
					return fmt.Errorf("Failed to parse FTL from %s", path)
				}
				if err := card.Locales[localeName].AddResource(resource); err != nil {
					return err
				}
			} else if name == "card.toml" {
				config, err := toml.LoadFile(path)
				if err != nil {
					return err
				}
				card.Tags = config.GetArray("tags").([]string)
				card.ID = int(config.Get("id").(int64))
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
	// Build collection
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
				Name:    collectionName,
				Locales: make(map[string]*fluent.Bundle),
				Cards:   make([]CollectionCard, 0),
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
					return fmt.Errorf("Failed to parse FTL from %s", path)
				}
				if err := collection.Locales[localeName].AddResource(resource); err != nil {
					return err
				}
			} else if name == "collection.toml" {
				config, err := toml.LoadFile(path)
				if err != nil {
					return err
				}
				collection.Tags = config.GetArray("tags").([]string)
				collection.ID = int(config.Get("id").(int64))

				cardList := config.Get("cards").([]*toml.Tree)
				for _, entry := range cardList {
					card := CollectionCard{
						Name: entry.Get("name").(string),
					}
					collection.Cards = append(collection.Cards, card)
				}
			}
			collections[collectionName] = collection
		}
		return err
	}
	if err := filepath.Walk("data/collections", traverse_collections); err != nil {
		return nil, err
	}
	// Build tags locales
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
				return fmt.Errorf("Failed to parse FTL from %s", path)
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

	log.Print("===========================================")
	log.Print("cards=", cards)
	log.Print("collections=", collections)
	log.Print("===========================================")

	// Verify every cards in collections exist
	cardsInCollection := make(map[string]string)
	for _, collection := range collections {
		for _, collectionCard := range collection.Cards {
			card, ok := cards[collectionCard.Name]
			if !ok {
				return nil, fmt.Errorf("Collection '%s' has card '%s' but it doesn't exist", collection.Name, collectionCard.Name)
			}
			cardsInCollection[card.Name] = collection.Name
		}
	}

	// Verify every card is in a collection, then build the id -> card lookup
	cardsById := make(map[int]struct {
		*Card
		*Collection
	})
	for _, card := range cards {
		collectionName, ok := cardsInCollection[card.Name]
		if !ok {
			return nil, fmt.Errorf("Card '%s' is not in a collection", card.Name)
		}

		collection := collections[collectionName]
		id := card.ID + collection.ID*1000
		_, ok = cardsById[id]
		if ok {
			return nil, fmt.Errorf("Card '%s' has id #%d, which is already used", card.Name, id)
		}
		cardsById[id] = struct {
			*Card
			*Collection
		}{
			card,
			collection,
		}
	}

	// Generate thumbnails
	for _, card := range cards {
		thumbnail, err := generateThumbnail(card.ArtworkPath, card.Name)
		if err != nil {
			return nil, err
		}
		card.ThumbnailPath = thumbnail
		cards[card.Name] = card
	}

	return &Data{
		Cards:       cards,
		Collections: collections,
		Tags:        tags,
		cardsById:   cardsById,
	}, nil
}

func (dc *Data) GetCard(id int) (*Card, *Collection) {
	col := dc.cardsById[id]
	return col.Card, col.Collection
}
