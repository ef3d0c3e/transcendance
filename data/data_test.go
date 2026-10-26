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

func TestMain(m *testing.M) {
	if err := os.Chdir(".."); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
