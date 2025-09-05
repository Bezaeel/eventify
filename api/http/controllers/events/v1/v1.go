package v1

import (
	"eventify/internal/service"
	"eventify/internal/shared/auth"
	"eventify/pkg/logger"
	"eventify/pkg/telemetry"

	"github.com/gofiber/fiber/v2"
)

type V1EventController struct {
	router           fiber.Router
	telemetryAdapter telemetry.ITelemetryAdapter
	service          service.IEventService
	jwtProvider      auth.IJWTProvider
	log              *logger.Logger
}

func NewV1EventController(
	app *fiber.App,
	telemetryAdapter telemetry.ITelemetryAdapter,
	service service.IEventService,
	jwtProvider auth.IJWTProvider,
	log *logger.Logger,
) *V1EventController {
	return &V1EventController{
		router:           app.Group("/api/v1/events"),
		telemetryAdapter: telemetryAdapter,
		service:          service,
		jwtProvider:      jwtProvider,
		log:              log,
	}
}

func (ec *V1EventController) RegisterV1Routes() {

	// Register event routes
	ec.registerUpdateEventRoutes()
	ec.registerGetEventsRoutes()
}

// v1 godoc
// @Summary call v1
// @Description Update an existing event by its ID
// @Tags events
// @Produce json
// @Router /api/v1/events [get]
func (ec *V1EventController) index(c *fiber.Ctx) error {
	// Track the event with telemetry adapter - trace context is handled internally
	telemetryProperties := map[string]string{
		"method": "index",
	}
	ec.telemetryAdapter.TrackEvent(c.Context(), "V1EventIndexCalled", telemetryProperties)

	// Call the service directly - trace context is automatically available from middleware
	_ = ec.service.GetAllEvents(c.Context())

	return c.SendString("v1")
}

// v1 godoc
// @Summary call v1
// @Description Update an existing event by its ID
// @Tags events
// @Produce json
// @Router /api/v1/events2 [get]
func (ec *V1EventController) index2(c *fiber.Ctx) error {
	// Track the event with telemetry adapter - trace context is handled internally
	telemetryProperties := map[string]string{
		"method": "index",
	}
	ec.telemetryAdapter.TrackEvent(c.Context(), "V1EventIndexCalled", telemetryProperties)

	// Call the service directly - trace context is automatically available from middleware
	_ = ec.service.Get2AllEvents(c.Context())

	return c.SendString("v1")
}
