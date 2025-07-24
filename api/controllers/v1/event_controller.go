package controllers

// import (
// 	"eventify/internal/api/middlewares"
// 	"eventify/internal/auth"
// 	"eventify/internal/constants"
// 	"eventify/internal/domain"
// 	"eventify/internal/service"
// 	"eventify/pkg"
// 	"eventify/pkg/logger"

// 	"github.com/gofiber/fiber/v2"
// 	"github.com/google/uuid"
// )

// type EventController struct {
// 	router      fiber.Router
// 	service     service.IEventService
// 	jwtProvider auth.IJWTProvider
// 	log         *logger.Logger
// }

// func NewEventController(
// 	app *fiber.App,
// 	service service.IEventService,
// 	jwtProvider auth.IJWTProvider,
// 	log *logger.Logger,
// ) *EventController {
// 	return &EventController{
// 		router:      app.Group("/api/v2/events"),
// 		service:     service,
// 		jwtProvider: jwtProvider,
// 		log:         log,
// 	}
// }

// func (ec *EventController) RegisterRoutes() {
// 	// Apply JWT middleware to all routes
// 	ec.router.Use(middlewares.JWTMiddleware(ec.jwtProvider))

// 	// Apply specific permission checks to each route
// 	ec.router.Get("/",
// 		middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Read}),
// 		middlewares.AnotherMiddleware(),
// 		ec.GetEventById)

// 	ec.router.Get("/:id", middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Read}), ec.GetEventById)
// 	ec.router.Put("/:id",
// 		middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Update}),
// 		middlewares.AnotherMiddleware(),
// 		ec.UpdateEvent)

// 	ec.router.Use(middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Create})).Post("/", ec.UpdateEvent)
// 	ec.router.Delete("/:id", middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Delete}), ec.DeleteEvent)
// }

// // GetEvents godoc
// // @Summary Get events
// // @Description Get all events
// // @Tags events
// // @Produce json
// // @Security BearerAuth
// // @Success 200 {object} domain.Event
// // @Failure 400 {object} pkg.ErrorResponse
// // @Failure 404 {object} pkg.ErrorResponse
// // @Router /api/v2/events [get]
// func (ec *EventController) GetEvents(c *fiber.Ctx) error {
// 	events := ec.service.GetAllEvents(c.Context())
// 	return c.Status(fiber.StatusOK).JSON(events)
// }

// // GetEventById godoc
// // @Summary Get event by ID
// // @Description Get a single event by its ID
// // @Tags events
// // @Accept json
// // @Produce json
// // @Security BearerAuth
// // @Param id path string true "Event ID"
// // @Success 200 {object} domain.Event
// // @Failure 400 {object} pkg.ErrorResponse
// // @Failure 404 {object} pkg.ErrorResponse
// // @Router /api/v2/events/{id} [get]
// func (ec *EventController) GetEventById(c *fiber.Ctx) error {
// 	id := c.Params("id")
// 	uuid, err := uuid.Parse(id)
// 	if err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
// 			Message: "Invalid UUID",
// 		})
// 	}
// 	event := ec.service.GetEventById(uuid, c.Context())
// 	if event == nil {
// 		return c.Status(fiber.StatusNotFound).JSON(pkg.ErrorResponse{
// 			Message: "event by Id not found",
// 		})
// 	}
// 	return c.Status(fiber.StatusOK).JSON(event)
// }

// // UpdateEvent godoc
// // @Summary Update an event
// // @Description Update an existing event by its ID
// // @Tags events
// // @Accept json
// // @Produce json
// // @Security BearerAuth
// // @Param id path string true "Event ID"
// // @Param event body domain.Event true "Event payload"
// // @Success 200 {object} domain.Event
// // @Failure 400 {object} pkg.ErrorResponse
// // @Failure 500 {object} pkg.ErrorResponse
// // @Router /api/v2/events/{id} [put]
// func (ec *EventController) UpdateEvent(c *fiber.Ctx) error {
// 	id := c.Params("id")
// 	uuid, err := uuid.Parse(id)
// 	if err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
// 			Message: "Invalid UUID",
// 		})
// 	}
// 	event := new(domain.Event)
// 	if err := c.BodyParser(event); err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
// 			Message: err.Error(),
// 		})
// 	}
// 	event.Id = uuid
// 	if err := ec.service.UpdateEvent(event, c.Context()); err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
// 			Message: err.Error(),
// 		})
// 	}
// 	return c.Status(fiber.StatusOK).JSON(event)
// }

// // DeleteEvent godoc
// // @Summary Delete an event
// // @Description Delete an event by its ID
// // @Tags events
// // @Accept json
// // @Produce json
// // @Security BearerAuth
// // @Param id path string true "Event ID"
// // @Success 204
// // @Failure 400 {object} pkg.ErrorResponse
// // @Failure 500 {object} pkg.ErrorResponse
// // @Router /api/v1/events/{id} [delete]
// func (ec *EventController) DeleteEvent(c *fiber.Ctx) error {
// 	id := c.Params("id")
// 	uuid, err := uuid.Parse(id)
// 	if err != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(pkg.ErrorResponse{
// 			Message: "Invalid UUID",
// 		})
// 	}
// 	if err := ec.service.DeleteEvent(uuid, c.Context()); err != nil {
// 		return c.Status(fiber.StatusInternalServerError).JSON(pkg.ErrorResponse{
// 			Message: err.Error(),
// 		})
// 	}
// 	return c.SendStatus(fiber.StatusNoContent)
// }