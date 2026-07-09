package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

var _serviceName string = ""

func InitTracer(serviceName string, endpoint string) (*trace.TracerProvider, error) {
	_serviceName = serviceName
	ctx := context.Background()

	fmt.Printf("Initializing tracer for service: %s with endpoint: %s\n", serviceName, endpoint)

	// Use the correct Jaeger OTLP gRPC endpoint
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure())
	if err != nil {
		fmt.Printf("Failed to create OTLP exporter: %v\n", err)
		return nil, err
	}

	fmt.Println("OTLP exporter created successfully")

	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),                          // Enable sampling for all traces
		trace.WithBatcher(exporter, trace.WithBatchTimeout(time.Second)), // Reduce batch timeout to 1 second
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		)),
	)

	fmt.Println("Tracer provider created successfully")

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	fmt.Println("OpenTelemetry tracer configured successfully")
	return tracerProvider, nil
}
