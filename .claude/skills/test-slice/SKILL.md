---
name: test-slice
description: Choose and write the right test for eventify code — integration tests with testcontainers for anything holding raw SQL, unit tests for transport adapters and the relay. Use when adding tests, when a change breaks tests, or when deciding whether something needs Docker.
---

# Test a slice

## Which test to write

The rule is mechanical: **does this code contain a SQL string?**

| Code | Test | Rationale |
|---|---|---|
| `api/internal/features/**` | Integration, testcontainers | Holds raw SQL. A mock cannot tell you the query is valid, hits an index, or matches the schema. |
| `api/internal/transport/**` | Unit | Decode/encode/status mapping. Inject a handler func. |
| `subscribers/internal/handler/**` | Integration | Holds raw SQL. Also assert idempotency. |
| `outbox/relay` | Unit | Inject a fake `relay.Publisher`. |
| `outbox.Enqueue`, `FetchUnpublished` | Integration | Holds raw SQL, and `FOR UPDATE SKIP LOCKED` cannot be mocked. |
| `platform/**` | Unit | Pure. |

**There are no service mocks.** There are no service interfaces to mock — that layer was deliberately removed. `mockgen` generates exactly one mock, for `ITelemetryAdapter`. If you find yourself wanting to mock a feature handler in order to test another feature handler, the second one is doing too much.

## Integration test: a feature handler

Mirror the package path under `tests/integration/`. One container per suite, torn down after.

```go
func TestIntegration_UpdateEventHandler(t *testing.T) {
	if testing.Short() { t.Skip("requires docker") }

	pool := setupPostgres(t)   // testcontainers + migrations, per suite
	h := events.NewUpdateEventHandler(pool)

	tests := []struct {
		name    string
		seed    domain.Event
		cmd     events.UpdateEventCommand
		wantErr apperrors.Kind
	}{
		{name: "updates an existing event", ...},
		{name: "missing event is NotFound", ..., wantErr: apperrors.NotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ...
			_, err := h.Handle(context.Background(), tt.cmd)
			if tt.wantErr != 0 {
				require.Equal(t, tt.wantErr, apperrors.KindOf(err))
				return
			}
			require.NoError(t, err)
		})
	}
}
```

Guard with `testing.Short()` so `make test-unit` (which passes `-short`) stays Docker-free.

## Unit test: a transport adapter

The adapter's whole job is decode → call → encode. Test exactly that, with a stub in place of the handler.

```go
func TestUpdateEvent_MapsNotFoundTo404(t *testing.T) {
	c := v1.NewController(stubHandler{
		err: apperrors.New(apperrors.NotFound, "nope"),
	})
	app := fiber.New()
	app.Put("/api/v1/events/:id", c.UpdateEvent)

	resp, _ := app.Test(httptest.NewRequest(http.MethodPut, "/api/v1/events/"+uuid.NewString(), body))

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}
```

Assert the **mapping**, not the SQL. Cover: malformed path param → 400; malformed body → 400; each `apperrors.Kind` → its status; success → correct JSON shape.

## Integration test: a subscriber handler

Dispatch the same envelope twice. This is the test that catches non-idempotent consumers, and nothing else will.

```go
func TestIntegration_EventCreatedV1_Idempotent(t *testing.T) {
	if testing.Short() { t.Skip("requires docker") }
	pool := setupPostgres(t)
	h := handler.NewEventCreatedV1(pool, logger.New(false))

	env := envelopeFor(t, eventcreatedv1.EventCreated{ID: uuid.NewString()})

	require.NoError(t, h.Handle(ctx, env))
	require.NoError(t, h.Handle(ctx, env))   // redelivery

	var n int
	pool.QueryRow(ctx, `SELECT count(*) FROM analytics_events WHERE message_id=$1`, env.MessageID).Scan(&n)
	require.Equal(t, 1, n)
}
```

## Unit test: the relay

```go
type fakePublisher struct {
	sent []events.Envelope
	err  error
}
func (f *fakePublisher) Publish(_ context.Context, _ string, e events.Envelope) error {
	if f.err != nil { return f.err }
	f.sent = append(f.sent, e); return nil
}
```

Cover: a publish failure mid-batch commits the rows already published and leaves the rest unpublished for retry.

## Conventions

- Table-driven; name sub-tests with `t.Run`.
- `-count=1` bypasses the test cache (already in the Makefile targets).
- Integration tests must not depend on external state — fresh container per suite.
- Mirror the package path under `tests/unit/` or `tests/integration/`.

## Checklist

- [ ] Code holds a SQL string → integration test, not a mock
- [ ] Integration tests guarded by `testing.Short()`
- [ ] Subscriber test dispatches the same `MessageID` twice
- [ ] Adapter test asserts status mapping for every `apperrors.Kind` it can return
- [ ] `make check` passes
