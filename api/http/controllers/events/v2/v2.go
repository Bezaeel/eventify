package v2

import (
	"eventify/internal/shared/auth"
	"eventify/internal/service"
	"eventify/pkg/logger"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type v2EventController struct {
	router      fiber.Router
	service     service.IEventService
	jwtProvider auth.IJWTProvider
	log         *logger.Logger
}

func NewV2EventController(
	app *fiber.App,
	service service.IEventService,
	jwtProvider auth.IJWTProvider,
	log *logger.Logger,
) *v2EventController {
	return &v2EventController{
		router:      app.Group("/api/v2/events"),
		service:     service,
		jwtProvider: jwtProvider,
		log:         log,
	}
}

func (ec *v2EventController) RegisterV2Routes() {
	ec.router.Get("/", ec.index)

	// Register event routes
	ec.registerUpdateEventRoutes()
}

// v2 godoc
// @Summary call v2
// @Description Update an existing event by its ID
// @Tags events
// @Produce json
// @Router /api/v2/events [get]
func (ec *v2EventController) index(c *fiber.Ctx) error {
	propagator := otel.GetTextMapPropagator()
		extractCtx := propagator.Extract(c.UserContext(), propagation.HeaderCarrier(c.GetReqHeaders()))

		fmt.Printf("Your correlationId is %v", c.Locals("correlationId"))

		tracer := otel.Tracer("Order Api")
		_, span := tracer.Start(extractCtx, "GetOrderById Controller")
		defer span.End()

		traceParent := c.Get("traceparent")
		fmt.Println("traceParent: " + traceParent)
	return c.SendString("v1")
}
