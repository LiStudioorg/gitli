// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"time"

	"github.com/LiStudioorg/gitli/internal/infra/mail"
	"github.com/LiStudioorg/gitli/internal/service/digest"
)

// sendDigests runs one digest pass and returns the number of emails sent.
func (a *App) sendDigests(ctx context.Context, now time.Time) (int, error) {
	if a.Deps.Digest == nil {
		return 0, nil
	}
	sender := mail.New(mail.Config{
		Enabled: a.Cfg.Mail.Enabled, Host: a.Cfg.Mail.Host, Port: a.Cfg.Mail.Port,
		Username: a.Cfg.Mail.Username, Password: a.Cfg.Mail.Password, From: a.Cfg.Mail.From, TLS: a.Cfg.Mail.TLS,
	})
	if !sender.Enabled() {
		return 0, nil
	}
	candidates, err := a.Deps.Digest.DueCandidates(ctx, now)
	if err != nil {
		return 0, err
	}
	baseURL := a.Deps.External.HTTPBase
	sent := 0
	for _, c := range candidates {
		items, err := a.Deps.Digest.Unread(ctx, c.UserID, 50)
		if err != nil {
			continue
		}
		u, _ := a.Deps.Users.ByID(ctx, c.UserID)
		name := ""
		if u != nil {
			name = u.Username
		}
		prefsURL := ""
		if baseURL != "" {
			prefsURL = baseURL + "/settings?tab=general"
		}
		body := digest.RenderHTML(name, baseURL, items, prefsURL)
		if err := sender.Send(ctx, c.Email, "gitli 通知摘要", body); err != nil {
			a.Log.Warn("digest send failed", "user", c.UserID, "error", err)
			continue
		}
		_ = a.Deps.Digest.MarkSent(ctx, c.UserID, now)
		sent++
	}
	return sent, nil
}

// runDigestScheduler runs the digest pass every hour.
func (a *App) runDigestScheduler(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := a.sendDigests(ctx, time.Now())
			if err != nil {
				a.Log.Warn("digest pass failed", "error", err)
			} else if n > 0 {
				a.Log.Info("notification digests sent", "count", n)
			}
		}
	}
}
