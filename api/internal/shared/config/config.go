// Package config assembles the api module's configuration from the environment.
package config

import (
	"fmt"

	platformconfig "eventify/platform/config"
	"eventify/platform/postgres"
)

// Config is everything the api binaries need in order to start.
type Config struct {
	DatabaseDSN   string
	JWTSecret     string
	AMQPURI       string
	HTTPPort      string
	GRPCPort      string
	GraphQLPort   string
	JWTExpiryMins int
}

// Load reads configuration from the environment.
//
// The previous implementation called viper.ReadInConfig() on a `.env` file
// resolved from "../../", so it only worked when the process was started from
// inside cmd/<server>/ — which is why the Makefile ran `cd cmd/http-server &&
// go run .`. It also returned an error when no .env file existed, making a
// container deployment (env vars, no file) impossible. The environment is the
// source of truth now; a .env file is something the shell or make loads, not
// something the binary hunts for on disk.
func Load() (Config, error) {
	dbPassword, err := platformconfig.MustString("DB_PASSWORD")
	if err != nil {
		return Config{}, err
	}

	// No default, deliberately. A signing key that falls back to a constant is
	// not a signing key. See auth.NewJWTProvider.
	jwtSecret, err := platformconfig.MustString("JWT_SECRET")
	if err != nil {
		return Config{}, fmt.Errorf("JWT_SECRET must be set: %w", err)
	}

	return Config{
		DatabaseDSN: postgres.DSN(
			platformconfig.String("DB_HOST", "localhost"),
			platformconfig.String("DB_PORT", "5432"),
			platformconfig.String("DB_USER", "postgres"),
			dbPassword,
			platformconfig.String("DB_NAME", "eventify"),
			platformconfig.String("DB_SSLMODE", "disable"),
		),
		JWTSecret:     jwtSecret,
		JWTExpiryMins: platformconfig.Int("JWT_EXPIRY_MINUTES", 60),
		AMQPURI:       platformconfig.String("AMQP_URI", "amqp://guest:guest@localhost:5672/"),
		HTTPPort:      platformconfig.String("HTTP_PORT", "3000"),
		GRPCPort:      platformconfig.String("GRPC_PORT", "3002"),
		GraphQLPort:   platformconfig.String("GRAPHQL_PORT", "8080"),
	}, nil
}
