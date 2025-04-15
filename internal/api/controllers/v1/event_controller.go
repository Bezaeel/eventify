package controllers

import (
	"eventify/internal/api/middlewares"
	"eventify/internal/auth"
	"eventify/internal/constants"
	"eventify/internal/domain"
	"eventify/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type EventController struct {
	router      fiber.Router
	service     service.IEventService
	jwtProvider auth.IJWTProvider
}

func NewEventController(
	app *fiber.App,
	service service.IEventService,
	jwtProvider auth.IJWTProvider,
) *EventController {
	return &EventController{
		router:      app.Group("/api/v1/events"),
		service:     service,
		jwtProvider: jwtProvider,
	}
}

func (ec *EventController) RegisterRoutes() {
	// Apply JWT middleware to all routes
	ec.router.Use(middlewares.JWTMiddleware(ec.jwtProvider))

	// Apply specific permission checks to each route
	ec.router.Get("/", middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Read}), ec.GetAllEvents)
	ec.router.Get("/:id", middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Read}), ec.GetEventById)
	ec.router.Post("/", middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Create}), ec.CreateEvent)
	ec.router.Put("/:id", middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Update}), ec.UpdateEvent)
	ec.router.Delete("/:id", middlewares.HasPermission([]string{constants.Permissions.EventPermissions.Delete}), ec.DeleteEvent)
}

// GetAllEvents godoc
// @Summary Get all events
// @Description Get a list of all events
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.Event
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/events [get]
func (ec *EventController) GetAllEvents(c *fiber.Ctx) error {
	events := ec.service.GetAllEvents()
	return c.Status(fiber.StatusOK).JSON(events)
}

// GetEventById godoc
// @Summary Get event by ID
// @Description Get a single event by its ID
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Success 200 {object} domain.Event
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/events/{id} [get]
func (ec *EventController) GetEventById(c *fiber.Ctx) error {
	id := c.Params("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid UUID",
		})
	}
	event := ec.service.GetEventById(uuid)
	if event == nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Message: "event by Id not found",
		})
	}
	return c.Status(fiber.StatusOK).JSON(event)
}

// CreateEvent godoc
// @Summary Create a new event
// @Description Create a new event with the input payload
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param event body domain.Event true "Event payload"
// @Success 201 {object} domain.Event
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/events [post]
func (ec *EventController) CreateEvent(c *fiber.Ctx) error {
	event := new(domain.Event)
	if err := c.BodyParser(event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: err.Error(),
		})
	}
	if err := ec.service.CreateEvent(event); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(event)
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
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/events/{id} [put]
func (ec *EventController) UpdateEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid UUID",
		})
	}
	event := new(domain.Event)
	if err := c.BodyParser(event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: err.Error(),
		})
	}
	event.Id = uuid
	if err := ec.service.UpdateEvent(event); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(event)
}

// DeleteEvent godoc
// @Summary Delete an event
// @Description Delete an event by its ID
// @Tags events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/events/{id} [delete]
func (ec *EventController) DeleteEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	uuid, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Message: "Invalid UUID",
		})
	}
	if err := ec.service.DeleteEvent(uuid); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Message: err.Error(),
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
