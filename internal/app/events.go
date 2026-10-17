// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// Event is a domain event emitted for webhook delivery and notifications.
type Event struct {
	Action  string // e.g. "issues", "pull_request", "push"
	RepoID  int64
	ActorID int64
	Payload map[string]any
}

// DispatchRepoEvent delivers a repository event to all active webhooks whose
// event filter matches. Delivery is best-effort and never blocks the caller.
func (a *App) DispatchRepoEvent(ctx context.Context, ev Event) {
	if a.Deps == nil {
		return
	}
	a.recordEventActivity(ctx, ev)
	if a.Deps.Webhooks == nil {
		return
	}
	hooks, err := a.Deps.Webhooks.ListForRepo(ctx, ev.RepoID)
	if err != nil {
		a.Log.Warn("dispatch: list webhooks", "error", err)
		return
	}
	body := map[string]any{
		"action":   ev.Action,
		"repo_id":  ev.RepoID,
		"actor_id": ev.ActorID,
		"data":     ev.Payload,
	}
	for _, h := range hooks {
		if !h.Active || !webhookWants(h.Events, ev.Action) {
			continue
		}
		hook := h
		payloadJSON, _ := json.Marshal(body)
		if a.Deps.Queue != nil {
			// Deliver asynchronously; encode the hook id + payload.
			if !a.Deps.Queue.Enqueue("webhook-deliver", fmt.Sprintf("%d|%s|%s", hook.ID, ev.Action, payloadJSON)) {
				a.deliverWebhook(ctx, hook, ev.Action, body)
			}
		} else {
			a.deliverWebhook(ctx, hook, ev.Action, body)
		}
	}
}

// recordEventActivity mirrors repository events into the activity timeline so
// the dashboard feed reflects issues, pull requests and discussions.
func (a *App) recordEventActivity(ctx context.Context, ev Event) {
	if a.Deps.Activities == nil || ev.ActorID == 0 || ev.RepoID == 0 {
		return
	}
	verb, ok := activityVerb(ev.Action, ev.Payload)
	if !ok {
		return
	}
	repoID := ev.RepoID
	fullName := ""
	if a.Deps.Repos != nil {
		if rp, err := a.Deps.Repos.ByID(ctx, ev.RepoID); err == nil && rp != nil {
			fullName = rp.Name
			if a.Deps.Users != nil && rp.OwnerType == domain.OwnerUser {
				if owner, err := a.Deps.Users.ByID(ctx, rp.OwnerID); err == nil && owner != nil {
					fullName = owner.Username + "/" + rp.Name
				}
			}
		}
	}
	objectID := ev.RepoID
	if v, ok := ev.Payload["number"].(int); ok {
		objectID = int64(v)
	} else if v, ok := ev.Payload["id"].(int64); ok {
		objectID = v
	}
	summary, _ := ev.Payload["title"].(string)
	if summary == "" {
		summary, _ = ev.Payload["tag"].(string)
	}
	_ = a.Deps.Activities.Record(ctx, &domain.Activity{
		ActorID:    ev.ActorID,
		Verb:       verb,
		ObjectType: ev.Action,
		ObjectID:   objectID,
		RepoID:     &repoID,
		RepoName:   fullName,
		Summary:    summary,
	})
}

// activityVerb maps an event action plus payload action to a timeline verb.
func activityVerb(action string, payload map[string]any) (string, bool) {
	sub, _ := payload["action"].(string)
	switch action {
	case "issues":
		switch sub {
		case "opened", "created":
			return "opened_issue", true
		case "closed":
			return "closed_issue", true
		default:
			return "", false
		}
	case "pull_request":
		switch sub {
		case "opened", "created":
			return "opened_pull", true
		case "closed", "merged":
			if merged, _ := payload["merged"].(bool); merged {
				return "merged_pull", true
			}
			return "closed_pull", true
		default:
			return "", false
		}
	case "release":
		if sub == "published" || sub == "created" {
			return "published_release", true
		}
		return "", false
	case "discussion":
		return "started_discussion", sub == "created"
	default:
		return "", false
	}
}

func (a *App) deliverWebhook(ctx context.Context, h *domain.Webhook, event string, payload any) {
	if _, err := a.Deps.Webhooks.Deliver(ctx, h, event, payload); err != nil {
		a.Log.Warn("webhook delivery failed", "webhook", h.ID, "error", err)
	}
}

// webhookWants reports whether an event name matches the webhook's filter.
func webhookWants(events []string, action string) bool {
	if len(events) == 0 {
		return true
	}
	for _, e := range events {
		if e == "*" || e == action {
			return true
		}
	}
	return false
}

// Notify creates an in-app notification for a user.
func (a *App) Notify(ctx context.Context, userID int64, subjectType string, subjectID int64, title, url string) {
	if a.Deps == nil || a.Deps.Notifications == nil || userID == 0 {
		return
	}
	_ = a.Deps.Notifications.Create(ctx, &domain.Notification{
		UserID:      userID,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Title:       title,
		URL:         url,
	})
}

// RunWebhookDeliver executes a queued webhook delivery. Payload format is
// "<hookID>|<event>|<json>".
func (a *App) RunWebhookDeliver(ctx context.Context, payload string) error {
	parts := splitN(payload, "|", 3)
	if len(parts) != 3 {
		return fmt.Errorf("webhook-deliver: malformed payload")
	}
	var hookID int64
	if _, err := fmt.Sscanf(parts[0], "%d", &hookID); err != nil {
		return fmt.Errorf("webhook-deliver: bad hook id: %w", err)
	}
	h, err := a.Deps.Webhooks.ByID(ctx, hookID)
	if err != nil {
		return err
	}
	var body any
	if err := json.Unmarshal([]byte(parts[2]), &body); err != nil {
		return fmt.Errorf("webhook-deliver: bad payload: %w", err)
	}
	_, err = a.Deps.Webhooks.Deliver(ctx, h, parts[1], body)
	return err
}

func splitN(s, sep string, n int) []string {
	out := make([]string, 0, n)
	for n > 1 {
		i := indexOf(s, sep)
		if i < 0 {
			break
		}
		out = append(out, s[:i])
		s = s[i+len(sep):]
		n--
	}
	out = append(out, s)
	return out
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
