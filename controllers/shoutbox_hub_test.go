package controllers

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShoutboxSanitize(t *testing.T) {
	cases := []struct {
		in   string
		out  string
		code string
	}{
		{"hello", "hello", ""},
		{"  padded\t", "padded", ""},
		{"two\nlines\r\n", "two lines", ""},
		{"\x00nul\x07bell\x1b[31m", "nulbell[31m", ""},
		// Bidirectional overrides and zero width spaces are removed, joiners are kept for emojis
		{"a‮b​c", "abc", ""},
		{"\U0001F468‍\U0001F469", "\U0001F468‍\U0001F469", ""},
		{"ź̂̃̄̅", "ź̂̃", ""},
		{"\xffbad", "�bad", ""},
		{"", "", shoutboxErrorEmpty},
		{" \n\t​ ", "", shoutboxErrorEmpty},
		{strings.Repeat("é", shoutboxMaxLength), strings.Repeat("é", shoutboxMaxLength), ""},
		{strings.Repeat("a", shoutboxMaxLength+1), "", shoutboxErrorTooLong},
		// Removed characters don't count
		{strings.Repeat("a​", shoutboxMaxLength), strings.Repeat("a", shoutboxMaxLength), ""},
	}
	for _, c := range cases {
		out, code := sanitizeShoutboxMessage(c.in)
		require.Equal(t, c.code, code, "%q", c.in)
		require.Equal(t, c.out, out, "%q", c.in)
	}
}

func TestShoutboxBucket(t *testing.T) {
	now := time.Now()
	bucket := newShoutboxBucket(now)
	for range shoutboxBurst {
		ok, _ := bucket.take(now)
		require.True(t, ok)
	}
	ok, wait := bucket.take(now)
	require.False(t, ok)
	require.Equal(t, shoutboxRefill, wait)

	// Half a token
	ok, wait = bucket.take(now.Add(shoutboxRefill / 2))
	require.False(t, ok)
	require.Equal(t, shoutboxRefill/2, wait)

	ok, _ = bucket.take(now.Add(shoutboxRefill))
	require.True(t, ok)

	// Never refills past the burst
	later := now.Add(time.Hour)
	for range shoutboxBurst {
		ok, _ = bucket.take(later)
		require.True(t, ok)
	}
	ok, _ = bucket.take(later)
	require.False(t, ok)
}

func TestShoutboxHubConnections(t *testing.T) {
	hub := newShoutboxHub()
	clients := make([]*shoutboxClient, shoutboxMaxConnections)
	for i := range clients {
		clients[i] = newShoutboxClient(1)
		require.True(t, hub.register(clients[i]))
	}
	require.False(t, hub.register(newShoutboxClient(1)))
	// Other users are unaffected
	require.True(t, hub.register(newShoutboxClient(2)))

	hub.unregister(clients[0])
	require.True(t, hub.register(newShoutboxClient(1)))
}

func TestShoutboxHubDropsSlowClients(t *testing.T) {
	hub := newShoutboxHub()
	slow := newShoutboxClient(1)
	fast := newShoutboxClient(2)
	require.True(t, hub.register(slow))
	require.True(t, hub.register(fast))

	for i := range shoutboxSendBuffer + 1 {
		hub.broadcast([]byte{byte(i)})
		<-fast.send
	}
	select {
	case <-slow.dropped:
	default:
		t.Fatal("slow client was not dropped")
	}
	select {
	case <-fast.dropped:
		t.Fatal("fast client was dropped")
	default:
	}
	require.NotContains(t, hub.clients, slow)
	require.Zero(t, hub.connections[1])
}
