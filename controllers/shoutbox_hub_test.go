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
		{"a\u202Eb\u200Bc", "abc", ""},
		{"\U0001F468\u200D\U0001F469", "\U0001F468\u200D\U0001F469", ""},
		{"z\u0301\u0302\u0303\u0304\u0305", "z\u0301\u0302\u0303", ""},
		{"\xffbad", "\uFFFDbad", ""},
		{"", "", shoutboxErrorEmpty},
		{" \n\t\u200B ", "", shoutboxErrorEmpty},
		{strings.Repeat("é", shoutboxMaxLength), strings.Repeat("é", shoutboxMaxLength), ""},
		{strings.Repeat("a", shoutboxMaxLength+1), "", shoutboxErrorTooLong},
		// Removed characters don't count
		{strings.Repeat("a\u200B", shoutboxMaxLength), strings.Repeat("a", shoutboxMaxLength), ""},
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
		require.True(t, hub.register(clients[i], nil))
	}
	require.False(t, hub.register(newShoutboxClient(1), nil))
	// Other users are unaffected
	require.True(t, hub.register(newShoutboxClient(2), nil))

	hub.unregister(clients[0])
	require.True(t, hub.register(newShoutboxClient(1), nil))
}

func TestShoutboxHubDropsSlowClients(t *testing.T) {
	hub := newShoutboxHub()
	slow := newShoutboxClient(1)
	fast := newShoutboxClient(2)
	require.True(t, hub.register(slow, nil))
	require.True(t, hub.register(fast, nil))

	for i := range shoutboxSendBuffer + 1 {
		hub.broadcast(3, []byte{byte(i)})
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
	require.NotContains(t, hub.users, uint(1))
	require.Contains(t, hub.users[2], fast)
}

// Pending payloads of a client
func drain(client *shoutboxClient) []byte {
	var got []byte
	for {
		select {
		case payload := <-client.send:
			got = append(got, payload...)
		default:
			return got
		}
	}
}

func TestShoutboxHubIgnores(t *testing.T) {
	hub := newShoutboxHub()
	ignorer := newShoutboxClient(1)
	expired := newShoutboxClient(2)
	other := newShoutboxClient(3)
	require.True(t, hub.register(ignorer, map[uint]time.Time{4: time.Now().Add(time.Hour)}))
	require.True(t, hub.register(expired, map[uint]time.Time{4: time.Now().Add(-time.Second)}))
	require.True(t, hub.register(other, nil))

	hub.broadcast(4, []byte("a"))
	require.Empty(t, drain(ignorer))
	require.Equal(t, []byte("a"), drain(expired))
	require.Equal(t, []byte("a"), drain(other))

	// Lifted
	hub.setIgnores(1, nil)
	hub.broadcast(4, []byte("b"))
	require.Equal(t, []byte("b"), drain(ignorer))

	// Disconnected users' ignores aren't kept
	hub.unregister(other)
	hub.setIgnores(3, map[uint]time.Time{4: time.Now().Add(time.Hour)})
	require.NotContains(t, hub.ignores, uint(3))
}

func TestShoutboxHubDeliver(t *testing.T) {
	hub := newShoutboxHub()
	tabs := []*shoutboxClient{newShoutboxClient(1), newShoutboxClient(1)}
	recipient := newShoutboxClient(2)
	other := newShoutboxClient(3)
	for _, client := range append(tabs, recipient, other) {
		require.True(t, hub.register(client, nil))
	}

	hub.deliver([]byte("a"), 1, 2)
	require.Equal(t, []byte("a"), drain(tabs[0]))
	require.Equal(t, []byte("a"), drain(tabs[1]))
	require.Equal(t, []byte("a"), drain(recipient))
	require.Empty(t, drain(other))

	// Once per client when both users are the same
	hub.deliver([]byte("b"), 1, 1)
	require.Equal(t, []byte("b"), drain(tabs[0]))

	require.False(t, hub.viewing(2, 1))
	hub.setViewing(recipient, 1)
	require.True(t, hub.viewing(2, 1))
	require.False(t, hub.viewing(1, 2))
	hub.setViewing(recipient, 0)
	require.False(t, hub.viewing(2, 1))
}
