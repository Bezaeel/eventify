package main

import (
	"eventify/api/http/controllers/events"
	"eventify/api/http/controllers/v1"
	adminControllers "eventify/api/http/controllers/v1/admin"
	"eventify/internal/service"
	"eventify/internal/shared/auth"
	"eventify/internal/shared/config"
	"eventify/pkg/database"
	"eventify/pkg/logger"
	"eventify/pkg/telemetry"
	"os"
	"os/signal"
	"syscall"

	scalar "github.com/oSethoum/fiber-scalar"
)

// @title Eventify API
// @version 1.0
// @description This is the API documentation for Eventify.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @host localhost:3000
func main() {
	// Initialize structured logger
	log := logger.New(true)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("cannot load config")
		os.Exit(1)
	}

	// Connect to database
	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		log.Error("cannot connect to database")
		os.Exit(1)
	}

	// Initialize JWT provider
	jwtProvider := auth.NewJWTProvider(
		os.Getenv("JWT_SECRET"),
		60, // 60 minutes expiry
		"eventify",
		"eventify-api",
	)

	// add telemetry
	telemetry.AddTelemetry("eventify-api")
	telemetryAdapter := telemetry.NewTelemetryAdapter()

	// Initialize permission service

	apiHttpServer := NewAPIServer(telemetryAdapter)
	app := apiHttpServer.App()

	// Initialize repositories
	eventService := service.NewEventService(db, log, telemetryAdapter)
	userService := service.NewUserService(db)
	roleService := service.NewRoleService(db)
	permissionService := service.NewPermissionService(db)

	// Initialize controllers
	authController := controllers.NewAuthController(app, userService, jwtProvider, permissionService, log)
	authController.RegisterRoutes()

	events.NewEventController(app, telemetryAdapter, eventService, jwtProvider, log)

	// Initialize password controller
	passwordController := controllers.NewPasswordController(app, userService, jwtProvider)
	passwordController.RegisterRoutes()

	// Initialize admin controller
	adminController := adminControllers.NewAdminController(app, userService, roleService, permissionService, jwtProvider)
	adminController.RegisterRoutes()

	// Add Swagger handler
	app.Get("/docs", scalar.Handler(&scalar.Options{
		SpecURL:  "../../docs/swagger.json",
		SpecFile: "../../docs/swagger.json",
		Layout:   scalar.LayoutClassic,
		Theme:    scalar.ThemeSolarized,
		DarkMode: true,
		// other options can go here
	}))

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info("Shutting down gracefully...")
		if err := telemetry.ShutdownTracer(); err != nil {
			log.ErrorWithError("Error shutting down tracer", err)
		}
		os.Exit(0)
	}()

	// Start the server
	log.Info("Starting server on :3000")
	app.Listen(":3000")
}
