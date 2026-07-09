# Telemetry and Operation ID Management

## Overview

The Eventify API automatically manages operation IDs and trace context for all requests through middleware, ensuring consistent tracing across all endpoints without requiring manual setup. All spans from the same request are properly linked together in a trace hierarchy.

## How It Works

### Automatic Trace Context Management

1. **Root Span Creation**: The `TelemetryMiddleware` creates a root span for each HTTP request
2. **Context Propagation**: The trace context flows through all subsequent calls
3. **Span Hierarchy**: All spans within a request are children of the root span
4. **Consistent Tracing**: All telemetry calls within the same request share the same trace ID

### Operation ID Format

Operation IDs follow the format: `op-YYYYMMDD-HHMMSS-XXXXXXXX`
- `op-`: Prefix indicating operation
- `YYYYMMDD-HHMMSS`: Timestamp when the request started
- `XXXXXXXX`: 8-character unique identifier

## Trace Hierarchy

```
HTTP Request (Root Span)
├── RequestStarted (Event)
├── V1EventIndexCalled (Event)
├── GetAllEvents (Operation Span)
│   └── Database Query (if tracked)
└── RequestEnded (implicit)
```

## Usage

### In Controllers

```go
func (ec *v1EventController) index(c *fiber.Ctx) error {
    // Track the event with telemetry adapter - trace context is handled internally
    telemetryProperties := map[string]string{
        "method": "index",
    }
    ec.telemetryAdapter.TrackEventFromFiber(c, "V1EventIndexCalled", telemetryProperties)
    
    // Call the service with proper trace context using framework-agnostic method
    traceCtx := ec.telemetryAdapter.GetTraceContextFromFiber(c)
    _ = ec.telemetryAdapter.CallServiceWithTrace(traceCtx, func(ctx context.Context) interface{} {
        return ec.service.GetAllEvents(ctx)
    })
    
    return c.SendString("v1")
}
```

### Accessing Operation ID and Trace Context

```go
// Get operation ID from Fiber context
operationID := telemetryAdapter.GetOperationIDFromFiberContext(c)

// Get trace context from Fiber context (for proper span linking)
traceCtx := telemetryAdapter.GetTraceContextFromFiber(c)

// Call service with trace context (framework-agnostic)
result := telemetryAdapter.CallServiceWithTrace(traceCtx, func(ctx context.Context) interface{} {
    return service.SomeMethod(ctx)
})

// Get operation ID from regular context
operationID := telemetryAdapter.GetOperationIDFromContext(ctx)
```

### In Services

```go
func (e *EventService) GetAllEvents(ctx context.Context) []domain.Event {
    // Automatically creates a child span of the root request span
    defer e.telemetryAdapter.TrackOperation(ctx, "GetAllEvents", map[string]string{
        "operation": "fetch_all_events",
        "service":   "EventService",
    })()
    
    // Database operations inherit the same trace context
    var events []domain.Event
    if err := e.db.WithContext(ctx).Find(&events).Error; err != nil {
        e.telemetryAdapter.TrackError(err, map[string]string{
            "operation": "fetch_all_events",
            "service":   "EventService",
        })
        return nil
    }
    
    return events
}
```

## Benefits

1. **Proper Span Linking**: All spans from the same request are properly linked in a trace hierarchy
2. **No Manual Setup**: No need to manually create or pass trace contexts in each endpoint
3. **Consistent Tracing**: All telemetry within a request shares the same trace ID and operation ID
4. **Distributed Tracing**: Proper trace context flows from controller to service to database
5. **Debugging**: Easy to correlate logs and telemetry across the entire request lifecycle
6. **Observability**: Complete request flow can be visualized in tracing tools

## Middleware Setup

The telemetry middleware is automatically applied globally in `cmd/http_server.go`:

```go
func NewAPIServer(telemetryAdapter telemetry.ITelemetryAdapter) *httpServer {
    app := fiber.New()
    
    app.Use(recover.New())
    app.Use(cors.New())
    app.Use(middlewares.TelemetryMiddleware(telemetryAdapter)) // Automatic trace context injection
    
    return &httpServer{app: app}
}
```

## Example Request Flow

1. **Request arrives**: `/api/v1/events`
2. **Middleware creates root span**: `HTTP Request` with trace ID `abc123`
3. **Middleware generates**: `op-20241201-143022-a1b2c3d4`
4. **Controller tracks**: Event with trace ID `abc123` and operation ID `op-20241201-143022-a1b2c3d4`
5. **Service tracks**: Operation with trace ID `abc123` and operation ID `op-20241201-143022-a1b2c3d4`
6. **Database tracks**: Query with trace ID `abc123` and operation ID `op-20241201-143022-a1b2c3d4`

All spans for this request will be linked together in the trace hierarchy, making it easy to visualize the complete request flow in tracing tools like Jaeger or Zipkin.

## Key Methods

- `TrackEventFromFiber(c *fiber.Ctx, name, props)`: Track an event using Fiber context (handles trace extraction internally)
- `CallServiceWithTrace(ctx context.Context, serviceCall)`: Call service with proper trace context (framework-agnostic)
- `GetTraceContextFromFiber(c *fiber.Ctx)`: Get trace context from Fiber context
- `GetOperationIDFromFiberContext(c *fiber.Ctx)`: Get operation ID from Fiber context
- `TrackEvent(ctx, name, props)`: Track an event with proper span linking
- `TrackOperation(ctx, name, props)`: Track an operation with proper span linking
- `StartRequestSpan(ctx, name, props)`: Start a new root span for requests 