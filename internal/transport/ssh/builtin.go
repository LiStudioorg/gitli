// SPDX-License-Identifier: AGPL-3.0
// Package ssh implements an optional built-in SSH server for git transport.
package ssh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/infra/netutil"
)

// Options configures the built-in SSH server.
type Options struct {
	Config   *config.Config
	Log      *slog.Logger
	Auth     PublicKeyAuthorizer
	Dispatch CommandDispatcher
}

// PublicKeyAuthorizer resolves an SSH public key fingerprint to an account.
type PublicKeyAuthorizer interface {
	// AuthorizeSSHKey maps a fingerprint to a login name. For user keys the
	// name is the username; for deploy keys it is the repository owner and
	// readOnly is true when the deploy key must not push.
	AuthorizeSSHKey(ctx context.Context, fingerprint string) (username string, readOnly bool, ok bool)
	TouchSSHKey(ctx context.Context, fingerprint string)
}

// CommandDispatcher runs an authorized git command and streams its output.
type CommandDispatcher interface {
	Dispatch(ctx context.Context, username, command string, in InputReader, out, errOut OutputWriter) error
}

// InputReader mirrors io.Reader for command stdin.
type InputReader interface {
	Read(p []byte) (int, error)
}

// OutputWriter mirrors io.Writer for command stdout/stderr.
type OutputWriter interface {
	Write(p []byte) (int, error)
}

// Server is the built-in SSH server.
type Server struct {
	cfg      *config.Config
	log      *slog.Logger
	auth     PublicKeyAuthorizer
	dispatch CommandDispatcher
	signer   ssh.Signer
	sem      chan struct{}
}

// NewServer builds a built-in SSH server, loading or generating its host key.
func NewServer(opts Options) (*Server, error) {
	if opts.Config == nil {
		return nil, errors.New("ssh: nil config")
	}
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	signer, err := loadOrCreateHostKey(opts.Config.SSH.HostKeyPath)
	if err != nil {
		return nil, err
	}
	max := opts.Config.SSH.MaxConnections
	if max <= 0 {
		max = 32
	}
	return &Server{
		cfg:      opts.Config,
		log:      opts.Log,
		auth:     opts.Auth,
		dispatch: opts.Dispatch,
		signer:   signer,
		sem:      make(chan struct{}, max),
	}, nil
}

// ListenAndServe accepts connections until ctx is cancelled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if !s.cfg.Server.SSHEnabled {
		s.log.Info("SSH server disabled by config")
		<-ctx.Done()
		return nil
	}
	if s.cfg.SSH.Mode == "system" {
		s.log.Info("SSH server mode=system; relying on system sshd + git-shell")
		<-ctx.Done()
		return nil
	}
	sshCfg := s.serverConfig()
	sshCfg.AddHostKey(s.signer)

	addrs := s.cfg.Server.SSHListen
	if len(addrs) == 0 {
		addrs = []string{"0.0.0.0:2222"}
	}
	var wg sync.WaitGroup
	for _, addr := range addrs {
		ln, err := net.Listen(netutil.NetworkFor(addr), addr)
		if err != nil {
			return fmt.Errorf("ssh: listen %s: %w", addr, err)
		}
		family := netutil.Family(ln.Addr())
		s.log.Info("SSH listening", "addr", ln.Addr().String(), "family", family)
		wg.Add(1)
		go func(l net.Listener) {
			defer wg.Done()
			s.serveListener(ctx, l, sshCfg)
		}(ln)
		go func(l net.Listener) {
			<-ctx.Done()
			_ = l.Close()
		}(ln)
	}
	wg.Wait()
	return nil
}

// serverConfig builds the hardened ssh.ServerConfig used for every connection.
//
// Only public-key authentication is permitted (NoClientAuth=false) and the
// number of authentication attempts is bounded. Port/agent/X11 forwarding are
// intentionally never enabled: the struct exposes no option to turn them on and
// the connection handler rejects any non-session channel.
func (s *Server) serverConfig() *ssh.ServerConfig {
	return &ssh.ServerConfig{
		PublicKeyCallback: s.authorize,
		ServerVersion:     "SSH-2.0-gitli",
		NoClientAuth:      false,
		MaxAuthTries:      6,
	}
}

