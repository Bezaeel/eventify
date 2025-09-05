package middlewares

import (
	"context"
	"eventify/pkg/telemetry"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// TelemetryMiddleware creates a middleware that automatically injects operation IDs
func TelemetryMiddleware(telemetryAdapter telemetry.ITelemetryAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generate a unique operation ID for this request
		operationID := telemetryAdapter.GenerateOperationID()

		// Create a new context with the operation ID
		ctx := context.WithValue(c.Context(), "operation_id", operationID)

		// Start a root span for this request
		ctx, span, cleanup := telemetryAdapter.StartRequestSpan(ctx, "HTTP Request", map[string]string{
			"http.method":     c.Method(),
			"http.path":       c.Path(),
			"http.user_agent": c.Get("User-Agent"),
			"http.ip":         c.IP(),
			"operation_id":    operationID,
		})

		// Debug logging for root span
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()
		fmt.Printf("MIDDLEWARE - Root span created - Method: %s, Path: %s, Operation ID: %s, Trace ID: %s, Span ID: %s\n",
			c.Method(), c.Path(), operationID, traceID, spanID)

		// Store the span context in Fiber context for later use
		c.Context().SetUserValue("trace_context", ctx)
		c.Context().SetUserValue("operation_id", operationID)

		// Track the request start
		telemetryAdapter.TrackEvent(ctx, "RequestStarted", map[string]string{
			"method":     c.Method(),
			"path":       c.Path(),
			"user_agent": c.Get("User-Agent"),
			"ip":         c.IP(),
		})

		// Continue to the next middleware/handler
		err := c.Next()

		// End the root span
		cleanup()
		fmt.Printf("MIDDLEWARE - Root span ended - Method: %s, Path: %s, Trace ID: %s, Span ID: %s\n",
			c.Method(), c.Path(), traceID, spanID)

		return err
	}
}
