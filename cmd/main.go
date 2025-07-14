package main

import (
	"eventify/internal/api/controllers/v1"
	adminControllers "eventify/internal/api/controllers/v1/admin"
	"eventify/internal/auth"
	"eventify/internal/config"
	"eventify/internal/service"
	"eventify/pkg/database"
	"eventify/pkg/logger"
	"os"

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

	// Initialize permission service

	apiHttpServer := NewAPIServer()
	app := apiHttpServer.App()

	// Initialize repositories
	eventService := service.NewEventService(db, log)
	userService := service.NewUserService(db)
	roleService := service.NewRoleService(db)
	permissionService := service.NewPermissionService(db)

	// Initialize controllers
	authController := controllers.NewAuthController(app, userService, jwtProvider, permissionService, log)
	authController.RegisterRoutes()

	eventController := controllers.NewEventController(app, eventService, jwtProvider, log)
	eventController.RegisterRoutes()

	// Initialize password controller
	passwordController := controllers.NewPasswordController(app, userService, jwtProvider)
	passwordController.RegisterRoutes()

	// Initialize admin controller
	adminController := adminControllers.NewAdminController(app, userService, roleService, permissionService, jwtProvider)
	adminController.RegisterRoutes()

	// Add Swagger handler
	app.Get("/docs", scalar.Handler(&scalar.Options{
						SpecURL: "../docs/swagger.json",
            SpecFile: "../docs/swagger.json",
            Layout:   scalar.LayoutClassic,
            Theme:    scalar.ThemeSolarized,
            DarkMode: true,
            // other options can go here
        }))

	// Start the server
	app.Listen(":3000")
}
