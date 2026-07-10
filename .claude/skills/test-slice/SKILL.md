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
| `outbox/relay`, `outbox/processors` | Integration, with a fake `processors.Publisher` | The publisher is faked; the database is real, because the relay's whole job is `FOR UPDATE SKIP LOCKED` inside a transaction. |
| `outbox.Enqueue`, `FetchQueued`, `Message` transitions | Integration | Holds raw SQL, and `FOR UPDATE SKIP LOCKED` cannot be mocked. |
| Documented operational SQL | Integration | A runbook nobody executes is a guess. See `recovery_test.go`. |
| `platform/**`, `handler.Registry` | Unit | Pure. |

Name integration tests `TestIntegration…`: `make test-integration` selects them with `-run Integration`.

## Does the test have teeth?

A test that cannot fail is documentation, not verification. Before you trust a new integration test, break the thing it guards and watch it go red:

```bash
# remove `ON CONFLICT (message_id) DO NOTHING`, then:
go test -run TestIntegrationEventCreated_IsIdempotentOnMessageID ./tests/...
# expect: FAIL ... duplicate key value violates unique constraint
```

Then restore the code. Do this once per test that guards a specific past bug.

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

Dispatch the same payload twice. This is the test that catches non-idempotent consumers, and nothing else will.

`Handle` takes raw bytes — there is no envelope. The routing key the message arrived on identifies the contract, and the subscriber closes over it per subscription.

```go
func TestIntegrationEventCreated_IsIdempotentOnMessageID(t *testing.T) {
	testsupport.SkipUnlessDocker(t)
	pool := testsupport.Pool(t)
	h := handler.NewEventCreated(pool, logger.New(false))

	messageID := uuid.New()
	body, err := json.Marshal(events.EventCreated{MessageID: messageID, ID: uuid.New()})
	require.NoError(t, err)

	require.NoError(t, h.Handle(ctx, body))
	require.NoError(t, h.Handle(ctx, body))   // redelivery

	var n int
	pool.QueryRow(ctx, `SELECT count(*) FROM analytics_events WHERE message_id=$1`, messageID).Scan(&n)
	require.Equal(t, 1, n)
}
```

## Integration test: the relay

The relay is integration-tested, not unit-tested: it exists to drive `FOR UPDATE SKIP LOCKED` inside a transaction, and that cannot be faked. Fake only the publisher.

```go
type fakePublisher struct {
	sent []published
	err  error
	mu   sync.Mutex
}

func (f *fakePublisher) Publish(_ context.Context, routingKey, messageID string, body []byte) error {
	f.mu.Lock(); defer f.mu.Unlock()
	if f.err != nil { return f.err }
	f.sent = append(f.sent, published{routingKey, messageID, body})
	return nil
}
```

Cover the whole lifecycle, not just the happy path:

- a batch drains and every row lands `COMPLETED`;
- a publish failure leaves rows `QUEUED` and they drain once the broker recovers;
- repeated failure reaches `EXCEEDED` after `MaxAttempts`, and is never claimed again;
- a payload type no processor claims is `POISONED` without spending an attempt;
- `Enqueue` rolls back with the caller's transaction.

## Integration test: documented operational SQL

If a doc tells an operator to run a query, a test runs that query verbatim. `outbox/tests/integration/recovery_test.go` drives a message to `EXCEEDED` and another to `POISONED`, executes the recovery `UPDATE` copied from the `outbox` package doc, and asserts both return to `QUEUED` and then publish.

A runbook that has never been executed is a guess. Renumber a status constant and that test goes red — which is the only reason to trust the runbook at all.

## Conventions

- Table-driven; name sub-tests with `t.Run`.
- `-count=1` bypasses the test cache (already in the Makefile targets).
- Integration tests must not depend on external state — fresh container per suite.
- Mirror the package path under `tests/unit/` or `tests/integration/`.

## Checklist

- [ ] Code holds a SQL string → integration test, not a mock
- [ ] Integration tests guarded by `testing.Short()` / `testsupport.SkipUnlessDocker(t)`
- [ ] Subscriber test dispatches the same `MessageID` twice
- [ ] Relay test covers `COMPLETED`, `QUEUED` retry, `EXCEEDED`, and `POISONED`
- [ ] Operational SQL quoted in a doc is executed verbatim by a test
- [ ] Adapter test asserts status mapping for every `apperrors.Kind` it can return
- [ ] `make check` passes, and each module builds alone with `GOWORK=off go build ./...`
