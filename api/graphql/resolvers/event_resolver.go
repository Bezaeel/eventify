package resolvers

import (
	"context"
	"eventify/internal/domain"
	"eventify/internal/service"
	"eventify/pkg/telemetry"
	"time"

	"github.com/google/uuid"
)

type EventResolver struct {
	eventService     *service.EventService
	telemetryAdapter telemetry.ITelemetryAdapter
}

func NewEventResolver(eventService *service.EventService, telemetryAdapter telemetry.ITelemetryAdapter) *EventResolver {
	return &EventResolver{
		eventService:     eventService,
		telemetryAdapter: telemetryAdapter,
	}
}

type CreateEventInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Date        string   `json:"date"`
	Location    string   `json:"location"`
	Organizer   string   `json:"organizer"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Capacity    int      `json:"capacity"`
}

type UpdateEventInput struct {
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	Date        *string   `json:"date"`
	Location    *string   `json:"location"`
	Organizer   *string   `json:"organizer"`
	Category    *string   `json:"category"`
	Tags        *[]string `json:"tags"`
	Capacity    *int      `json:"capacity"`
}

type CreateEventResponse struct {
	EventID string `json:"eventId"`
	Message string `json:"message"`
}

type UpdateEventResponse struct {
	Event   *domain.Event `json:"event"`
	Message string        `json:"message"`
}

type DeleteEventResponse struct {
	Message string `json:"message"`
}

type ListEventsResponse struct {
	Events []*domain.Event `json:"events"`
	Total  int             `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

// Query Resolvers
func (r *EventResolver) Event(ctx context.Context, id string) (*domain.Event, error) {
	// Track telemetry
	r.telemetryAdapter.TrackEvent(ctx, "GetEvent", map[string]string{
		"operation": "get_event",
		"service":   "EventService",
		"event_id":  id,
	})

	// Parse UUID
	eventID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	// Use shared service
	domainEvent := r.eventService.GetEventById(eventID, ctx)
	if domainEvent == nil {
		return nil, nil
	}

	// Convert domain model to GraphQL response
	return r.convertToGraphQLEvent(domainEvent), nil
}

func (r *EventResolver) Events(ctx context.Context, page, limit int) (*ListEventsResponse, error) {
	// Track telemetry
	r.telemetryAdapter.TrackEvent(ctx, "ListEvents", map[string]string{
		"operation": "list_events",
		"service":   "EventService",
	})

	// Use shared service
	domainEvents := r.eventService.GetAllEvents(ctx)

	// Convert domain models to GraphQL responses
	events := make([]*domain.Event, len(domainEvents))
	for i, event := range domainEvents {
		events[i] = r.convertToGraphQLEvent(&event)
	}

	return &ListEventsResponse{
		Events: events,
		Total:  len(events),
		Page:   page,
		Limit:  limit,
	}, nil
}

// Mutation Resolvers
func (r *EventResolver) CreateEvent(ctx context.Context, input CreateEventInput) (*CreateEventResponse, error) {
	// Track telemetry
	r.telemetryAdapter.TrackEvent(ctx, "CreateEvent", map[string]string{
		"operation": "create_event",
		"service":   "EventService",
	})

	// Parse date
	date, err := time.Parse(time.RFC3339, input.Date)
	if err != nil {
		return nil, err
	}

	// Convert GraphQL input to domain model
	event := &domain.Event{
		Id:          uuid.New(),
		Name:        input.Name,
		Description: input.Description,
		Date:        date,
		Location:    input.Location,
		Organizer:   input.Organizer,
		Category:    input.Category,
		Tags:        input.Tags,
		Capacity:    input.Capacity,
		CreatedBy:   uuid.New(), // TODO: Get from context
		CreatedAt:   time.Now(),
	}

	// Use shared service
	err = r.eventService.CreateEvent(event, ctx)
	if err != nil {
		// Track error
		r.telemetryAdapter.TrackError(err, map[string]string{
			"operation": "create_event",
			"service":   "EventService",
		})
		return nil, err
	}

	return &CreateEventResponse{
		EventID: event.Id.String(),
		Message: "Event created successfully",
	}, nil
}

