package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Characters kept from a key comment to name its owner
var unsafeUserChars = regexp.MustCompile(`[^A-Za-z0-9._@+-]`)

// An admin allowed to sign in
type authorizedKey struct {
	// SHA256 fingerprint, as printed by ssh-keygen -l
	Fingerprint string
	// Name given to the tools, from the key comment
	User string
	Line string
}

// The authorized_keys file, read again whenever it changes
type keyring struct {
	path string

	mu      sync.Mutex
	modTime time.Time
	size    int64
	keys    map[string]authorizedKey
}

func newKeyring(path string) *keyring {
	return &keyring{path: path, keys: map[string]authorizedKey{}}
}

// Owner name from a key comment, the fingerprint when there is none
func keyUser(comment string, fingerprint string) string {
	user := unsafeUserChars.ReplaceAllString(strings.TrimSpace(comment), "")
	if len(user) > 64 {
		user = user[:64]
	}
	if user == "" {
		return "key-" + strings.TrimPrefix(fingerprint, "SHA256:")[:12]
	}
	return user
}

func parseAuthorizedKeys(data []byte) (map[string]authorizedKey, error) {
	keys := map[string]authorizedKey{}
	for n, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		key, comment, _, _, err := ssh.ParseAuthorizedKey(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", n+1, err)
		}
		fingerprint := ssh.FingerprintSHA256(key)
		keys[fingerprint] = authorizedKey{
			Fingerprint: fingerprint,
			User:        keyUser(comment, fingerprint),
			Line:        string(line),
		}
	}
	return keys, nil
}

// Read the file again when it changed, a missing file allows nobody
func (k *keyring) refresh() error {
	info, err := os.Stat(k.path)
	if errors.Is(err, fs.ErrNotExist) {
		k.keys = map[string]authorizedKey{}
		k.modTime = time.Time{}
		k.size = 0
		return nil
	}
	if err != nil {
		return err
	}
	if info.ModTime().Equal(k.modTime) && info.Size() == k.size {
		return nil
	}
	data, err := os.ReadFile(k.path)
	if err != nil {
		return err
	}
	keys, err := parseAuthorizedKeys(data)
	if err != nil {
		return fmt.Errorf("%s: %w", k.path, err)
	}
	k.keys = keys
	k.modTime = info.ModTime()
	k.size = info.Size()
	return nil
}

// The admin owning a key fingerprint, if it is still authorized
func (k *keyring) lookup(fingerprint string) (authorizedKey, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := k.refresh(); err != nil {
		// A broken file must not keep old keys working
		logf("WARN cannot read authorized keys: %s", err)
		k.keys = map[string]authorizedKey{}
		k.modTime = time.Time{}
		return authorizedKey{}, false
	}
	key, ok := k.keys[fingerprint]
	return key, ok
}

func (k *keyring) list() ([]authorizedKey, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := k.refresh(); err != nil {
		return nil, err
	}
	keys := make([]authorizedKey, 0, len(k.keys))
	for _, key := range k.keys {
		keys = append(keys, key)
	}
	return keys, nil
}

// Authorize a public key, its comment names the admin
func (k *keyring) add(line string) (authorizedKey, error) {
	key, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		return authorizedKey{}, fmt.Errorf("not a public key: %w", err)
	}
	if strings.TrimSpace(comment) == "" {
		return authorizedKey{}, errors.New("the key needs a comment naming its owner, e.g. alice@laptop")
	}
	fingerprint := ssh.FingerprintSHA256(key)
	entry := authorizedKey{
		Fingerprint: fingerprint,
		User:        keyUser(comment, fingerprint),
		Line:        strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))) + " " + strings.TrimSpace(comment),
	}

	k.mu.Lock()
	defer k.mu.Unlock()
	if err := k.refresh(); err != nil {
		return authorizedKey{}, err
	}
	if _, ok := k.keys[fingerprint]; ok {
		return authorizedKey{}, errors.New("this key is already authorized")
	}
	if err := os.MkdirAll(filepath.Dir(k.path), 0o700); err != nil {
		return authorizedKey{}, err
	}
	file, err := os.OpenFile(k.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return authorizedKey{}, err
	}
	if _, err := file.WriteString(entry.Line + "\n"); err != nil {
		_ = file.Close()
		return authorizedKey{}, err
	}
	return entry, file.Close()
}

// Revoke the keys matching a fingerprint or owner name, their sessions end
func (k *keyring) remove(match string) (int, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := k.refresh(); err != nil {
		return 0, err
	}
	var kept []string
	removed := 0
	for _, key := range k.keys {
		if key.Fingerprint == match || key.User == match {
			removed++
			continue
		}
		kept = append(kept, key.Line+"\n")
	}
	if removed == 0 {
		return 0, nil
	}
	if err := os.WriteFile(k.path, []byte(strings.Join(kept, "")), 0o600); err != nil {
		return 0, err
	}
	return removed, k.refresh()
}
