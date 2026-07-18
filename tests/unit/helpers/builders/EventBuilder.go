package builders

// create events for tests

import (
	"eventify/internal/domain"
	"time"

	"github.com/google/uuid"
)

type EventBuilder struct {
	event *domain.Event
}

func NewEventBuilder() *EventBuilder {
	return &EventBuilder{
		event: &domain.Event{
			Id:        uuid.New(),
			Name:      "Test Event",
			Location:  "Test Location",
			Date:      time.Now().Add(24 * time.Hour),
			CreatedBy: uuid.New(),
		},
	}
}

func (eb *EventBuilder) WithName(name string) *EventBuilder {
	eb.event.Name = name
	return eb
}

func (eb *EventBuilder) WithLocation(location string) *EventBuilder {
	eb.event.Location = location
	return eb
}

func (eb *EventBuilder) WithDate(date time.Time) *EventBuilder {
	eb.event.Date = date
	return eb
}

func (eb *EventBuilder) WithCreatedBy(createdBy uuid.UUID) *EventBuilder {
	eb.event.CreatedBy = createdBy
	return eb
}

func (eb *EventBuilder) Create() *domain.Event {
	return eb.event
}
