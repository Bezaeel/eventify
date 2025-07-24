package v1

import (
	"eventify/internal/auth"
	"eventify/internal/service"
	"eventify/pkg/logger"

	"github.com/gofiber/fiber/v2"
)

type v1EventController struct {
	router      fiber.Router
	service     service.IEventService
	jwtProvider auth.IJWTProvider
	log         *logger.Logger
}

func NewV1EventController(
	app *fiber.App,
	service service.IEventService,
	jwtProvider auth.IJWTProvider,
	log *logger.Logger,
) *v1EventController {
	return &v1EventController{
		router:      app.Group("/api/v1/events"),
		service:     service,
		jwtProvider: jwtProvider,
		log:         log,
	}
}

func (ec *v1EventController) RegisterV1Routes() {

	
	ec.router.Get("/", ec.index)

	// Register event routes
	ec.registerUpdateEventRoutes()
}

// v1 godoc
// @Summary call v1
// @Description Update an existing event by its ID
// @Tags events
// @Produce json
// @Router /api/v1/events [get]
func (ec *v1EventController) index(c *fiber.Ctx) error {
	return c.SendString("v1")
}

