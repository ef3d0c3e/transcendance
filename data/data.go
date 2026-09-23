package data

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	fluent "github.com/hakastein/gofluent"
	toml "github.com/pelletier/go-toml"
)

type Card struct {
	// ID in collection
	ID             int
	Name           string
	Locales        map[string]*fluent.Bundle
	Tags           []string
	ThumbnailPath string
	ArtworkPath   string
}

type CollectionCard struct {
	name string
}

type Collection struct {
	ID      int
	Name    string
	Locales map[string]*fluent.Bundle
	Cards   []CollectionCard
}

func generate_thumbnail(path string) (string, error) {
	// TODO
	return path, nil
}

type Data struct {
	cards       map[string]*Card
	collections map[string]*Collection
	cardsById   map[int]struct {
		*Card
		*Collection
	}
}

// Load cards and collections from the `data/` directory
func LoadCards() (*Data, error) {
	cards := make(map[string]*Card)
	collections := make(map[string]*Collection)

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
				localeName := strings.Split(name, ".")[0]
				card.Locales[localeName] = fluent.NewBundle("en")
				card.Locales[localeName].AddResource(fluent.NewResource(path))
			} else if name == "card.toml" {
				config, err := toml.LoadFile(path)
				if err != nil {
					return err
				}
				card.Tags = config.GetArray("tags").([]string)
				card.ID = int(config.Get("id").(int64))
			} else if strings.HasPrefix(name, "artwork.") {
				card.ArtworkPath = path
				thumbnail, err := generate_thumbnail(path)
				if err != nil {
					return err
				}
				card.ThumbnailPath = thumbnail
			}
			cards[cardName] = card
		}
		return err
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
				Name:    collectionName,
				Locales: make(map[string]*fluent.Bundle),
				Cards:   make([]CollectionCard, 0),
			}
		} else {
			collectionName := components[len(components)-2]
			name := components[len(components)-1]
			collection := collections[collectionName]
			if strings.HasSuffix(name, ".ftl") {
				localeName := strings.Split(name, ".")[0]
				collection.Locales[localeName] = fluent.NewBundle(path)
				collection.Locales[localeName].AddResource(fluent.NewResource(path))
			} else if name == "collection.toml" {
				config, err := toml.LoadFile(path)
				if err != nil {
					return err
				}
				collection.ID = int(config.Get("id").(int64))

				cardList := config.Get("cards").([]*toml.Tree)
				for _, entry := range cardList {
					card := CollectionCard{
						name: entry.Get("name").(string),
					}
					collection.Cards = append(collection.Cards, card)
				}
			}
			collections[collectionName] = collection
		}
		return err
	}
	if err := filepath.Walk("data/cards", traverse_cards); err != nil {
		return nil, err
	}
	if err := filepath.Walk("data/collections", traverse_collections); err != nil {
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
			card, ok := cards[collectionCard.name]
			if !ok {
				return nil, fmt.Errorf("Collection '%s' has card '%s' but it doesn't exist", collection.Name, collectionCard.name)
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
		cardsById[id] = struct {*Card; *Collection}{
			card,
			collection,
		}
	}

	return &Data{
		cards:       cards,
		collections: collections,
		cardsById:   cardsById,
	}, nil
}

func (dc *Data) GetCard(id int) (*Card, *Collection) {
	col := dc.cardsById[id]
	return col.Card, col.Collection
}
