// Package handlers is the gRPC adapter for the event use cases.
//
// It decodes proto messages into commands, calls the same handlers the HTTP and
// GraphQL adapters call, and maps apperrors.Kind onto codes.Code. No SQL here.
package handlers

import (
	"context"

	"eventify/api/internal/domain"
	"eventify/api/internal/features/events"
	"eventify/api/internal/shared/constants"
	"eventify/api/internal/transport/grpc/interceptors"
	"eventify/api/internal/transport/grpc/proto"
	"eventify/platform/apperrors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handlers are the use cases this service exposes, as function values so a test
// can inject a stub and assert the codes.Code mapping without a database.
type Handlers struct {
	Create func(context.Context, events.CreateEventCommand) (events.CreateEventResult, error)
	Update func(context.Context, events.UpdateEventCommand) (events.UpdateEventResult, error)
	Get    func(context.Context, events.GetEventQuery) (domain.Event, error)
	List   func(context.Context, events.GetEventsQuery) (events.GetEventsResult, error)
	Delete func(context.Context, events.DeleteEventCommand) error
}

// EventHandler implements proto.EventServiceServer.
type EventHandler struct {
	proto.UnimplementedEventServiceServer
	h Handlers
}

// NewEventHandler builds the gRPC service.
func NewEventHandler(h Handlers) *EventHandler { return &EventHandler{h: h} }

// grpcError maps a transport-agnostic error onto a gRPC status.
//
// The counterpart of httperr.Status. A feature handler returns a Kind; each
// transport decides what that means on its own wire. Internal errors do not
// leak the driver message — the old handlers formatted `%v` of the raw error
// into the status, exposing table and column names to clients.
func grpcError(err error) error {
	switch apperrors.KindOf(err) {
	case apperrors.NotFound:
		return status.Error(codes.NotFound, err.Error())
	case apperrors.Invalid:
		return status.Error(codes.InvalidArgument, err.Error())
	case apperrors.Conflict:
		return status.Error(codes.AlreadyExists, err.Error())
	case apperrors.Unauthorized:
		return status.Error(codes.Unauthenticated, err.Error())
	case apperrors.Forbidden:
		return status.Error(codes.PermissionDenied, err.Error())
	case apperrors.Internal:
		return status.Error(codes.Internal, "internal error")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func toProtoEvent(e domain.Event) *proto.Event {
	pe := &proto.Event{
		Id:          e.ID.String(),
		Name:        e.Name,
		Description: e.Description,
		Date:        timestamppb.New(e.Date),
		Location:    e.Location,
		Organizer:   e.Organizer,
		Category:    e.Category,
		Tags:        e.Tags,
		Capacity:    int32(e.Capacity),
		CreatedBy:   e.CreatedBy.String(),
		CreatedAt:   timestamppb.New(e.CreatedAt),
	}
	if e.UpdatedAt != nil {
		pe.UpdatedAt = timestamppb.New(*e.UpdatedAt)
	}
	return pe
}

// CreateEvent creates an event, attributed to the authenticated caller.
func (h *EventHandler) CreateEvent(ctx context.Context, req *proto.CreateEventRequest) (*proto.CreateEventResponse, error) {
	if err := interceptors.RequirePermission(ctx, constants.Permissions.EventPermissions.Create); err != nil {
		return nil, err
	}
	claims, _ := interceptors.Claims(ctx)

	res, err := h.h.Create(ctx, events.CreateEventCommand{
		Name: req.Name, Description: req.Description, Date: req.Date.AsTime(),
		Location: req.Location, Organizer: req.Organizer, Category: req.Category,
		Tags: req.Tags, Capacity: int(req.Capacity),
		// From the token, not uuid.New().
		CreatedBy: claims.UserID,
	})
	if err != nil {
		return nil, grpcError(err)
	}
	return &proto.CreateEventResponse{EventId: res.EventID.String(), Message: "event created"}, nil
}

// GetEvent fetches one event.
func (h *EventHandler) GetEvent(ctx context.Context, req *proto.GetEventRequest) (*proto.GetEventResponse, error) {
	id, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event id")
	}

	e, err := h.h.Get(ctx, events.GetEventQuery{EventID: id})
	if err != nil {
		return nil, grpcError(err)
	}
	return &proto.GetEventResponse{Event: toProtoEvent(e)}, nil
}

// UpdateEvent updates an event.
func (h *EventHandler) UpdateEvent(ctx context.Context, req *proto.UpdateEventRequest) (*proto.UpdateEventResponse, error) {
	if err := interceptors.RequirePermission(ctx, constants.Permissions.EventPermissions.Update); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event id")
	}

	if _, err := h.h.Update(ctx, events.UpdateEventCommand{
		EventID: id, Name: req.Name, Description: req.Description, Date: req.Date.AsTime(),
		Location: req.Location, Organizer: req.Organizer, Category: req.Category,
		Tags: req.Tags, Capacity: int(req.Capacity),
	}); err != nil {
		return nil, grpcError(err)
	}

	e, err := h.h.Get(ctx, events.GetEventQuery{EventID: id})
	if err != nil {
		return nil, grpcError(err)
	}
	return &proto.UpdateEventResponse{Event: toProtoEvent(e), Message: "event updated"}, nil
}

// DeleteEvent removes an event.
func (h *EventHandler) DeleteEvent(ctx context.Context, req *proto.DeleteEventRequest) (*proto.DeleteEventResponse, error) {
	if err := interceptors.RequirePermission(ctx, constants.Permissions.EventPermissions.Delete); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.EventId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid event id")
	}

	if err := h.h.Delete(ctx, events.DeleteEventCommand{EventID: id}); err != nil {
		return nil, grpcError(err)
	}
	return &proto.DeleteEventResponse{Message: "event deleted"}, nil
}

// ListEvents pages through events.
func (h *EventHandler) ListEvents(ctx context.Context, req *proto.ListEventsRequest) (*proto.ListEventsResponse, error) {
	page, limit := int(req.Page), int(req.Limit)
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}

	res, err := h.h.List(ctx, events.GetEventsQuery{Limit: limit, Offset: (page - 1) * limit})
	if err != nil {
		return nil, grpcError(err)
	}

	out := make([]*proto.Event, 0, len(res.Events))
	for _, e := range res.Events {
		out = append(out, toProtoEvent(e))
	}
	return &proto.ListEventsResponse{
		Events: out, Total: int32(res.Total), Page: int32(page), Limit: int32(limit),
	}, nil
}
