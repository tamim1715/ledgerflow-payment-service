package webhook

import (
	"time"
)

type EventStatus string

const (
	EventStatusProcessed EventStatus = "PROCESSED"
	EventStatusIgnored   EventStatus = "IGNORED"
	EventStatusFailed    EventStatus = "FAILED"
)

type WebhookEvent struct {
	ID          string      `db:"id" json:"id"`
	Provider    string      `db:"provider" json:"provider"`
	EventID     string      `db:"event_id" json:"event_id"`
	EventType   string      `db:"event_type" json:"event_type"`
	Payload     string      `db:"payload" json:"payload"`
	Status      EventStatus `db:"status" json:"status"`
	ProcessedAt time.Time   `db:"processed_at" json:"processed_at"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
}
