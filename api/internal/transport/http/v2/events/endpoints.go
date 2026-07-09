package events

import (
	"time"

	"eventify/api/internal/domain"
	"eventify/api/internal/features/events"
	"eventify/api/internal/transport/http/httperr"
	"eventify/platform/apperrors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ---- v2 DTOs. Note `organiser`, and the richer update response. -------------

type eventResponse struct {
	Date        time.Time  `json:"date"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	ID          uuid.UUID  `json:"id"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Location    string     `json:"location"`
	Organiser   string     `json:"organiser"` // v1 spells this "organizer"
	Category    string     `json:"category"`
	Tags        []string   `json:"tags"`
	Capacity    int        `json:"capacity"`
}

type listEventsResponse struct {
	Events []eventResponse `json:"events"`
	Total  int             `json:"total"`
}

type updateEventRequest struct {
	Date        time.Time `json:"date"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Organiser   string    `json:"organiser"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	Capacity    int       `json:"capacity"`
}

func toEventResponse(e domain.Event) eventResponse {
	return eventResponse{
		ID: e.ID, Name: e.Name, Description: e.Description, Location: e.Location,
		Date: e.Date, Organiser: e.Organizer, Category: e.Category, Tags: e.Tags,
		Capacity: e.Capacity, CreatedBy: e.CreatedBy, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}

// ---- endpoints -------------------------------------------------------------

// List godoc
// @Summary List events (v2)
// @Tags events
// @Produce json
// @Security BearerAuth
// @Success 200 {object} listEventsResponse
// @Router /api/v2/events [get]
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

// Update godoc
// @Summary Update an event (v2)
// @Description Same behaviour as v1; the request accepts `organiser` and the
// @Description response returns the full updated event.
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Param event body updateEventRequest true "Event payload"
// @Success 200 {object} eventResponse
// @Failure 404 {object} httperr.ErrorResponse
// @Router /api/v2/events/{id} [put]
func (c *Controller) Update(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid event id"))
	}

	var req updateEventRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.Wrap(apperrors.Invalid, "invalid request body", err))
	}

	// The v2 DTO maps onto the v1 command. No new SQL.
	if _, err := c.h.Update.Handle(ctx.UserContext(), events.UpdateEventCommand{
		EventID: id, Name: req.Name, Description: req.Description, Location: req.Location,
		Date: req.Date, Organizer: req.Organiser, Category: req.Category,
		Tags: req.Tags, Capacity: req.Capacity,
	}); err != nil {
		return httperr.Write(ctx, err)
	}

	// v2 returns the full event, so it reads back through the same Get handler.
	e, err := c.h.Get.Handle(ctx.UserContext(), events.GetEventQuery{EventID: id})
	if err != nil {
		return httperr.Write(ctx, err)
	}
	return ctx.Status(fiber.StatusOK).JSON(toEventResponse(e))
}
