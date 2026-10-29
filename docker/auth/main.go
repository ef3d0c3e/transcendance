// Device sign-in for the admin tools: a browser shows a code, an admin approves it with ssh -p PORT CODE@HOST
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
)

func logf(format string, args ...any) {
	log.Printf(format, args...)
}

func env(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func main() {
	dataDir := env("AUTH_DATA_DIR", "/data")
	keys := newKeyring(filepath.Join(dataDir, "authorized_keys"))
	if len(os.Args) > 1 {
		os.Exit(runCommand(keys, os.Args[1:]))
	}
	if err := run(dataDir, keys); err != nil {
		log.Fatal(err)
	}
}

func run(dataDir string, keys *keyring) error {
	sessionTTL, err := time.ParseDuration(env("AUTH_SESSION_TTL", "12h"))
	if err != nil {
		return fmt.Errorf("AUTH_SESSION_TTL: %w", err)
	}
	secret, err := loadSecret(filepath.Join(dataDir, "session.key"))
	if err != nil {
		return err
	}
	hostKey, err := loadHostKey(filepath.Join(dataDir, "ssh_host_ed25519_key"))
	if err != nil {
		return err
	}

	s := &server{
		keys:           keys,
		secret:         secret,
		challenges:     newChallengeStore(5*time.Minute, time.Now),
		challengeLimit: newLimiter(10, time.Minute, time.Now),
		sessionTTL:     sessionTTL,
		sshPort:        env("AUTH_SSH_PUBLIC_PORT", "2233"),
		hostKeyPrint:   ssh.FingerprintSHA256(hostKey.PublicKey()),
		now:            time.Now,
	}
	if admins, err := keys.list(); err != nil {
		return err
	} else if len(admins) == 0 {
		logf("WARN no admin key yet, add one with: docker compose exec auth auth add-key \"$(cat ~/.ssh/id_ed25519.pub)\"")
	}

	sshListener, err := net.Listen("tcp", env("AUTH_SSH_ADDR", ":2233"))
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:              env("AUTH_HTTP_ADDR", ":9000"),
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errs := make(chan error, 2)
	go func() {
		errs <- s.serveSSH(sshListener, s.sshConfig(hostKey), newLimiter(20, time.Minute, time.Now))
	}()
	go func() {
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errs <- err
		}
	}()
	logf("INFO listening for ssh on %s and http on %s, host key %s", sshListener.Addr(), httpServer.Addr, s.hostKeyPrint)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errs:
		return err
	case <-stop:
	}
	_ = sshListener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return httpServer.Shutdown(ctx)
}

const usage = `usage:
  auth                    run the server
  auth add-key "<public key> <owner>"
  auth remove-key <owner or SHA256 fingerprint>
  auth list-keys
  auth health             check the running server, for the container health check
`

func runCommand(keys *keyring, args []string) int {
	switch {
	case args[0] == "add-key" && len(args) == 2:
		key, err := keys.add(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("Authorized %s (%s)\n", key.User, key.Fingerprint)
	case args[0] == "remove-key" && len(args) == 2:
		removed, err := keys.remove(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if removed == 0 {
			fmt.Fprintln(os.Stderr, "no such key")
			return 1
		}
		fmt.Printf("Removed %d key(s), their sessions are revoked\n", removed)
	case args[0] == "health" && len(args) == 1:
		client := http.Client{Timeout: 3 * time.Second}
		response, err := client.Get("http://127.0.0.1" + env("AUTH_HTTP_ADDR", ":9000") + "/auth/healthz")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			fmt.Fprintln(os.Stderr, response.Status)
			return 1
		}
	case args[0] == "list-keys" && len(args) == 1:
		admins, err := keys.list()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		for _, admin := range admins {
			fmt.Printf("%s %s\n", admin.Fingerprint, admin.User)
		}
	default:
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	return 0
}
