package data

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectionDefaultBoosterSize(t *testing.T) {
	cards, err := LoadCards()
	require.NoError(t, err)

	// The dev collection doesn't set `booster-size`
	collection := cards.GetCollection(1)
	require.NotNil(t, collection)
	require.Equal(t, defaultBoosterSize, collection.BoosterSize)
}

func TestTranslationsFallBackToEnglish(t *testing.T) {
	cards, err := LoadCards()
	require.NoError(t, err)

	card, collection := cards.GetCard(1001)
	require.NotNil(t, card)
	english, err := card.Translate("en", "title")
	require.NoError(t, err)
	spanish, err := card.Translate("es", "title")
	require.NoError(t, err)
	require.Equal(t, english, spanish)

	french, err := card.Translate("fr", "title")
	require.NoError(t, err)
	require.NotEmpty(t, french)

	title, err := collection.Translate("es", "title")
	require.NoError(t, err)
	require.NotEmpty(t, title)
	require.NotNil(t, cards.TagLocale("es"))

	_, err = card.Translate("es", "no-such-message")
	require.Error(t, err)
}

func TestMain(m *testing.M) {
	if err := os.Chdir(".."); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
