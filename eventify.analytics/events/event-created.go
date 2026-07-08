package events

import "github.com/google/uuid"

type EventCreated struct {
	MessageId   uuid.UUID `json:"message_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	CountryCode string    `json:"country_code"`
	OccurredAt  string    `json:"occurred_at"`
	CreatedBy   string    `json:"created_by"`
}
