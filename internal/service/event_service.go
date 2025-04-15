package service

import (
	"eventify/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventService struct {
	db *gorm.DB
}

type IEventService interface {
	CreateEvent(event *domain.Event) error
	GetEventById(id uuid.UUID) *domain.Event
	UpdateEvent(event *domain.Event) error
	DeleteEvent(id uuid.UUID) error
	GetAllEvents() []domain.Event
}

func NewEventService(db *gorm.DB) *EventService {
	return &EventService{db}
}

// CreateEvent creates a new event in the database.
func (e *EventService) CreateEvent(event *domain.Event) error {
	return e.db.Create(event).Error
}

// GetEventById retrieves an event by its ID.
func (e *EventService) GetEventById(id uuid.UUID) *domain.Event {
	event := &domain.Event{}
	if err := e.db.First(event, id).Error; err != nil {
		return nil
	}
	return event
}

// UpdateEvent updates an existing event in the database.
func (e *EventService) UpdateEvent(event *domain.Event) error {
	return e.db.Save(event).Error
}

// DeleteEvent deletes an event by its ID.
func (e *EventService) DeleteEvent(id uuid.UUID) error {
	return e.db.Delete(&domain.Event{}, id).Error
}

// GetAllEvents retrieves all events from the database.
func (e *EventService) GetAllEvents() []domain.Event {
	var events []domain.Event
	if err := e.db.Find(&events).Error; err != nil {
		return nil
	}
	return events
}
