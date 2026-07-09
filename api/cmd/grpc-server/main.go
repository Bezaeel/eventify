// Command grpc-server serves the gRPC API.
package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"eventify/api/internal/features/events"
	"eventify/api/internal/shared/auth"
	"eventify/api/internal/shared/config"
	"eventify/api/internal/transport/grpc/handlers"
	"eventify/api/internal/transport/grpc/interceptors"
	"eventify/api/internal/transport/grpc/proto"
	"eventify/platform/logger"
	"eventify/platform/postgres"
	"eventify/platform/telemetry"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log := logger.New(true)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.ErrorWithError("load config", err)
		os.Exit(1)
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseDSN)
	if err != nil {
		log.ErrorWithError("connect postgres", err)
		os.Exit(1)
	}
	defer pool.Close()

	jwtProvider, err := auth.NewJWTProvider(cfg.JWTSecret, cfg.JWTExpiryMins, "eventify", "eventify-api")
	if err != nil {
		log.ErrorWithError("configure jwt", err)
		os.Exit(1)
	}

	telemetry.AddTelemetry("eventify-grpc")

	// The auth interceptor is not optional: without it every RPC on this port is
	// unauthenticated, which is how the service shipped before.
	server := grpc.NewServer(grpc.UnaryInterceptor(interceptors.Auth(jwtProvider)))

	proto.RegisterEventServiceServer(server, handlers.NewEventHandler(handlers.Handlers{
		Create: events.NewCreateEventHandler(pool),
		Update: events.NewUpdateEventHandler(pool),
		Get:    events.NewGetEventHandler(pool),
		List:   events.NewGetEventsHandler(pool),
		Delete: events.NewDeleteEventHandler(pool),
	}))
	reflection.Register(server)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.ErrorWithError("listen", err)
		os.Exit(1)
	}

	go func() {
		<-ctx.Done()
		log.Info("shutting down grpc server")
		server.GracefulStop()
		if err := telemetry.ShutdownTracer(); err != nil {
			log.ErrorWithError("shutdown tracer", err)
		}
	}()

	log.Info("grpc server listening on :" + cfg.GRPCPort)
	if err := server.Serve(lis); err != nil {
		log.ErrorWithError("grpc server stopped", err)
		os.Exit(1)
	}
}
