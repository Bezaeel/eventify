package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ITelemetryAdapter interface {
	TrackEvent(ctx context.Context, eventName string, properties map[string]string)
	TrackError(err error, properties map[string]string)
	GenerateOperationID() string
	GetOperationIDFromContext(ctx context.Context) string
	StartRequestSpan(ctx context.Context, operationName string, properties map[string]string) (context.Context, trace.Span, func())
}

type TelemetryAdapter struct {
	// Add fields for telemetry client if needed
	serviceName string
}

func NewTelemetryAdapter() *TelemetryAdapter {
	return &TelemetryAdapter{
		serviceName: _serviceName,
	}
}

// StartRequestSpan starts a new span for a request and returns the context with span, span, and cleanup function
func (ta *TelemetryAdapter) StartRequestSpan(ctx context.Context, operationName string, properties map[string]string) (context.Context, trace.Span, func()) {
	tracer := otel.Tracer(ta.serviceName)

	// Start a new span - this will be the root span for the request
	ctx, span := tracer.Start(ctx, operationName)

	// Add properties as span attributes
	for key, value := range properties {
		span.SetAttributes(attribute.String(key, value))
	}

	// Return context with span, span, and cleanup function
	return ctx, span, func() {
		span.End()
	}
}

// GenerateOperationID creates a unique operation ID for requests
func (ta *TelemetryAdapter) GenerateOperationID() string {
	return fmt.Sprintf("op-%s-%s", time.Now().Format("20060102-150405"), uuid.New().String()[:8])
}

// GetOperationIDFromContext extracts operation ID from context
func (ta *TelemetryAdapter) GetOperationIDFromContext(ctx context.Context) string {
	// First try to get from context
	if val, ok := ctx.Value("operation_id").(string); ok {
		return val
	}

	// If not in context, try to get from trace
	if span := trace.SpanFromContext(ctx); span != nil {
		if traceID := span.SpanContext().TraceID(); traceID.IsValid() {
			return traceID.String()
		}
	}

	// Fallback: generate a new one
	return ta.GenerateOperationID()
}

// getOperationID extracts operation ID from context or generates one from trace
func (ta *TelemetryAdapter) getOperationID(ctx context.Context) string {
	return ta.GetOperationIDFromContext(ctx)
}

// TrackError implements ITelemetryAdapter.
func (ta *TelemetryAdapter) TrackError(err error, properties map[string]string) {
	tracer := otel.Tracer(ta.serviceName)
	_, span := tracer.Start(context.Background(), "Error")
	defer span.End()

	span.RecordError(err)

	// Add properties as span attributes
	for key, value := range properties {
		span.SetAttributes(attribute.String(key, value))
	}
}

// TrackEvent allows tracking events with an existing trace context
func (ta *TelemetryAdapter) TrackEvent(ctx context.Context, eventName string, properties map[string]string) {
	tracer := otel.Tracer(ta.serviceName)

	// Continue the trace from the provided context (this will continue existing trace if present)
	ctx, span := tracer.Start(ctx, eventName)
	defer span.End()

	// Get operation ID from context or trace
	operationID := ta.getOperationID(ctx)
	properties["operation_id"] = operationID

	// Add properties as span attributes
	for key, value := range properties {
		span.SetAttributes(attribute.String(key, value))
	}

	// Debug logging to verify trace linking
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	fmt.Printf("TrackEvent - Event: %s, Operation ID: %s, Trace ID: %s, Span ID: %s\n",
		eventName, operationID, traceID, spanID)
}
