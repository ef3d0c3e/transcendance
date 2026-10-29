package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	// Time for a client to authenticate
	handshakeTimeout = 10 * time.Second
	// Time for the admin to answer the prompt
	promptTimeout = 2 * time.Minute
)

var errDenied = errors.New("denied")

// Host key of the SSH server, created on first start
func loadHostKey(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		_, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		block, err := ssh.MarshalPrivateKey(key, "transcendance-auth")
		if err != nil {
			return nil, err
		}
		data = pem.EncodeToMemory(block)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(data)
}

func (s *server) sshConfig(hostKey ssh.Signer) *ssh.ServerConfig {
	config := &ssh.ServerConfig{
		ServerVersion: "SSH-2.0-transcendance-auth",
		MaxAuthTries:  3,
		// Called for each offered key, the permissions only apply once its signature is verified
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			admin, ok := s.keys.lookup(ssh.FingerprintSHA256(key))
			if !ok {
				return nil, errDenied
			}
			if _, ok := s.challenges.pending(meta.User()); !ok {
				return nil, errDenied
			}
			return &ssh.Permissions{Extensions: map[string]string{
				"user": admin.User,
				"key":  admin.Fingerprint,
			}}, nil
		},
	}
	config.AddHostKey(hostKey)
	return config
}

func (s *server) serveSSH(listener net.Listener, config *ssh.ServerConfig, limit *limiter) error {
	for {
		conn, err := listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		if err != nil {
			return err
		}
		host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		if !limit.allow(host) {
			_ = conn.Close()
			continue
		}
		go s.handleSSH(conn, config)
	}
}

func (s *server) handleSSH(conn net.Conn, config *ssh.ServerConfig) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
	sconn, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer func() { _ = sconn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(promptTimeout))
	go ssh.DiscardRequests(requests)

	// One prompt per connection, then wait for the client to hang up so it exits cleanly
	prompted := false
	for newChannel := range channels {
		if prompted || newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.Prohibited, "one session per connection")
			continue
		}
		prompted = true
		channel, channelRequests, err := newChannel.Accept()
		if err != nil {
			return
		}
		s.handleSession(sconn, channel, channelRequests)
	}
}

// Wait for a shell or command request, then ask the admin to approve
func (s *server) handleSession(sconn *ssh.ServerConn, channel ssh.Channel, requests <-chan *ssh.Request) {
	defer func() { _ = channel.Close() }()
	pty := false
	for request := range requests {
		switch request.Type {
		case "pty-req":
			pty = true
			_ = request.Reply(true, nil)
		case "shell", "exec":
			_ = request.Reply(true, nil)
			go ssh.DiscardRequests(requests)
			status := s.prompt(channel, pty, sconn.User(), sconn.Permissions.Extensions)
			_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
			return
		default:
			_ = request.Reply(false, nil)
		}
	}
}

func (s *server) prompt(channel ssh.Channel, pty bool, code string, admin map[string]string) uint32 {
	c, ok := s.challenges.pending(code)
	if !ok {
		_, _ = fmt.Fprint(channel, "This sign-in request expired, ask for a new code.\r\n")
		return 1
	}
	_, _ = fmt.Fprintf(channel,
		"Sign-in request for the transcendance admin tools\r\n"+
			"  From:    %s\r\n"+
			"  Browser: %s\r\n\r\n"+
			"Only approve a request you made yourself.\r\n"+
			"Sign in as %s? [y/N] ",
		printable(c.IP, 64), printable(c.Agent, 160), admin["user"])

	answer, err := readAnswer(channel, pty)
	approved := err == nil && (answer == "y" || answer == "yes")
	if !s.challenges.decide(code, approved, admin["user"], admin["key"]) {
		_, _ = fmt.Fprint(channel, "\r\nThis sign-in request expired, ask for a new code.\r\n")
		return 1
	}
	if !approved {
		logf("INFO %s denied a sign-in from %s", admin["user"], c.IP)
		_, _ = fmt.Fprint(channel, "\r\nDenied.\r\n")
		return 1
	}
	logf("INFO %s approved a sign-in from %s", admin["user"], c.IP)
	_, _ = fmt.Fprint(channel, "\r\nApproved, you can go back to your browser.\r\n")
	return 0
}

// Read a short answer, echoing it when the client sent a terminal
func readAnswer(channel ssh.Channel, pty bool) (string, error) {
	var answer []byte
	buf := make([]byte, 1)
	for len(answer) < 16 {
		if _, err := channel.Read(buf); err != nil {
			if errors.Is(err, io.EOF) && len(answer) > 0 {
				break
			}
			return "", err
		}
		switch b := buf[0]; {
		case b == '\r' || b == '\n':
			return strings.ToLower(strings.TrimSpace(string(answer))), nil
		// Ctrl-C and Ctrl-D refuse
		case b == 0x03 || b == 0x04:
			return "", errDenied
		case b == 0x7f || b == 0x08:
			if len(answer) > 0 {
				answer = answer[:len(answer)-1]
				if pty {
					_, _ = channel.Write([]byte("\b \b"))
				}
			}
		case b >= 0x20 && b < 0x7f:
			answer = append(answer, b)
			if pty {
				_, _ = channel.Write(buf)
			}
		}
	}
	return strings.ToLower(strings.TrimSpace(string(answer))), nil
}
