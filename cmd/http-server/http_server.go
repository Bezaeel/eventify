package main

import (
	"eventify/api/http/middlewares"
	"eventify/pkg/telemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type httpServer struct {
	app *fiber.App
}

func NewAPIServer(telemetryAdapter telemetry.ITelemetryAdapter) *httpServer {
	app := fiber.New()

	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(middlewares.TelemetryMiddleware(telemetryAdapter))

	return &httpServer{app: app}
}

func (s *httpServer) App() *fiber.App {
	return s.app
}
