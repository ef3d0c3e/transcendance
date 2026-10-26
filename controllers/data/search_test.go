package data

import (
	"os"
	"strconv"
	"sync"
	"testing"

	"transcendance/data"
	"transcendance/localization"
	"transcendance/views"

	"github.com/stretchr/testify/require"
)

var testController *DataController

func newTestController() *DataController {
	return &DataController{
		Renderer: testController.Renderer,
		Data:     testController.Data,
	}
}

func cardNames(pairs []CardPair) []string {
	names := make([]string, len(pairs))
	for i, pair := range pairs {
		names[i] = pair.card.Name
	}
	return names
}

func TestCardSearchByTitle(t *testing.T) {
	ss, err := newTestController().GetCardSearchSet("en")
	require.NoError(t, err)

	cards, total, err := ss.Search("scotsman", 20, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(1), total)
	require.Equal(t, []string{"flying-scotsman"}, cardNames(cards))
}

func TestCardSearchIndexedByGlobalID(t *testing.T) {
	ss, err := newTestController().GetCardSearchSet("en")
	require.NoError(t, err)

	require.Len(t, ss.byID, len(testController.Data.Cards))
	for id, pair := range ss.byID {
		require.Equal(t, id, strconv.Itoa(pair.collection.ID*1000+pair.card.ID))
	}
}

func TestCardSearchRepeatedSpaces(t *testing.T) {
	ss, err := newTestController().GetCardSearchSet("en")
	require.NoError(t, err)

	single, singleTotal, err := ss.Search("ford model", 20, 0)
	require.NoError(t, err)
	repeated, repeatedTotal, err := ss.Search("  ford   model ", 20, 0)
	require.NoError(t, err)

	require.NotEmpty(t, single)
	require.Equal(t, singleTotal, repeatedTotal)
	require.Equal(t, cardNames(single), cardNames(repeated))
}

func TestCardSearchBlank(t *testing.T) {
	ss, err := newTestController().GetCardSearchSet("en")
	require.NoError(t, err)

	cards, total, err := ss.Search("   ", 20, 0)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, cards)
}

func TestCollectionSearchFuzzyName(t *testing.T) {
	ss, err := newTestController().GetCollectionSearchSet("en")
	require.NoError(t, err)

	collections, _, err := ss.Search("developmnt", 20, 0)
	require.NoError(t, err)
	require.Len(t, collections, 1)
	require.Equal(t, "dev", collections[0].Name)
}

func TestSearchSetConcurrentBuild(t *testing.T) {
	dc := newTestController()

	var wg sync.WaitGroup
	sets := make([]*CardSearchSet, 8)
	for i := range sets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ss, err := dc.GetCardSearchSet("fr")
			require.NoError(t, err)
			sets[i] = ss
			_, err = dc.GetCollectionSearchSet("fr")
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	// Every caller gets the same, single search set
	for _, ss := range sets {
		require.Same(t, sets[0], ss)
	}
}

func TestMain(m *testing.M) {
	if err := os.Chdir("../.."); err != nil {
		panic(err)
	}

	cards, err := data.LoadCards()
	if err != nil {
		panic(err)
	}
	l10n, err := localization.New()
	if err != nil {
		panic(err)
	}
	testController = &DataController{
		Renderer: views.NewRenderer(l10n, views.NewThemer()),
		Data:     cards,
	}

	os.Exit(m.Run())
}
