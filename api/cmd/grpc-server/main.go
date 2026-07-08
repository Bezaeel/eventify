package main

import (
	"eventify/api/grpc/handlers"
	"eventify/api/grpc/proto"
	"eventify/internal/service"
	"eventify/internal/shared/config"
	"eventify/pkg/database"
	"eventify/pkg/logger"
	"eventify/pkg/telemetry"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	telemetry.AddTelemetry("eventify-grpc")
	telemetryAdapter := telemetry.NewTelemetryAdapter()

	// Initialize services
	eventService := service.NewEventService(db, log, telemetryAdapter)

	// Initialize gRPC server
	grpcServer := grpc.NewServer()

	// Register services
	eventHandler := handlers.NewEventHandler(eventService, telemetryAdapter)
	proto.RegisterEventServiceServer(grpcServer, eventHandler)

	// Enable reflection for development
	reflection.Register(grpcServer)

	// Start server
	port := ":3002"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Error("failed to listen")
		os.Exit(1)
	}

	log.Info(fmt.Sprintf("Starting gRPC server on %s", port))

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info("Shutting down gRPC server gracefully...")
		grpcServer.GracefulStop()
		if err := telemetry.ShutdownTracer(); err != nil {
			log.ErrorWithError("Error shutting down tracer", err)
		}
		os.Exit(0)
	}()

	// Start the server
	if err := grpcServer.Serve(lis); err != nil {
		log.Error("failed to serve")
		os.Exit(1)
	}
}
