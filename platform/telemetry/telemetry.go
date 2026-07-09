package telemetry

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/sdk/trace"
)

var (
	tracerProvider *trace.TracerProvider
	once           sync.Once
)

func AddTelemetry(serviceName string) {
	// Initialize OpenTelemetry only once
	once.Do(func() {
		var err error
		tracerProvider, err = InitTracer(serviceName, "localhost:4318")
		if err != nil {
			panic(err)
		}
		fmt.Println("Telemetry initialized for service:", serviceName)
	})
}

// GetTracerProvider returns the initialized tracer provider
func GetTracerProvider() *trace.TracerProvider {
	return tracerProvider
}

// ShutdownTracer should be called when the application is shutting down
func ShutdownTracer() error {
	if tracerProvider != nil {
		return tracerProvider.Shutdown(context.Background())
	}
	return nil
}
