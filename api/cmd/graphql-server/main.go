package main

import (
	"eventify/api/graphql/generated"
	"eventify/api/graphql/resolvers"
	"eventify/internal/service"
	"eventify/internal/shared/config"
	"eventify/pkg/database"
	"eventify/pkg/logger"
	"eventify/pkg/telemetry"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
)

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

	// Initialize telemetry
	telemetry.AddTelemetry("eventify-graphql")
	telemetryAdapter := telemetry.NewTelemetryAdapter()

	// Initialize services
	eventService := service.NewEventService(db, log, telemetryAdapter)

	// Initialize GraphQL resolvers
	resolver := resolvers.NewResolver(eventService, log, telemetryAdapter)

	// Create GraphQL schema using generated code
	schema := generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	})
	srv := handler.NewDefaultServer(schema)

	// Setup router
	router := mux.NewRouter()
	router.Handle("/", playground.Handler("GraphQL playground", "/query"))
	router.Handle("/query", srv)

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info("Shutting down GraphQL server gracefully...")
		if err := telemetry.ShutdownTracer(); err != nil {
			log.ErrorWithError("Error shutting down tracer", err)
		}
		os.Exit(0)
	}()

	// Start server
	port := ":3001"
	log.Info(fmt.Sprintf("Starting GraphQL server on %s", port))
	if err := http.ListenAndServe(port, router); err != nil {
		log.Error("Failed to start GraphQL server")
		os.Exit(1)
	}
}

