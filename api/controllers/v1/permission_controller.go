package controllers

import (
	"eventify/internal/auth"
	"eventify/internal/domain"

	"github.com/gofiber/fiber/v2"
)

type PermissionController struct {
	router      fiber.Router
	service     domain.IPermissionService
	jwtProvider auth.IJWTProvider
}

func NewPermissionController(
	app *fiber.App,
	service domain.IPermissionService,
	jwtProvider auth.IJWTProvider,
) *PermissionController {
	return &PermissionController{
		router:      app.Group("/api/v1/permissions"),
		service:     service,
		jwtProvider: jwtProvider,
	}
}
