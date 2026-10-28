package controllers

import (
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// Maximum length of a message, in characters
	shoutboxMaxLength = 500
	// Messages sent to clients when they connect
	shoutboxHistorySize = 50
	// Largest frame accepted from clients, a maximum length message fits even with every character escaped
	shoutboxReadLimit = 8192
	// Open connections allowed per user, one for each tab
	shoutboxMaxConnections = 8
	// Messages queued for a client before it is considered too slow and dropped
	shoutboxSendBuffer = 64

	// Rate limit: a user can send `shoutboxBurst` messages at once, then one every `shoutboxRefill`
	shoutboxBurst  = 5
	shoutboxRefill = 2 * time.Second

	// Combining marks kept after a character, the rest are dropped to prevent zalgo text
	shoutboxMaxCombining = 3
)

// Error codes sent to clients, each one has a `shoutbox-error-<code>` translation
const (
	shoutboxErrorEmpty       = "empty"
	shoutboxErrorTooLong     = "too-long"
	shoutboxErrorRateLimited = "rate-limited"
	shoutboxErrorInternal    = "internal"
)

// Clean up a message body, returns an error code if it can't be posted
//
// Line breaks and tabs become spaces, other control and invisible formatting characters (e.g. bidirectional
// overrides) are removed, as well as excessive combining marks
func sanitizeShoutboxMessage(body string) (string, string) {
	if !utf8.ValidString(body) {
		body = strings.ToValidUTF8(body, "\uFFFD")
	}

	var b strings.Builder
	b.Grow(len(body))
	combining := 0
	for _, r := range body {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			r = ' '
		case unicode.Is(unicode.Cc, r) || (unicode.Is(unicode.Cf, r) && r != '\u200D'):
			// Zero width joiners are kept for emojis
			continue
		}
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
			combining++
			if combining > shoutboxMaxCombining {
				continue
			}
		} else {
			combining = 0
		}
		b.WriteRune(r)
	}

	clean := strings.TrimSpace(b.String())
	if clean == "" {
		return "", shoutboxErrorEmpty
	}
	if utf8.RuneCountInString(clean) > shoutboxMaxLength {
		return "", shoutboxErrorTooLong
	}
	return clean, ""
}

// Token bucket rate limiter
type shoutboxBucket struct {
	tokens float64
	last   time.Time
}

func newShoutboxBucket(now time.Time) *shoutboxBucket {
	return &shoutboxBucket{tokens: shoutboxBurst, last: now}
}

func (b *shoutboxBucket) refill(now time.Time) {
	b.tokens = min(shoutboxBurst, b.tokens+float64(now.Sub(b.last))/float64(shoutboxRefill))
	b.last = now
}

// Take a token, otherwise returns how long until one is available
func (b *shoutboxBucket) take(now time.Time) (bool, time.Duration) {
	b.refill(now)
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) * float64(shoutboxRefill))
}

// Connected shoutbox clients
type shoutboxHub struct {
	mu      sync.Mutex
	clients map[*shoutboxClient]struct{}
	// Open connections per user
	connections map[uint]int
	// Rate limits are per user, opening more connections doesn't give more messages
	buckets map[uint]*shoutboxBucket
}

func newShoutboxHub() *shoutboxHub {
	return &shoutboxHub{
		clients:     make(map[*shoutboxClient]struct{}),
		connections: make(map[uint]int),
		buckets:     make(map[uint]*shoutboxBucket),
	}
}

// Add a client, fails when its user has too many connections
func (h *shoutboxHub) register(client *shoutboxClient) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[client.userID] >= shoutboxMaxConnections {
		return false
	}
	h.clients[client] = struct{}{}
	h.connections[client.userID]++

	// Full buckets of disconnected users are the same as new ones
	now := time.Now()
	for id, bucket := range h.buckets {
		if h.connections[id] == 0 {
			bucket.refill(now)
			if bucket.tokens >= shoutboxBurst {
				delete(h.buckets, id)
			}
		}
	}
	return true
}

func (h *shoutboxHub) unregister(client *shoutboxClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; !ok {
		return
	}
	delete(h.clients, client)
	if h.connections[client.userID]--; h.connections[client.userID] <= 0 {
		delete(h.connections, client.userID)
	}
}

// Take one of the user's message tokens
func (h *shoutboxHub) allow(userID uint) (bool, time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	bucket, ok := h.buckets[userID]
	if !ok {
		bucket = newShoutboxBucket(now)
		h.buckets[userID] = bucket
	}
	return bucket.take(now)
}

// Queue a payload for every client, slow clients that can't keep up are disconnected
func (h *shoutboxHub) broadcast(payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		if !client.queue(payload) {
			delete(h.clients, client)
			if h.connections[client.userID]--; h.connections[client.userID] <= 0 {
				delete(h.connections, client.userID)
			}
		}
	}
}

// A websocket connection to the shoutbox
type shoutboxClient struct {
	userID uint
	send   chan []byte
	// Closed when the client is dropped for being too slow
	dropped   chan struct{}
	closeOnce sync.Once
}

func newShoutboxClient(userID uint) *shoutboxClient {
	return &shoutboxClient{
		userID:  userID,
		send:    make(chan []byte, shoutboxSendBuffer),
		dropped: make(chan struct{}),
	}
}

// Queue a payload without blocking, false if the client's buffer is full
func (c *shoutboxClient) queue(payload []byte) bool {
	select {
	case c.send <- payload:
		return true
	default:
		c.closeOnce.Do(func() { close(c.dropped) })
		return false
	}
}
