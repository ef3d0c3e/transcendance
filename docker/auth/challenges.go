package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"
)

// Letters of a challenge code, without the ones easily confused
const codeAlphabet = "abcdefghjkmnpqrstuvwxyz"

// Most challenges waiting at once, the oldest expire first
const maxPendingChallenges = 1000

var errTooManyChallenges = errors.New("too many pending sign-in requests")

type challengeStatus string

const (
	statusPending  challengeStatus = "pending"
	statusApproved challengeStatus = "approved"
	statusDenied   challengeStatus = "denied"
	statusExpired  challengeStatus = "expired"
)

// A browser waiting for an admin to approve it over SSH
type challenge struct {
	// Typed as the SSH user name
	Code string
	// Secret kept in the requesting browser's cookie
	Browser string
	// Who asked, shown before approving
	IP      string
	Agent   string
	Expires time.Time
	Status  challengeStatus
	// Set when approved
	User string
	Key  string
}

type challengeStore struct {
	ttl time.Duration
	now func() time.Time

	mu        sync.Mutex
	byCode    map[string]*challenge
	byBrowser map[string]*challenge
}

func newChallengeStore(ttl time.Duration, now func() time.Time) *challengeStore {
	return &challengeStore{
		ttl:       ttl,
		now:       now,
		byCode:    map[string]*challenge{},
		byBrowser: map[string]*challenge{},
	}
}

// Code like abcd-efgh-jkmn, about 54 random bits
func newCode() (string, error) {
	var code strings.Builder
	buf := make([]byte, 1)
	for code.Len() < 14 {
		if code.Len() == 4 || code.Len() == 9 {
			code.WriteByte('-')
			continue
		}
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		// Rejection keeps every letter equally likely
		limit := byte(256 - 256%len(codeAlphabet))
		if buf[0] >= limit {
			continue
		}
		code.WriteByte(codeAlphabet[int(buf[0])%len(codeAlphabet)])
	}
	return code.String(), nil
}

func newBrowserToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

func (cs *challengeStore) delete(c *challenge) {
	delete(cs.byCode, c.Code)
	delete(cs.byBrowser, c.Browser)
}

func (cs *challengeStore) purge(now time.Time) {
	for _, c := range cs.byCode {
		if !now.Before(c.Expires) {
			cs.delete(c)
		}
	}
}

func (cs *challengeStore) create(ip string, agent string) (challenge, error) {
	code, err := newCode()
	if err != nil {
		return challenge{}, err
	}
	browser, err := newBrowserToken()
	if err != nil {
		return challenge{}, err
	}
	now := cs.now()

	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.purge(now)
	if len(cs.byCode) >= maxPendingChallenges {
		return challenge{}, errTooManyChallenges
	}
	if _, taken := cs.byCode[code]; taken {
		return challenge{}, errors.New("challenge code collision")
	}
	c := &challenge{
		Code:    code,
		Browser: browser,
		IP:      ip,
		Agent:   agent,
		Expires: now.Add(cs.ttl),
		Status:  statusPending,
	}
	cs.byCode[code] = c
	cs.byBrowser[browser] = c
	return *c, nil
}

// The challenge waiting for approval under a code
func (cs *challengeStore) pending(code string) (challenge, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	c, ok := cs.byCode[code]
	if !ok || c.Status != statusPending || !cs.now().Before(c.Expires) {
		return challenge{}, false
	}
	return *c, true
}

// Approve or deny a pending challenge, false when it is no longer pending
func (cs *challengeStore) decide(code string, approved bool, user string, key string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	c, ok := cs.byCode[code]
	if !ok || c.Status != statusPending || !cs.now().Before(c.Expires) {
		return false
	}
	if approved {
		c.Status = statusApproved
		c.User = user
		c.Key = key
	} else {
		c.Status = statusDenied
	}
	return true
}

// Status seen by the browser, an approved or denied challenge is used up
func (cs *challengeStore) claim(browser string) challenge {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	c, ok := cs.byBrowser[browser]
	if !ok {
		return challenge{Status: statusExpired}
	}
	if c.Status == statusPending && !cs.now().Before(c.Expires) {
		cs.delete(c)
		return challenge{Status: statusExpired}
	}
	if c.Status != statusPending {
		cs.delete(c)
	}
	return *c
}
