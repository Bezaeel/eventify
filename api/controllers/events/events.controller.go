package events

import (
	v1 "eventify/api/controllers/events/v1"
	v2 "eventify/api/controllers/events/v2"
	"eventify/internal/auth"
	"eventify/internal/service"
	"eventify/pkg/logger"

	"github.com/gofiber/fiber/v2"
)

type EventController struct {
	Router      fiber.Router
	Service     service.IEventService
	JwtProvider auth.IJWTProvider
	Log         *logger.Logger
}

func NewEventController(
	app *fiber.App,
	service service.IEventService,
	jwtProvider auth.IJWTProvider,
	log *logger.Logger,
) {
	v1Controller := v1.NewV1EventController(app, service, jwtProvider, log)
	v1Controller.RegisterV1Routes()

	v2Controller := v2.NewV2EventController(app, service, jwtProvider, log)
	v2Controller.RegisterV2Routes()
}
