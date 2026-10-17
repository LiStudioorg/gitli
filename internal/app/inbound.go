// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"bytes"
	"context"
	"encoding/hex"
	"time"

	"github.com/LiStudioorg/gitli/internal/infra/inboundmail"
	"github.com/LiStudioorg/gitli/internal/service/comment"
)

// ReplyToken returns a signer for reply addresses, or nil when no cipher/pepper
// is available.
func (a *App) replySigner() *inboundmail.Token {
	key := a.replyKey()
	if len(key) < 16 {
		return nil
	}
	return inboundmail.NewToken(key)
}

// replyKey derives an HMAC key from available secrets.
func (a *App) replyKey() []byte {
	if a.Cfg.Security.Crypto.KeyHex != "" {
		if b, err := hex.DecodeString(a.Cfg.Security.Crypto.KeyHex); err == nil {
			return b
		}
	}
	if a.Cfg.Security.Password.Pepper != "" {
		return []byte(a.Cfg.Security.Password.Pepper)
	}
	return nil
}

// HandleInboundMessage processes one raw reply email and stores it as a comment.
func (a *App) HandleInboundMessage(ctx context.Context, raw []byte) error {
	tk := a.replySigner()
	if tk == nil {
		return nil
	}
	reply, err := inboundmail.ParseMessage(bytes.NewReader(raw), tk)
	if err != nil {
		return err
	}
	u, err := a.Deps.Users.ByEmail(ctx, reply.From)
	if err != nil {
		return err
	}
	switch reply.Kind {
	case "issue":
		is, err := a.Deps.Issues.ByID(ctx, reply.TargetID)
		if err != nil {
			return err
		}
		_, err = a.Deps.Comments.Create(ctx, comment.CreateParams{
			RepoID: is.RepoID, IssueID: &is.ID, AuthorID: u.ID, Body: reply.Body,
		})
		return err
	case "pull":
		p, err := a.Deps.Pulls.ByID(ctx, reply.TargetID)
		if err != nil {
			return err
		}
		_, err = a.Deps.Comments.Create(ctx, comment.CreateParams{
			RepoID: p.RepoID, PullID: &p.ID, AuthorID: u.ID, Body: reply.Body,
		})
		return err
	}
	return nil
}

// runInboundMailer periodically fetches and processes reply emails.
func (a *App) runInboundMailer(ctx context.Context) {
	fetcher := inboundmail.NewFetcher(a.Cfg.Mail.Inbound)
	if fetcher == nil {
		return
	}
	interval := time.Duration(a.Cfg.Mail.Inbound.IntervalSec) * time.Second
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := fetcher.Fetch(ctx, func(raw []byte) error {
				return a.HandleInboundMessage(ctx, raw)
			})
			if err != nil {
				a.Log.Warn("inbound mail fetch failed", "error", err)
			} else if n > 0 {
				a.Log.Info("inbound replies processed", "count", n)
			}
		}
	}
}
