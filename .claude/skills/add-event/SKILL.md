---
name: add-event
description: Publish a new domain event through the transactional outbox, or evolve an existing event to a new version, and consume it in the subscribers module. Use when a use case needs to emit an event, or when an event payload must change.
---

# Add or evolve a domain event

A published event is a contract with **other processes**. A subscriber running last week's binary will still receive events produced by today's. Everything below follows from that.

## Add a new event

### 1. Declare the contract

`events/<eventname>/v1/<event_name>.go`:

```go
// Package v1 is the first published version of the EventCancelled contract.
// Frozen. To change this shape, add a v2 package.
package v1

const (
	Name    = "EventCancelled"
	Version = "v1"
)

type EventCancelled struct {
	OccurredAt time.Time `json:"occurred_at"`
	ID         string    `json:"id"`
	Reason     string    `json:"reason"`
}
```

Field order largest → smallest (`fieldalignment`). JSON tags are the wire format — treat a tag rename as a breaking change.

### 2. Emit it from the feature handler, inside the transaction

```go
tx, err := h.pool.Begin(ctx)
if err != nil { return res, apperrors.Wrap(apperrors.Internal, "begin", err) }
defer func() { _ = tx.Rollback(ctx) }()

// ...business write against tx...

evt := eventcancelledv1.EventCancelled{ID: res.EventID.String(), Reason: cmd.Reason, OccurredAt: time.Now().UTC()}
if err := outbox.Enqueue(ctx, tx, eventcancelledv1.Name, eventcancelledv1.Version, evt); err != nil {
	return res, err
}
return res, tx.Commit(ctx)
```

**`Enqueue` must receive the `tx`, not the pool.** Passing the pool compiles, and silently reintroduces the dual-write bug the outbox exists to prevent: the row commits independently of the write it accompanies.

The relay picks the row up and publishes it to `eventify.events.EventCancelled.v1`. You write no publishing code.

### 3. Consume it

`subscribers/internal/handler/event_cancelled_v1.go` implementing `handler.Handler`, then register it:

```go
registry, err := handler.NewRegistry(
	handler.NewEventCreatedV1(pool, log),
	handler.NewEventCancelledV1(pool, log),   // add here — not a new main.go
)
```

**The handler must be idempotent.** The relay is at-least-once: it can publish a row and crash before marking it published, redelivering on restart. Deduplicate on `Envelope.MessageID`:

```sql
INSERT INTO analytics_events (message_id, ...) VALUES ($1, ...)
ON CONFLICT (message_id) DO NOTHING
```

An event handler that is not idempotent will double-count in production, and no test that dispatches each message once will catch it.

## Evolve an existing event

**Never edit a published struct.** Adding a field to `v1` breaks every consumer that validates payloads, and the compiler cannot warn you — the break happens over the wire, at runtime, in another process.

1. Create `events/<eventname>/v2/` with the new shape and `Version = "v2"`.
2. **Dual-publish**: enqueue both v1 and v2 from the handler.
3. Add a v2 subscriber handler; register both.
4. Wait until every consumer is confirmed on v2 (check the DLQ and v1 volume).
5. Delete the v1 enqueue, then the v1 handler, then the v1 package.

Steps 2–4 are not optional. Skipping to step 5 drops events on the floor for any consumer still on the old binary.

## Delivery semantics

| Guarantee | Reality |
|---|---|
| Write + enqueue | Atomic. One transaction. |
| Enqueue → publish | At-least-once. Relay may republish after a crash. |
| Publish → handle | At-least-once. Nacked messages redeliver. |
| Ordering | Per routing key, best-effort. **Do not depend on it.** |

## Checklist

- [ ] New event lives in its own `vN` package with `Name` + `Version` constants
- [ ] `outbox.Enqueue` receives a `pgx.Tx`, never a `*pgxpool.Pool`
- [ ] Subscriber handler deduplicates on `MessageID`
- [ ] Handler registered in `NewRegistry`, not in a new binary
- [ ] Evolving an event? v2 added, dual-publishing, v1 not modified
- [ ] Integration test dispatches the same MessageID twice and asserts one row
