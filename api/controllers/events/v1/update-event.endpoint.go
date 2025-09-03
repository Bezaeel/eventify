package v1

import (
	"eventify/api/middlewares"
	"eventify/internal/constants"
	"eventify/internal/domain"
	"time"

	"eventify/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (ec *V1EventController) registerUpdateEventRoutes() {
	ec.router.Put("/:id",
		middlewares.JWTMiddleware(ec.jwtProvider),
		middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Update}),
		// middlewares.AnotherMiddleware(),
		ec.UpdateEvent)
}

// UpdateEvent godoc
// @Summary Update an event
// @Description Update an existing event by its ID
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Param event body domain.Event true "Event payload"
// @Success 200 {object} domain.Event
// @Failure 400 {object} pkg.ErrorResponse
// @Failure 500 {object} pkg.ErrorResponse
// @Router /api/v1/events/{id} [put]
func (ec *V1EventController) UpdateEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	eventId, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid UUID",
		})
	}
	request := new(UpdateEventRequest)
	if err := c.BodyParser(request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: err.Error(),
		})
	}
	request.Id = eventId
	event := mapUpdateEventRequestToEventEntity(request)
	if err := ec.service.UpdateEvent(event, c.Context()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: err.Error(),
		})
	}
	response := mapEventEntityToUpdateEventResponse(event)
	return c.Status(fiber.StatusOK).JSON(response)
}

// dtos
type UpdateEventRequest struct {
	Id          uuid.UUID `json:"id" validate:"required,uuid"`
	Name        string    `json:"name" validate:"required"`
	Description string    `json:"description" validate:"required"`
	Date        time.Time `json:"date" validate:"required"`
	Location    string    `json:"location" validate:"required"`
	Organizer   string    `json:"organizer" validate:"required"`
	Category    string    `json:"category" validate:"required"`
	Tags        []string  `json:"tags" validate:"dive,required"`
	Capacity    int       `json:"capacity" validate:"required,min=1"`
}

type UpdateEventResponse struct {
	EventID uuid.UUID `json:"event_id"`
}

// mapper
func mapEventEntityToUpdateEventResponse(event *domain.Event) *UpdateEventResponse {
	return &UpdateEventResponse{
		EventID: event.Id,
	}
}

func mapUpdateEventRequestToEventEntity(req *UpdateEventRequest) *domain.Event {
	now := time.Now()

	return &domain.Event{
		Name:      req.Name,
		Date:      req.Date,
		Location:  req.Location,
		UpdatedAt: &now,
	}
}

// validator
