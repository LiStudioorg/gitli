// SPDX-License-Identifier: AGPL-3.0
package domain

import (
	"errors"
	"time"
)

// Webhook is an outbound webhook endpoint.
type Webhook struct {
	ID        int64
	RepoID    *int64
	OrgID     *int64
	URL       string
	Secret    string
	Events    []string
	Active    bool
	CreatedAt time.Time
}

// WebhookDelivery records a single delivery attempt chain.
type WebhookDelivery struct {
	ID           int64
	WebhookID    int64
	Event        string
	StatusCode   int
	RequestBody  string
	ResponseBody string
	Attempts     int
	DeliveredAt  *time.Time
	CreatedAt    time.Time
}

// Notification is an in-app notification.
type Notification struct {
	ID          int64
	UserID      int64
	SubjectType string
	SubjectID   int64
	Title       string
	URL         string
	IsRead      bool
	CreatedAt   time.Time
}

// Activity is a timeline entry.
type Activity struct {
	ID         int64     `json:"id"`
	ActorID    int64     `json:"actor_id"`
	Verb       string    `json:"verb"`
	ObjectType string    `json:"object_type"`
	ObjectID   int64     `json:"object_id"`
	RepoID     *int64    `json:"repo_id,omitempty"`
	RepoName   string    `json:"repo_name,omitempty"`
	Summary    string    `json:"summary,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	ActorName  string    `json:"actor_name,omitempty"`
}

// AuditLog records a security-relevant action.
type AuditLog struct {
	ID             int64     `json:"id"`
	ActorID        *int64    `json:"actor_id"`
	ActorType      string    `json:"actor_type"`
	ActorIP        string    `json:"actor_ip"`
	ActorUserAgent string    `json:"actor_user_agent"`
	Event          string    `json:"event"`
	Resource       string    `json:"resource"`
	Action         string    `json:"action"`
	TargetType     string    `json:"target_type"`
	TargetID       *int64    `json:"target_id"`
	TargetName     string    `json:"target_name"`
	OldValue       string    `json:"old_value"`
	NewValue       string    `json:"new_value"`
	Result         string    `json:"result"`
	ErrorMessage   string    `json:"error_message"`
	RequestID      string    `json:"request_id"`
	Metadata       string    `json:"metadata"`
	IP             string    `json:"ip"`
	Detail         string    `json:"detail"`
	CreatedAt      time.Time `json:"created_at"`
}

// ErrAuditNotFound indicates a missing audit log.
var ErrAuditNotFound = errors.New("audit: not found")