// serveListener accepts connections from a single listener.
func (s *Server) serveListener(ctx context.Context, ln net.Listener, sshCfg *ssh.ServerConfig) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		select {
		case s.sem <- struct{}{}:
			go func() {
				defer func() { <-s.sem }()
				s.handleConn(ctx, conn, sshCfg)
			}()
		default:
			s.log.Warn("ssh: connection limit reached, rejecting")
			_ = conn.Close()
		}
	}
}

func (s *Server) authorize(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	fp := ssh.FingerprintSHA256(key)
	username, readOnly, ok := s.auth.AuthorizeSSHKey(context.Background(), fp)
	if !ok {
		return nil, fmt.Errorf("ssh: unknown public key %s", fp)
	}
	if conn.User() != username && conn.User() != "git" {
		return nil, fmt.Errorf("ssh: user mismatch")
	}
	s.auth.TouchSSHKey(context.Background(), fp)
	ro := "0"
	if readOnly {
		ro = "1"
	}
	return &ssh.Permissions{Extensions: map[string]string{"user": username, "deploy_readonly": ro}}, nil
}

func (s *Server) handleConn(ctx context.Context, nConn net.Conn, sshCfg *ssh.ServerConfig) {
	defer func() { _ = nConn.Close() }()
	handshakeTimeout := time.Duration(s.cfg.SSH.HandshakeTimeoutSeconds) * time.Second
	if handshakeTimeout <= 0 {
		handshakeTimeout = 10 * time.Second
	}
	_ = nConn.SetDeadline(time.Now().Add(handshakeTimeout))
	sconn, chans, reqs, err := ssh.NewServerConn(nConn, sshCfg)
	if err != nil {
		s.log.Debug("ssh: handshake failed", "error", err)
		return
	}
	defer func() { _ = sconn.Close() }()
	_ = nConn.SetDeadline(time.Time{})
	go ssh.DiscardRequests(reqs)

	username := sconn.Permissions.Extensions["user"]
	readOnly := sconn.Permissions.Extensions["deploy_readonly"] == "1"
	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			_ = newChan.Reject(ssh.UnknownChannelType, "only session channels supported")
			continue
		}
		ch, chReqs, err := newChan.Accept()
		if err != nil {
			continue
		}
		go s.handleSession(ctx, username, readOnly, ch, chReqs)
	}
}

func loadOrCreateHostKey(path string) (ssh.Signer, error) {
	if path == "" {
		path = "data/ssh_host_ed25519_key"
	}
	data, err := os.ReadFile(path)
	if err == nil {
		return ssh.ParsePrivateKey(data)
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("ssh: read host key: %w", err)
	}
	signer, pemBytes, err := generateHostKey()
	if err != nil {
		return nil, err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("ssh: mkdir host key dir: %w", err)
		}
	}
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("ssh: write host key: %w", err)
	}
	return signer, nil
}

// IsAllowedVerb reports whether verb is a git transport command that the
// built-in SSH server is willing to dispatch.
func IsAllowedVerb(verb string) bool {
	switch verb {
	case "git-upload-pack", "git-receive-pack", "git-upload-archive":
		return true
	default:
		return false
	}
}

// ParseCommand extracts the git command from an SSH exec request, rejecting
// anything that is not a well-formed git-{upload,receive}-pack invocation.
func ParseCommand(cmd string) (string, bool) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", false
	}
	// Reject embedded NUL bytes anywhere in the request.
	if strings.ContainsRune(cmd, '\x00') {
		return "", false
	}
	for _, verb := range []string{"git-upload-pack", "git-receive-pack", "git-upload-archive"} {
		if !IsAllowedVerb(verb) {
			continue
		}
		prefix := verb + " "
		if !strings.HasPrefix(cmd, prefix) {
			continue
		}
		arg := strings.Trim(strings.TrimPrefix(cmd, prefix), "'\"")
		// Shell metacharacters must never survive into the argument.
		if strings.ContainsAny(arg, ";&|`$<>\n\r") {
			return "", false
		}
		arg = strings.TrimPrefix(arg, "/")
		// Reject NUL bytes and path traversal segments.
		if strings.ContainsAny(arg, "\x00") {
			return "", false
		}
		for _, seg := range strings.Split(arg, "/") {
			if seg == ".." {
				return "", false
			}
		}
		return verb + " " + arg, true
	}
	return "", false
}
