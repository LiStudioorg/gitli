// SPDX-License-Identifier: AGPL-3.0
package ssh

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// handleSession processes channel requests, honoring the exec request and the
// configured idle timeout.
func (s *Server) handleSession(ctx context.Context, username string, readOnly bool, ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer func() { _ = ch.Close() }()
	idle := time.Duration(s.cfg.SSH.IdleTimeoutSeconds) * time.Second
	if idle <= 0 {
		idle = 300 * time.Second
	}
	timer := time.NewTimer(idle)
	defer timer.Stop()

	for {
		select {
		case req, ok := <-reqs:
			if !ok {
				return
			}
			// Only "exec" is ever honored. Every other request type — including
			// interactive shells, PTYs, subsystems (sftp), env, agent/X11
			// forwarding and port forwarding — is rejected outright. We never
			// allocate a shell, never enable forwarding and never run anything
			// for these requests.
			switch req.Type {
			case "exec":
				cmd, ok := decodeExecPayload(req.Payload)
				if !ok {
					_ = req.Reply(false, nil)
					continue
				}
				_ = req.Reply(true, nil)
				s.runCommand(ctx, username, readOnly, cmd, ch)
				return
			case "shell", "pty-req", "window-change":
				// A rejected shell/pty still leaves the session open: keep the
				// channel alive and continue waiting rather than acting on it.
				_ = req.Reply(false, nil)
				continue
			default:
				// subsystem (sftp), env, signal, x11-req,
				// auth-agent-req@openssh.com, exec-less forwards, etc.
				s.logRejectedRequest(req.Type)
				_ = req.Reply(false, nil)
			}
		case <-timer.C:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) logRejectedRequest(reqType string) {
	if s.log != nil {
		s.log.Debug("ssh: rejected session request", "type", reqType)
	}
}

func decodeExecPayload(payload []byte) (string, bool) {
	if len(payload) < 4 {
		return "", false
	}
	n := int(payload[0])<<24 | int(payload[1])<<16 | int(payload[2])<<8 | int(payload[3])
	if 4+n > len(payload) {
		return "", false
	}
	return string(payload[4 : 4+n]), true
}

func (s *Server) runCommand(ctx context.Context, username string, readOnly bool, raw string, ch ssh.Channel) {
	cmd, ok := ParseCommand(raw)
	if !ok {
		_, _ = fmt.Fprintln(ch.Stderr(), "gitli: unsupported command")
		sendExit(ch, 128)
		return
	}
	if s.dispatch == nil {
		_, _ = fmt.Fprintln(ch.Stderr(), "gitli: git dispatcher not configured")
		sendExit(ch, 1)
		return
	}
	if readOnly && strings.HasPrefix(cmd, "git-receive-pack") {
		_, _ = fmt.Fprintln(ch.Stderr(), "gitli: deploy key is read-only; push is not permitted")
		sendExit(ch, 1)
		return
	}
	if err := s.dispatch.Dispatch(ctx, username, cmd, ch, ch, ch.Stderr()); err != nil {
		_, _ = fmt.Fprintf(ch.Stderr(), "gitli: %v\n", err)
		sendExit(ch, 1)
		return
	}
	sendExit(ch, 0)
}

func sendExit(ch ssh.Channel, status uint32) {
	payload := []byte{byte(status >> 24), byte(status >> 16), byte(status >> 8), byte(status)}
	_, _ = ch.SendRequest("exit-status", false, payload)
}
