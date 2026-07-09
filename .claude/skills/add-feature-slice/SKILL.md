---
name: add-feature-slice
description: Add a new use case to the api module as a vertical slice — a command/query handler with raw SQL, plus thin adapters on each transport that needs it. Use when adding any new endpoint, RPC, or GraphQL field that reads or writes the database.
---

# Add a feature slice

A use case is written once in `api/internal/features/` and exposed by as many transports as need it. The handler never learns which one called it.

## 1. Write the handler

`api/internal/features/<area>/<use_case>.go`, one file per use case.

```go
package events

import (
	"context"
	"errors"

	"eventify/platform/apperrors"
	"eventify/platform/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CancelEventCommand is the input. Fields are already-parsed Go types —
// parsing is the transport's job, not the handler's.
type CancelEventCommand struct {
	EventID uuid.UUID
	Reason  string
}

type CancelEventResult struct {
	EventID    uuid.UUID
	CancelledAt time.Time
}

// CancelEventHandler takes a Querier, not a *pgxpool.Pool, so a caller can
// enrol it in a transaction alongside outbox.Enqueue.
type CancelEventHandler struct {
	db postgres.Querier
}

func NewCancelEventHandler(db postgres.Querier) CancelEventHandler {
	return CancelEventHandler{db: db}
}

func (h CancelEventHandler) Handle(ctx context.Context, cmd CancelEventCommand) (CancelEventResult, error) {
	var res CancelEventResult
	err := h.db.QueryRow(ctx,
		`UPDATE events
		    SET cancelled_at = now(), cancellation_reason = $2
		  WHERE id = $1 AND cancelled_at IS NULL
		  RETURNING id, cancelled_at`,
		cmd.EventID, cmd.Reason,
	).Scan(&res.EventID, &res.CancelledAt)

	if errors.Is(err, pgx.ErrNoRows) {
		// Semantic error. NOT http.StatusNotFound — the handler does not know
		// it is being called over HTTP.
		return res, apperrors.New(apperrors.NotFound, "event not found or already cancelled")
	}
	if err != nil {
		return res, apperrors.Wrap(apperrors.Internal, "cancel event", err)
	}
	return res, nil
}
```

Rules:

- **Raw SQL, inline.** No repository, no service.
- **Positional placeholders** (`$1`), pgx does not accept `?`.
- Return `apperrors.Error` with a `Kind`. Never a status code, never `fiber.Error`.
- **A private helper only when the use case needs more than one round trip.** Keep it unexported, in the same file. Two round trips usually means you want a transaction — take one and pass the `pgx.Tx` down as the `Querier`.
- No `fiber.Ctx`, no `proto.*`, no `graphql` types anywhere in this file.

## 2. If the use case emits an event

Take a transaction and enqueue in it. This is the only way the write and the publish are atomic.

```go
func (h CancelEventHandler) Handle(ctx context.Context, cmd CancelEventCommand) (CancelEventResult, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil { return res, apperrors.Wrap(apperrors.Internal, "begin", err) }
	defer func() { _ = tx.Rollback(ctx) }()

	// ... SQL against tx ...

	evt := eventcancelledv1.EventCancelled{ID: res.EventID.String(), ...}
	if err := outbox.Enqueue(ctx, tx, eventcancelledv1.Name, eventcancelledv1.Version, evt); err != nil {
		return res, err
	}
	return res, tx.Commit(ctx)
}
```

Passing the pool instead of `tx` compiles and silently defeats the pattern.

## 3. Add the transport adapters

Only for the transports that should expose it. An adapter decodes, calls `Handle`, and encodes. It contains no business rules and no SQL.

`api/internal/transport/http/v1/events/cancel_event.go`:

```go
type cancelEventRequest struct {
	Reason string `json:"reason"`
}

func (c *Controller) CancelEvent(ctx *fiber.Ctx) error {
	id, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, "invalid event id"))
	}
	var req cancelEventRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httperr.Write(ctx, apperrors.New(apperrors.Invalid, err.Error()))
	}

	res, err := c.cancelEvent.Handle(ctx.Context(), events.CancelEventCommand{
		EventID: id, Reason: req.Reason,
	})
	if err != nil {
		return httperr.Write(ctx, err) // maps apperrors.Kind -> status
	}
	return ctx.Status(fiber.StatusOK).JSON(cancelEventResponse{EventID: res.EventID})
}
```

gRPC and GraphQL adapters call the *same* `Handle` and map `apperrors.KindOf(err)` to `codes.Code` / a GraphQL error.

## 4. Wire it up

Construct the handler once in `api/cmd/<server>/main.go` and pass it to the controller. No DI container.

## 5. Test it

- Handler → **integration** test (testcontainers), because it holds SQL.
- Adapter → **unit** test with a stubbed handler func.

See the `test-slice` skill.

## Checklist

- [ ] Handler has no transport types and returns `apperrors`
- [ ] Handler takes `postgres.Querier`
- [ ] Emitting a domain event? `outbox.Enqueue` inside the same `pgx.Tx`
- [ ] Adapter contains zero SQL
- [ ] `go vet ./... && staticcheck ./...` clean
- [ ] `fieldalignment -fix ./...` run on new structs