func (r *EventResolver) UpdateEvent(ctx context.Context, id string, input UpdateEventInput) (*UpdateEventResponse, error) {
	// Track telemetry
	r.telemetryAdapter.TrackEvent(ctx, "UpdateEvent", map[string]string{
		"operation": "update_event",
		"service":   "EventService",
		"event_id":  id,
	})

	// Parse UUID
	eventID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	// Get existing event
	existingEvent := r.eventService.GetEventById(eventID, ctx)
	if existingEvent == nil {
		return nil, nil
	}

	// Update fields if provided
	if input.Name != nil {
		existingEvent.Name = *input.Name
	}
	if input.Description != nil {
		existingEvent.Description = *input.Description
	}
	if input.Date != nil {
		date, err := time.Parse(time.RFC3339, *input.Date)
		if err != nil {
			return nil, err
		}
		existingEvent.Date = date
	}
	if input.Location != nil {
		existingEvent.Location = *input.Location
	}
	if input.Organizer != nil {
		existingEvent.Organizer = *input.Organizer
	}
	if input.Category != nil {
		existingEvent.Category = *input.Category
	}
	if input.Tags != nil {
		existingEvent.Tags = *input.Tags
	}
	if input.Capacity != nil {
		existingEvent.Capacity = *input.Capacity
	}

	now := time.Now()
	existingEvent.UpdatedAt = &now

	// Use shared service
	err = r.eventService.UpdateEvent(existingEvent, ctx)
	if err != nil {
		// Track error
		r.telemetryAdapter.TrackError(err, map[string]string{
			"operation": "update_event",
			"service":   "EventService",
			"event_id":  id,
		})
		return nil, err
	}

	// Convert domain model to GraphQL response
	graphQLEvent := r.convertToGraphQLEvent(existingEvent)

	return &UpdateEventResponse{
		Event:   graphQLEvent,
		Message: "Event updated successfully",
	}, nil
}

func (r *EventResolver) DeleteEvent(ctx context.Context, id string) (*DeleteEventResponse, error) {
	// Track telemetry
	r.telemetryAdapter.TrackEvent(ctx, "DeleteEvent", map[string]string{
		"operation": "delete_event",
		"service":   "EventService",
		"event_id":  id,
	})

	// Parse UUID
	eventID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	// Use shared service
	err = r.eventService.DeleteEvent(eventID, ctx)
	if err != nil {
		// Track error
		r.telemetryAdapter.TrackError(err, map[string]string{
			"operation": "delete_event",
			"service":   "EventService",
			"event_id":  id,
		})
		return nil, err
	}

	return &DeleteEventResponse{
		Message: "Event deleted successfully",
	}, nil
}

// Helper methods
func (r *EventResolver) convertToGraphQLEvent(event *domain.Event) *domain.Event {
	graphQLEvent := &domain.Event{
		Id:          event.Id,
		Name:        event.Name,
		Description: event.Description,
		Date:        event.Date,
		Location:    event.Location,
		Organizer:   event.Organizer,
		Category:    event.Category,
		Tags:        event.Tags,
		Capacity:    event.Capacity,
		CreatedBy:   event.CreatedBy,
		CreatedAt:   event.CreatedAt,
	}

	if event.UpdatedAt != nil {
		graphQLEvent.UpdatedAt = event.UpdatedAt
	}

	// Convert creator if available
	if event.Creator.ID != uuid.Nil {
		creator := &domain.User{
			ID:        event.Creator.ID,
			Email:     event.Creator.Email,
			FirstName: event.Creator.FirstName,
			LastName:  event.Creator.LastName,
			CreatedAt: event.Creator.CreatedAt,
		}

		if event.Creator.UpdatedAt != nil {
			creator.UpdatedAt = event.Creator.UpdatedAt
		}

		graphQLEvent.Creator = *creator
	}

	return graphQLEvent
}
