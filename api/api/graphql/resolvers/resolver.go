package resolvers

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"eventify/internal/service"
	"eventify/pkg/logger"
	"eventify/pkg/telemetry"
)

type Resolver struct {
	eventService     service.IEventService
	log              *logger.Logger
	telemetryAdapter telemetry.ITelemetryAdapter
}

// NewResolver creates a new resolver with dependencies
func NewResolver(eventService service.IEventService, log *logger.Logger, telemetryAdapter telemetry.ITelemetryAdapter) *Resolver {
	return &Resolver{
		eventService:     eventService,
		log:              log,
		telemetryAdapter: telemetryAdapter,
	}
}
