package v2

import (
	"eventify/api/http/middlewares"
	"eventify/internal/shared/constants"
	"eventify/internal/domain"

	"eventify/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (ec *v2EventController) registerUpdateEventRoutes() {
	ec.router.Put("/:id",
		middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Update}),
		middlewares.AnotherMiddleware(),
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
// @Router /api/v2/events/{id} [put]
func (ec *v2EventController) UpdateEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: "Invalid UUID",
		})
	}
	event := new(domain.Event)
	if err := c.BodyParser(event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
			Message: err.Error(),
		})
	}
	event.Id = uuid
	if err := ec.service.UpdateEvent(event, c.Context()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(event)
}

// dtos
type CreateEventRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Date        string   `json:"date" validate:"required"`
	Location    string   `json:"location" validate:"required"`
	Organizer   string   `json:"organizer" validate:"required"`
	Category    string   `json:"category" validate:"required"`
	Tags        []string `json:"tags" validate:"dive,required"`
	Capacity    int      `json:"capacity" validate:"required,min=1"`
}

type CreateEventResponse struct {
	EventID uuid.UUID `json:"event_id"`
}

// mapper
// validator
