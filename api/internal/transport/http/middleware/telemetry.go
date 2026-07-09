package middleware

import (
	"context"

	"eventify/platform/telemetry"

	"github.com/gofiber/fiber/v2"
)

// ctxKey is an unexported type for context keys defined in this package.
//
// The old middleware called context.WithValue(ctx, "operation_id", …) with a
// bare string. `go vet` flags that: any other package storing the same string
// silently collides. A private key type makes collision impossible.
type ctxKey int

const operationIDKey ctxKey = iota

// OperationID returns the operation id attached by Telemetry, if any.
func OperationID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(operationIDKey).(string)
	return id, ok
}

// Telemetry starts a root span for each request and attaches an operation id.
func Telemetry(adapter telemetry.ITelemetryAdapter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		operationID := adapter.GenerateOperationID()
		ctx := context.WithValue(c.Context(), operationIDKey, operationID)

		ctx, _, end := adapter.StartRequestSpan(ctx, "HTTP "+c.Method()+" "+c.Path(), map[string]string{
			"http.method":     c.Method(),
			"http.path":       c.Path(),
			"http.user_agent": c.Get(fiber.HeaderUserAgent),
			"http.ip":         c.IP(),
			"operation_id":    operationID,
		})
		// The old middleware fmt.Printf'd the trace and span ids on every
		// request, on the way in and again on the way out. That is two
		// unstructured lines per request straight to stdout.
		defer end()

		c.SetUserContext(ctx)
		return c.Next()
	}
}
