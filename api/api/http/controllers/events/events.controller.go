package events

import (
	v1 "eventify/api/http/controllers/events/v1"
	v2 "eventify/api/http/controllers/events/v2"
	"eventify/internal/shared/auth"
	"eventify/internal/service"
	"eventify/pkg/logger"
	"eventify/pkg/telemetry"

	"github.com/gofiber/fiber/v2"
)

type EventController struct {
	Router      fiber.Router
	Telemetry   telemetry.ITelemetryAdapter
	Service     service.IEventService
	JwtProvider auth.IJWTProvider
	Log         *logger.Logger
}

func NewEventController(
	app *fiber.App,
	telemetryAdapter telemetry.ITelemetryAdapter,
	service service.IEventService,
	jwtProvider auth.IJWTProvider,
	log *logger.Logger,
) {
	v1Controller := v1.NewV1EventController(app, telemetryAdapter, service, jwtProvider, log)
	v1Controller.RegisterV1Routes()

	v2Controller := v2.NewV2EventController(app, service, jwtProvider, log)
	v2Controller.RegisterV2Routes()
}
