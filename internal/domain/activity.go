// SPDX-License-Identifier: AGPL-3.0
package domain

import "time"

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
	ID         int64
	ActorID    int64
	Verb       string
	ObjectType string
	ObjectID   int64
	RepoID     *int64
	CreatedAt  time.Time
}

// AuditLog records a security-relevant action.
type AuditLog struct {
	ID         int64
	ActorID    *int64
	Action     string
	TargetType string
	TargetID   *int64
	IP         string
	Detail     string
	CreatedAt  time.Time
}
