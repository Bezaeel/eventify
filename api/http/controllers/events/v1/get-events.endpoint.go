package v1

import (
	"eventify/api/http/middlewares"

	"github.com/gofiber/fiber/v2"
)

func (ec *V1EventController) registerGetEventsRoutes() {
	ec.router.Get("/",
		middlewares.JWTMiddleware(ec.jwtProvider),
		ec.GetEvents)
}

// GetEvents godoc
// @Summary Get events
// @Description Get all events
// @Tags events
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.Event
// @Failure 400 {object} pkg.ErrorResponse
// @Failure 404 {object} pkg.ErrorResponse
// @Router /api/v2/events [get]
func (ec *V1EventController) GetEvents(c *fiber.Ctx) error {
	events := ec.service.GetAllEvents(c.Context())
	return c.Status(fiber.StatusOK).JSON(events)
}
