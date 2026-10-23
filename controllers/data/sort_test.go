package data

import (
	"testing"
	"time"

	"transcendance/data"

	fluent "github.com/hakastein/gofluent"
)

// A stack whose card and collection both carry an English title, so the
// title-based sorts have something to compare
func sortStack(id uint, count int64, cardTitle string, rarity string, collectionTitle string) cardStack {
	titled := func(name string) map[string]*fluent.Bundle {
		bundle := fluent.NewBundle("en")
		if err := bundle.AddResource(fluent.NewResource("title = " + name + "\n")); err != nil {
			panic(err)
		}
		return map[string]*fluent.Bundle{"en": bundle}
	}
	card := testCard(cardTitle, rarity)
	card.Locales = titled(cardTitle)
	return cardStack{
		CardID: id,
		Count:  count,
		// Equal timestamps: only the sort itself can order these stacks
		FirstAcquired: time.Unix(0, 0),
		LastAcquired:  time.Unix(0, 0),
		card:          card,
		collection: &data.Collection{
			Locales: titled(collectionTitle),
		},
	}
}

// Card titles of the stacks, in the order they stand
func stackTitles(stacks []cardStack) []string {
	titles := make([]string, len(stacks))
	for i, stack := range stacks {
		titles[i] = stack.card.Name
	}
	return titles
}

func TestSortCardStacksByRarity(t *testing.T) {
	// Card IDs run opposite to the expected order, so the tie-break can't decide it
	stacks := []cardStack{
		sortStack(1, 1, "legendary", "legendary", "dev"),
		sortStack(2, 1, "common", "common", "dev"),
		sortStack(3, 1, "rare", "rare", "dev"),
	}

	sortCardStacks(stacks, inventorySortRarity, "en", nil)

	want := []string{"legendary", "rare", "common"}
	got := stackTitles(stacks)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order %v, want %v", got, want)
		}
	}
}

func TestSortCardStacksByCount(t *testing.T) {
	stacks := []cardStack{
		sortStack(1, 1, "one", "common", "dev"),
		sortStack(2, 30, "thirty", "common", "dev"),
		sortStack(3, 5, "five", "common", "dev"),
	}

	sortCardStacks(stacks, inventorySortCount, "en", nil)

	want := []string{"thirty", "five", "one"}
	got := stackTitles(stacks)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order %v, want %v", got, want)
		}
	}
}

func TestSortTradeupStacks(t *testing.T) {
	// Collection first, then rarity, then how many copies to spend
	stacks := []cardStack{
		sortStack(4, 1, "alpha-rare", "rare", "Alpha"),
		sortStack(3, 30, "alpha-common-big", "common", "Alpha"),
		sortStack(2, 2, "zeta-common", "common", "Zeta"),
		sortStack(1, 5, "alpha-common-small", "common", "Alpha"),
	}

	sortTradeupStacks(stacks, "en")

	want := []string{"alpha-rare", "alpha-common-big", "alpha-common-small", "zeta-common"}
	got := stackTitles(stacks)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order %v, want %v", got, want)
		}
	}
}
