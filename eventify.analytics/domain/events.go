package domain

import "github.com/google/uuid"

// events.go defines the structure of events in the system.
type EventCreated struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	MessageId   uuid.UUID `json:"message_id"`
	CountryCode string    `json:"country_code"`
	OccurredAt  string    `json:"occurred_at"`
	CreatedBy   string    `json:"created_by"`
}
