// Command graphql-server serves the GraphQL API.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eventify/api/internal/features/events"
	"eventify/api/internal/shared/auth"
	"eventify/api/internal/shared/config"
	"eventify/api/internal/transport/graphql/generated"
	gqlmiddleware "eventify/api/internal/transport/graphql/middleware"
	"eventify/api/internal/transport/graphql/resolvers"
	"eventify/platform/logger"
	"eventify/platform/postgres"
	"eventify/platform/telemetry"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
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

	telemetry.AddTelemetry("eventify-graphql")

	resolver := resolvers.NewResolver(
		events.NewCreateEventHandler(pool),
		events.NewUpdateEventHandler(pool),
		events.NewGetEventHandler(pool),
		events.NewGetEventsHandler(pool),
		events.NewDeleteEventHandler(pool),
	)

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	router := mux.NewRouter()
	router.Handle("/", playground.Handler("GraphQL playground", "/query"))
	// Auth attaches claims when a valid bearer token is present; mutations
	// require them, queries do not.
	router.Handle("/query", gqlmiddleware.Auth(jwtProvider)(srv))

	// The old server listened on :3001 while the ReadMe documented :8080.
	httpServer := &http.Server{
		Addr:              ":" + cfg.GraphQLPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		log.Info("shutting down graphql server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.ErrorWithError("shutdown graphql server", err)
		}
		if err := telemetry.ShutdownTracer(); err != nil {
			log.ErrorWithError("shutdown tracer", err)
		}
	}()

	log.Info("graphql server listening on :" + cfg.GraphQLPort)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.ErrorWithError("graphql server stopped", err)
		os.Exit(1)
	}
}
