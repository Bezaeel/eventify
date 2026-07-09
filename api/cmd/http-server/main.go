// Command http-server serves the REST API.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"eventify/api/internal/shared/auth"
	"eventify/api/internal/shared/config"
	transporthttp "eventify/api/internal/transport/http"
	"eventify/platform/logger"
	"eventify/platform/postgres"
	"eventify/platform/telemetry"

	scalar "github.com/oSethoum/fiber-scalar"
)

// @title Eventify API
// @version 1.0
// @description Event management over REST, gRPC and GraphQL, sharing one implementation per use case.

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and a JWT.

// @host localhost:3000
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

	// Fails closed when JWT_SECRET is unset or too short.
	jwtProvider, err := auth.NewJWTProvider(cfg.JWTSecret, cfg.JWTExpiryMins, "eventify", "eventify-api")
	if err != nil {
		log.ErrorWithError("configure jwt", err)
		os.Exit(1)
	}

	telemetry.AddTelemetry("eventify-api")
	adapter := telemetry.NewTelemetryAdapter()

	app := transporthttp.NewApp(pool, jwtProvider, adapter)

	app.Get("/docs", scalar.Handler(&scalar.Options{
		SpecURL:  "docs/swagger.json",
		SpecFile: "docs/swagger.json",
		Layout:   scalar.LayoutClassic,
		Theme:    scalar.ThemeSolarized,
		DarkMode: true,
	}))

	go func() {
		<-ctx.Done()
		log.Info("shutting down http server")
		if err := app.Shutdown(); err != nil {
			log.ErrorWithError("shutdown http server", err)
		}
		if err := telemetry.ShutdownTracer(); err != nil {
			log.ErrorWithError("shutdown tracer", err)
		}
	}()

	log.Info("http server listening on :" + cfg.HTTPPort)
	if err := app.Listen(":" + cfg.HTTPPort); err != nil {
		log.ErrorWithError("http server stopped", err)
		os.Exit(1)
	}
}
