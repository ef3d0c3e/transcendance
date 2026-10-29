package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strings"
	"time"
)

// Signed in admin, kept in a cookie signed by the server
type session struct {
	User string `json:"u"`
	// Fingerprint of the approving key, the session ends when it is revoked
	Key     string `json:"k"`
	Expires int64  `json:"e"`
}

func signSession(secret []byte, s session) (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// The session in a cookie value, if its signature is valid and it has not expired
func parseSession(secret []byte, value string, now time.Time) (session, bool) {
	payload, signature, found := strings.Cut(value, ".")
	if !found {
		return session{}, false
	}
	got, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return session{}, false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return session{}, false
	}
	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return session{}, false
	}
	var s session
	if err := json.Unmarshal(data, &s); err != nil {
		return session{}, false
	}
	if s.User == "" || s.Key == "" || now.Unix() >= s.Expires {
		return session{}, false
	}
	return s, true
}

// Secret signing the sessions, created on first start
func loadSecret(path string) ([]byte, error) {
	secret, err := os.ReadFile(path)
	if err == nil {
		if len(secret) < 32 {
			return nil, errors.New(path + " is too short, delete it to create a new one")
		}
		return secret, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	secret = make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	return secret, os.WriteFile(path, secret, 0o600)
}
