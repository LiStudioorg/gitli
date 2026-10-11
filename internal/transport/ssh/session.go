// SPDX-License-Identifier: AGPL-3.0
package ssh

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

// handleSession processes channel requests, honoring the exec request and the
// configured idle timeout.
func (s *Server) handleSession(ctx context.Context, username string, ch ssh.Channel, reqs <-chan *ssh.Request) {
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
			switch req.Type {
			case "exec":
				cmd, ok := decodeExecPayload(req.Payload)
				if !ok {
					_ = req.Reply(false, nil)
					continue
				}
				_ = req.Reply(true, nil)
				s.runCommand(ctx, username, cmd, ch)
				return
			default:
				_ = req.Reply(false, nil)
			}
		case <-timer.C:
			return
		case <-ctx.Done():
			return
		}
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

func (s *Server) runCommand(ctx context.Context, username, raw string, ch ssh.Channel) {
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
