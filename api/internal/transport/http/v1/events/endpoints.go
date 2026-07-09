package events

import (
	"time"

	"eventify/api/internal/domain"
	"eventify/api/internal/features/events"
	"eventify/api/internal/transport/http/httperr"
	"eventify/api/internal/transport/http/middleware"
	"eventify/platform/apperrors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ---- DTOs. These are the v1 wire contract. Changing one is a v2. ------------

type eventResponse struct {
	Date        time.Time  `json:"date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	ID          uuid.UUID  `json:"id"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Location    string     `json:"location"`
	Organizer   string     `json:"organizer"`
	Category    string     `json:"category"`
	Tags        []string   `json:"tags"`
	Capacity    int        `json:"capacity"`
}

type listEventsResponse struct {
	Events []eventResponse `json:"events"`
	Total  int             `json:"total"`
}

type writeEventRequest struct {
	Date        time.Time `json:"date"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Organizer   string    `json:"organizer"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	Capacity    int       `json:"capacity"`
}

type createEventResponse struct {
	CreatedAt time.Time `json:"created_at"`
	EventID   uuid.UUID `json:"event_id"`
}

type updateEventResponse struct {
	UpdatedAt time.Time `json:"updated_at"`
	EventID   uuid.UUID `json:"event_id"`
}

func toEventResponse(e domain.Event) eventResponse {
	return eventResponse{
		ID: e.ID, Name: e.Name, Description: e.Description, Location: e.Location,
		Date: e.Date, Organizer: e.Organizer, Category: e.Category, Tags: e.Tags,
		Capacity: e.Capacity, CreatedBy: e.CreatedBy, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}

// ---- endpoints -------------------------------------------------------------

// List godoc
// @Summary List events
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Page size (default 50, max 200)"
// @Param offset query int false "Rows to skip"
// @Success 200 {object} listEventsResponse
// @Router /api/v1/events [get]
func (c *Controller) List(ctx *fiber.Ctx) error {
	res, err := c.h.List.Handle(ctx.UserContext(), events.GetEventsQuery{
		Limit:  ctx.QueryInt("limit", 0),
		Offset: ctx.QueryInt("offset", 0),
	})
	if err != nil {
		return httperr.Write(ctx, err)
	}

	out := listEventsResponse{Events: make([]eventResponse, 0, len(res.Events)), Total: res.Total}
	for _, e := range res.Events {
		out.Events = append(out.Events, toEventResponse(e))
	}
	return ctx.Status(fiber.StatusOK).JSON(out)
}

// Get godoc
// @Summary Get an event
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Success 200 {object} eventResponse
// @Failure 404 {object} httperr.ErrorResponse
// @Router /api/v1/events/{id} [get]
func (c *Controller) Get(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid event id"))
	}

	e, err := c.h.Get.Handle(ctx.UserContext(), events.GetEventQuery{EventID: id})
	if err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(toEventResponse(e))
}

// Create godoc
// @Summary Create an event
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param event body writeEventRequest true "Event payload"
// @Success 201 {object} createEventResponse
// @Failure 400 {object} httperr.ErrorResponse
// @Router /api/v1/events [post]
func (c *Controller) Create(ctx *fiber.Ctx) error {
	var req writeEventRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.Wrap(apperrors.Invalid, "invalid request body", err))
	}

	claims, ok := middleware.Claims(ctx)
	if !ok {
		return httperr.Write(ctx, apperrors.New(apperrors.Unauthorized, "authentication required"))
	}

	res, err := c.h.Create.Handle(ctx.UserContext(), events.CreateEventCommand{
		Name: req.Name, Description: req.Description, Location: req.Location,
		Date: req.Date, Organizer: req.Organizer, Category: req.Category,
		Tags: req.Tags, Capacity: req.Capacity,
		// The gRPC handler and GraphQL resolver both set CreatedBy to
		// uuid.New() with a `TODO: Get from context`, attributing every event
		// to a user that does not exist.
		CreatedBy: claims.UserID,
	})
	if err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(createEventResponse{EventID: res.EventID, CreatedAt: res.CreatedAt})
}

// Update godoc
// @Summary Update an event
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Param event body writeEventRequest true "Event payload"
// @Success 200 {object} updateEventResponse
// @Failure 400 {object} httperr.ErrorResponse
// @Failure 404 {object} httperr.ErrorResponse
// @Router /api/v1/events/{id} [put]
func (c *Controller) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid event id"))
	}

	var req writeEventRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.Wrap(apperrors.Invalid, "invalid request body", err))
	}

	res, err := c.h.Update.Handle(ctx.UserContext(), events.UpdateEventCommand{
		EventID: id, Name: req.Name, Description: req.Description, Location: req.Location,
		Date: req.Date, Organizer: req.Organizer, Category: req.Category,
		Tags: req.Tags, Capacity: req.Capacity,
	})
	if err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(updateEventResponse{EventID: res.EventID, UpdatedAt: res.UpdatedAt})
}

// Delete godoc
// @Summary Delete an event
// @Tags events
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Success 204
// @Failure 404 {object} httperr.ErrorResponse
// @Router /api/v1/events/{id} [delete]
func (c *Controller) Delete(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid event id"))
	}

	if err := c.h.Delete.Handle(ctx.UserContext(), events.DeleteEventCommand{EventID: id}); err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
