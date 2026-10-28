package controllers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBoosterPriceDefault(t *testing.T) {
	require.Equal(t, uint(defaultBoosterPrice), boosterPrice(4242))
}

func TestBoosterPriceOverride(t *testing.T) {
	boosterPrices[42] = 1
	defer delete(boosterPrices, 42)

	require.Equal(t, uint(1), boosterPrice(42))
	require.Equal(t, uint(defaultBoosterPrice), boosterPrice(43))
}

func TestTopupAmountKnown(t *testing.T) {
	require.NotEmpty(t, topupPacks)
	for _, pack := range topupPacks {
		amount, ok := topupAmount(pack.ID)
		require.True(t, ok)
		require.Equal(t, pack.Amount, amount)
		require.NotZero(t, amount)
	}
}

func TestTopupAmountUnknown(t *testing.T) {
	for _, id := range []string{"", "diamond", "SMALL", "small "} {
		_, ok := topupAmount(id)
		require.Falsef(t, ok, "pack %q must not be on sale", id)
	}
}
