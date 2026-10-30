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
	// Private messages and ignores
	shoutboxErrorUserNotFound = "user-not-found"
	shoutboxErrorSelf         = "self"
	shoutboxErrorBlocked      = "blocked"
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
	mu sync.Mutex
	// Open connections per user, one for each tab
	users map[uint]map[*shoutboxClient]struct{}
	// Rate limits are per user, opening more connections doesn't give more messages
	buckets map[uint]*shoutboxBucket
	// Users ignored by each connected user, until when
	ignores map[uint]map[uint]time.Time
}

func newShoutboxHub() *shoutboxHub {
	return &shoutboxHub{
		users:   make(map[uint]map[*shoutboxClient]struct{}),
		buckets: make(map[uint]*shoutboxBucket),
		ignores: make(map[uint]map[uint]time.Time),
	}
}

// Add a client with its user's ignores, fails when its user has too many connections
func (h *shoutboxHub) register(client *shoutboxClient, ignores map[uint]time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.users[client.userID]
	if len(clients) >= shoutboxMaxConnections {
		return false
	}
	if clients == nil {
		clients = make(map[*shoutboxClient]struct{})
		h.users[client.userID] = clients
	}
	clients[client] = struct{}{}
	h.ignores[client.userID] = ignores

	// Full buckets of disconnected users are the same as new ones
	now := time.Now()
	for id, bucket := range h.buckets {
		if len(h.users[id]) == 0 {
			bucket.refill(now)
			if bucket.tokens >= shoutboxBurst {
				delete(h.buckets, id)
			}
		}
	}
	return true
}

// Remove a client, the caller holds the lock
func (h *shoutboxHub) remove(client *shoutboxClient) {
	clients, ok := h.users[client.userID]
	if !ok {
		return
	}
	delete(clients, client)
	if len(clients) == 0 {
		delete(h.users, client.userID)
		delete(h.ignores, client.userID)
	}
}

func (h *shoutboxHub) unregister(client *shoutboxClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.remove(client)
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

// Replace the ignores of a user, if connected
func (h *shoutboxHub) setIgnores(userID uint, ignores map[uint]time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.users[userID]; ok {
		h.ignores[userID] = ignores
	}
}

// Whether a connected user ignores another, the caller holds the lock
func (h *shoutboxHub) ignoring(userID uint, otherID uint, now time.Time) bool {
	until, ok := h.ignores[userID][otherID]
	return ok && now.Before(until)
}

// Set the user whose conversation a client shows, 0 for none
func (h *shoutboxHub) setViewing(client *shoutboxClient, partnerID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	client.viewing = partnerID
}

// Whether one of the user's clients shows the conversation with a partner
func (h *shoutboxHub) viewing(userID uint, partnerID uint) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.users[userID] {
		if client.viewing == partnerID {
			return true
		}
	}
	return false
}

// Queue a payload for a client, slow clients that can't keep up are disconnected. The caller holds the lock.
func (h *shoutboxHub) queue(client *shoutboxClient, payload []byte) {
	if !client.queue(payload) {
		h.remove(client)
	}
}

// Queue a payload for every client, except those of users ignoring the sender
func (h *shoutboxHub) broadcast(senderID uint, payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	for userID, clients := range h.users {
		if h.ignoring(userID, senderID, now) {
			continue
		}
		for client := range clients {
			h.queue(client, payload)
		}
	}
}

// Queue a payload for every client of the given users
func (h *shoutboxHub) deliver(payload []byte, userIDs ...uint) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i, userID := range userIDs {
		// A user messaging themselves would get it twice
		if i > 0 && userID == userIDs[0] {
			continue
		}
		for client := range h.users[userID] {
			h.queue(client, payload)
		}
	}
}

// A websocket connection to the shoutbox
type shoutboxClient struct {
	userID uint
	// User whose conversation is shown, 0 for none. Guarded by the hub's lock.
	viewing uint
	send    chan []byte
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
