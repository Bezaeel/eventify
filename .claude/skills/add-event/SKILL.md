---
name: add-event
description: Publish a new domain event through the transactional outbox, process it in the relay, and consume it in the subscribers module. Use when a use case needs to emit an event, or when an event payload must change.
---

# Add or change a domain event

A published event is a contract with **other processes**. Everything below follows from that.

Events are **not versioned in their type**. The pipeline is versioned: producer and consumer deploy together. See *Change an existing event* for the one edge that does not cover.

## Add a new event

### 1. Declare the contract

`events/<event_name>.go`, package `events`:

```go
package events

// EventCancelledName identifies the contract on the wire and in
// outbox_messages.payload_type. It lives next to the struct it names so the
// producer and the relay's processor compile against one declaration.
const EventCancelledName = "EventCancelled"

// MessageID is the deduplication key; delivery is at-least-once.
type EventCancelled struct {
	OccurredAt time.Time `json:"occurred_at"`
	MessageID  uuid.UUID `json:"message_id"`
	ID         uuid.UUID `json:"id"`
	Reason     string    `json:"reason"`
}
```

The name constant is not decoration. Without it the producer and the relay each carry their own string literal; a typo in one enqueues rows no processor claims, they poison, and no compiler objects.

Field order largest → smallest (`fieldalignment`). JSON tags are the wire format.

### 2. Emit it from the feature handler, inside the transaction

```go
tx, err := h.pool.Begin(ctx)
if err != nil { return res, apperrors.Wrap(apperrors.Internal, "begin", err) }
defer func() { _ = tx.Rollback(ctx) }()

// ...business write against tx...

messageID := uuid.New()
evt := events.EventCancelled{
	MessageID:  messageID,
	ID:         res.EventID,
	Reason:     cmd.Reason,
	OccurredAt: time.Now().UTC(),
}
if err := outbox.Enqueue(ctx, tx, events.EventCancelledName, messageID, evt); err != nil {
	return res, err
}
return res, tx.Commit(ctx)
```

Two invariants live in that snippet.

**`Enqueue` must receive the `tx`, not the pool.** Passing the pool compiles, and silently reintroduces the dual-write bug the outbox exists to prevent: the row commits independently of the write it accompanies.

**One message ID, minted by the caller.** It is stamped into the payload *and* passed to `Enqueue`. The consumer deduplicates on the copy in the payload; an operator finds the row by the copy on the row. If those disagree, a duplicate delivery cannot be traced to the row that caused it.

### 3. Register a processor in the relay

The relay publishes nothing it has no processor for — an unclaimed payload type is **poisoned** on its first poll. Add a line to `outbox/cmd/relay/main.go`:

```go
procs := []processors.IOutboxProcessor{
	processors.NewGeneric(pub, events.EventCreatedName),
	processors.NewGeneric(pub, events.EventCancelledName),   // add here
}
```

`Generic` publishes the stored bytes unchanged, which is what almost every event wants. If the event must be enriched, or a service called, before it is safe to publish, write a typed processor instead:

```go
type CancelProcessor struct {
	processors.Base[events.EventCancelled]
}

func NewCancelProcessor(pub processors.Publisher) *CancelProcessor {
	return &CancelProcessor{processors.Base[events.EventCancelled]{
		Pub: pub, PayloadType: events.EventCancelledName,
	}}
}

func (p *CancelProcessor) ProcessAsync(ctx context.Context, m *outbox.Message) error {
	return p.Process(ctx, m, func(ctx context.Context, id uuid.UUID, evt events.EventCancelled) error {
		// ...work that must happen before publishing...
		return p.Publish(ctx, m)
	})
}
```

Dispatch is on the declared `PayloadType`. **Never `reflect.TypeOf(payload).String()`.** The row is written by one binary and read by another, possibly after a refactor renamed the struct; a reflected name would stop matching rows already queued and poison the backlog.

The relay publishes to `events.RoutingKey(payloadType)` — `eventify.events.EventCancelled`. You write no publishing code.

### 4. Consume it

`subscribers/internal/handler/event_cancelled.go` implementing `handler.Handler` — `Name() string` and `Handle(ctx, payload []byte) error`. There is no envelope: the routing key the message arrived on identifies the contract, and `Handle` receives the raw payload bytes.

Register it:

```go
registry, err := handler.NewRegistry(
	handler.NewEventCreated(pool, log),
	handler.NewEventCancelled(pool, log),   // add here — not a new main.go
)
```

**The handler must be idempotent.** The relay is at-least-once: it can publish a row and crash before recording that it did, redelivering on restart. Deduplicate on the payload's `MessageID`:

```sql
INSERT INTO analytics_events (message_id, ...) VALUES ($1, ...)
ON CONFLICT (message_id) DO NOTHING
```

Read `MessageID` from the payload, not from broker metadata — only the payload survives a replay from a dump or a hop through a bridge. Reject a payload whose `MessageID` is `uuid.Nil`: it would insert under the nil key and silently swallow the next such event as a duplicate.

An event handler that is not idempotent will double-count in production, and no test that dispatches each message once will catch it.

## Change an existing event

**Additive changes only.** Add a field, and an old message simply leaves it zero.

Never rename a field, change its type, or remove one. During a rollout, messages published by the old producer are still sitting in RabbitMQ when the new consumer starts reading; the new struct decodes them. A rename silently zeroes that field on every message already queued, and nothing fails loudly. The pipeline cannot enforce this for you.

If a change cannot be made additively, it is a new event with a new name — not a new version of this one.

## Delivery semantics

| Guarantee | Reality |
|---|---|
| Write + enqueue | Atomic. One transaction. |
| Enqueue → publish | At-least-once. Relay may republish after a crash. |
| Publish → handle | At-least-once. Nacked messages redeliver. |
| Unclaimed payload type | **Poisoned**, not published. Register a processor. |
| Repeated failure | **Exceeded** after `MaxAttempts` polls. Needs manual reset. |
| Ordering | Per routing key, best-effort. **Do not depend on it.** |

A stalled outbox is cleared by hand, once the cause is fixed:

```sql
UPDATE outbox_messages SET status = 1, attempts = 0 WHERE status IN (2, 4);
```

## Checklist

- [ ] Contract in `events/`, flat package, with a `<Name>Name` constant beside it
- [ ] No version in the type; any change to an existing struct is additive
- [ ] `outbox.Enqueue` receives a `pgx.Tx`, never a `*pgxpool.Pool`
- [ ] One message ID: minted by the caller, stamped in the payload, passed to `Enqueue`
- [ ] Processor registered in `outbox/cmd/relay/main.go`, or the event poisons
- [ ] Processor dispatches on the declared payload type, not `reflect.TypeOf`
- [ ] Subscriber handler deduplicates on the payload's `MessageID` and rejects `uuid.Nil`
- [ ] Handler registered in `NewRegistry`, not in a new binary
- [ ] Integration test dispatches the same MessageID twice and asserts one row
- [ ] `CLAUDE.md` and the skills still describe reality — see `sync-context`
