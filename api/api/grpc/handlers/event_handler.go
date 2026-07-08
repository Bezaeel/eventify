package handlers

import (
	"context"

	"eventify/api/grpc/proto"
	"eventify/internal/domain"
	"eventify/internal/service"
	"eventify/pkg/telemetry"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventHandler struct {
	proto.UnimplementedEventServiceServer
	eventService     *service.EventService
	telemetryAdapter telemetry.ITelemetryAdapter
}

func NewEventHandler(eventService *service.EventService, telemetryAdapter telemetry.ITelemetryAdapter) *EventHandler {
	return &EventHandler{
		eventService:     eventService,
		telemetryAdapter: telemetryAdapter,
	}
}

func (h *EventHandler) CreateEvent(ctx context.Context, req *proto.CreateEventRequest) (*proto.CreateEventResponse, error) {
	// Track telemetry
	h.telemetryAdapter.TrackEvent(ctx, "CreateEvent", map[string]string{
		"operation": "create_event",
		"service":   "EventService",
	})

	// Convert gRPC request to domain model
	event := &domain.Event{
		Id:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Date:        req.Date.AsTime(),
		Location:    req.Location,
		Organizer:   req.Organizer,
		Category:    req.Category,
		Tags:        req.Tags,
		Capacity:    int(req.Capacity),
		CreatedBy:   uuid.New(), // TODO: Get from context
		CreatedAt:   time.Now(),
	}

	// Use shared service
	err := h.eventService.CreateEvent(event, ctx)
	if err != nil {
		// Track error
		h.telemetryAdapter.TrackError(err, map[string]string{
			"operation": "create_event",
			"service":   "EventService",
		})
		return nil, status.Errorf(codes.Internal, "failed to create event: %v", err)
	}

	return &proto.CreateEventResponse{
		EventId: event.Id.String(),
		Message: "Event created successfully",
	}, nil
}

func (h *EventHandler) GetEvent(ctx context.Context, req *proto.GetEventRequest) (*proto.GetEventResponse, error) {
	// Track telemetry
	h.telemetryAdapter.TrackEvent(ctx, "GetEvent", map[string]string{
		"operation": "get_event",
		"service":   "EventService",
		"event_id":  req.EventId,
	})

	// Parse UUID
	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event ID: %v", err)
	}

	// Use shared service
	event := h.eventService.GetEventById(eventID, ctx)
	if event == nil {
		return nil, status.Errorf(codes.NotFound, "event not found")
	}

	// Convert domain model to gRPC response
	protoEvent := h.convertToProtoEvent(event)

	return &proto.GetEventResponse{
		Event: protoEvent,
	}, nil
}

func (h *EventHandler) UpdateEvent(ctx context.Context, req *proto.UpdateEventRequest) (*proto.UpdateEventResponse, error) {
	// Track telemetry
	h.telemetryAdapter.TrackEvent(ctx, "UpdateEvent", map[string]string{
		"operation": "update_event",
		"service":   "EventService",
		"event_id":  req.EventId,
	})

	// Parse UUID
	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event ID: %v", err)
	}

	// Convert gRPC request to domain model
	event := &domain.Event{
		Id:          eventID,
		Name:        req.Name,
		Description: req.Description,
		Date:        req.Date.AsTime(),
		Location:    req.Location,
		Organizer:   req.Organizer,
		Category:    req.Category,
		Tags:        req.Tags,
		Capacity:    int(req.Capacity),
		UpdatedAt:   &time.Time{},
	}

	// Use shared service
	err = h.eventService.UpdateEvent(event, ctx)
	if err != nil {
		// Track error
		h.telemetryAdapter.TrackError(err, map[string]string{
			"operation": "update_event",
			"service":   "EventService",
			"event_id":  req.EventId,
		})
		return nil, status.Errorf(codes.Internal, "failed to update event: %v", err)
	}

	// Convert domain model to gRPC response
	protoEvent := h.convertToProtoEvent(event)

	return &proto.UpdateEventResponse{
		Event:   protoEvent,
		Message: "Event updated successfully",
	}, nil
}

func (h *EventHandler) DeleteEvent(ctx context.Context, req *proto.DeleteEventRequest) (*proto.DeleteEventResponse, error) {
	// Track telemetry
	h.telemetryAdapter.TrackEvent(ctx, "DeleteEvent", map[string]string{
		"operation": "delete_event",
		"service":   "EventService",
		"event_id":  req.EventId,
	})

	// Parse UUID
	eventID, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event ID: %v", err)
	}

	// Use shared service
	err = h.eventService.DeleteEvent(eventID, ctx)
	if err != nil {
		// Track error
		h.telemetryAdapter.TrackError(err, map[string]string{
			"operation": "delete_event",
			"service":   "EventService",
			"event_id":  req.EventId,
		})
		return nil, status.Errorf(codes.Internal, "failed to delete event: %v", err)
	}

	return &proto.DeleteEventResponse{
		Message: "Event deleted successfully",
	}, nil
}

func (h *EventHandler) ListEvents(ctx context.Context, req *proto.ListEventsRequest) (*proto.ListEventsResponse, error) {
	// Track telemetry
	h.telemetryAdapter.TrackEvent(ctx, "ListEvents", map[string]string{
		"operation": "list_events",
		"service":   "EventService",
	})

	// Use shared service
	events := h.eventService.GetAllEvents(ctx)

	// Convert domain models to gRPC responses
	protoEvents := make([]*proto.Event, len(events))
	for i, event := range events {
		protoEvents[i] = h.convertToProtoEvent(&event)
	}

	return &proto.ListEventsResponse{
		Events: protoEvents,
		Total:  int32(len(events)),
		Page:   req.Page,
		Limit:  req.Limit,
	}, nil
}

func (h *EventHandler) convertToProtoEvent(event *domain.Event) *proto.Event {
	protoEvent := &proto.Event{
		Id:          event.Id.String(),
		Name:        event.Name,
		Description: event.Description,
		Date:        timestamppb.New(event.Date),
		Location:    event.Location,
		Organizer:   event.Organizer,
		Category:    event.Category,
		Tags:        event.Tags,
		Capacity:    int32(event.Capacity),
		CreatedBy:   event.CreatedBy.String(),
		CreatedAt:   timestamppb.New(event.CreatedAt),
	}

	if event.UpdatedAt != nil {
		protoEvent.UpdatedAt = timestamppb.New(*event.UpdatedAt)
	}

	return protoEvent
}
