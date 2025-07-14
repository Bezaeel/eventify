package service

import (
	"context"
	"eventify/internal/domain"
	"eventify/pkg/logger"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EventService struct {
	db  *gorm.DB
	log *logger.Logger
}

type IEventService interface {
	CreateEvent(event *domain.Event, ctx context.Context) error
	GetEventById(id uuid.UUID, ctx context.Context) *domain.Event
	UpdateEvent(event *domain.Event, ctx context.Context) error
	DeleteEvent(id uuid.UUID, ctx context.Context) error
	GetAllEvents(ctx context.Context) []domain.Event
}

func NewEventService(db *gorm.DB, log *logger.Logger) *EventService {
	return &EventService{db, log}
}

// CreateEvent creates a new event in the database.
func (e *EventService) CreateEvent(event *domain.Event, ctx context.Context) error {
	e.log.WithFields(logger.Fields{
		"timeInUTC": time.Now().UTC(),
		"time":      time.Now(),
	}).Info("Received CreateEvent")

	return e.db.WithContext(ctx).Create(event).Error
}

// GetEventById retrieves an event by its ID.
func (e *EventService) GetEventById(id uuid.UUID, ctx context.Context) *domain.Event {
	event := &domain.Event{}
	if err := e.db.WithContext(ctx).First(event, id).Error; err != nil {
		return nil
	}
	return event
}

// UpdateEvent updates an existing event in the database.
func (e *EventService) UpdateEvent(event *domain.Event, ctx context.Context) error {
	return e.db.WithContext(ctx).Save(event).Error
}

// DeleteEvent deletes an event by its ID.
func (e *EventService) DeleteEvent(id uuid.UUID, ctx context.Context) error {
	return e.db.WithContext(ctx).Delete(&domain.Event{}, id).Error
}

// GetAllEvents retrieves all events from the database.
func (e *EventService) GetAllEvents(ctx context.Context) []domain.Event {
	var events []domain.Event
	if err := e.db.WithContext(ctx).Find(&events).Error; err != nil {
		return nil
	}
	return events
}
