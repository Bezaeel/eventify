# Eventify

## What this project is

An event-management system exposed over **three transports** — REST (Fiber), gRPC, and GraphQL (gqlgen) — sharing **one implementation of each use case**. Writes emit domain events through a **transactional outbox**; a relay drains them to RabbitMQ; a **subscriber** projects them into an analytics read model.

The goal, stated once so every decision below follows from it:

> **A use case is written once, tested once, and reachable from every transport. A transport knows how to decode and encode; it knows nothing about SQL. A handler knows SQL; it knows nothing about HTTP.**

## Workspace layout

A Go workspace (`go.work`) of five modules. Bare module paths (`eventify/api`, not `github.com/...`), so every cross-module dependency carries a `replace` directive as well as a `require` — without it a module cannot build outside the workspace (Docker, CI).

```
go.work
platform/     eventify/platform     logger, telemetry, postgres pool, config, apperrors
events/       eventify/events       versioned wire contracts + Envelope + RoutingKey
outbox/       eventify/outbox       outbox table, Enqueue, relay binary
api/          eventify/api          three transports over shared feature slices
subscribers/  eventify/subscribers  one binary, one router, many event handlers
```

Dependency direction, never violated:

```
api ─┐
     ├─► platform          (platform imports nothing from the others)
outbox ─┤
     │
subscribers ─┘

api ──► outbox ──► events ◄── subscribers
```

`platform` must never import `api`. It once did — `postgres.go` imported `internal/shared/config` — which is why it takes a DSN string now.

## The `api` module

```
api/
  cmd/{http-server,grpc-server,graphql-server}/   entry points, wiring only
  internal/
    domain/          plain structs; no gorm tags, no fiber types, no proto types
    features/        THE USE CASES. raw SQL lives here.
      events/update_event.go        UpdateEventCommand + Handler
      events/get_events.go          GetEventsQuery   + Handler
    transport/
      http/v1/events/update_event.go    DTOs + route + mapping
      http/v2/events/update_event.go    different DTOs, same command
      grpc/...                          proto package eventify.events.v1
      graphql/...                       additive schema, @deprecated
    shared/{auth,config,constants}
    migrations/
```

### Feature slices

One file per use case. It owns its command/query struct, its result struct, and its SQL.

```go
type UpdateEventCommand struct { ID uuid.UUID; Name, Location string }
type UpdateEventResult  struct { EventID uuid.UUID }

type UpdateEventHandler struct{ db postgres.Querier }

func (h UpdateEventHandler) Handle(ctx context.Context, cmd UpdateEventCommand) (UpdateEventResult, error) {
	var id uuid.UUID
	err := h.db.QueryRow(ctx,
		`UPDATE events SET name=$1, location=$2, updated_at=now() WHERE id=$3 RETURNING id`,
		cmd.Name, cmd.Location, cmd.ID).Scan(&id)
	return UpdateEventResult{EventID: id}, err
}
```

- **No service layer. No repository layer.** SQL is inline, in the handler.
- Handlers take `postgres.Querier`, satisfied by both `*pgxpool.Pool` and `pgx.Tx`. That is what lets a write and its outbox row commit in one transaction.
- A **private** helper function is warranted only when one use case needs more than one database round trip. It stays in the same file, unexported.
- Handlers return `apperrors.Error`. **Never an HTTP status code** — the handler does not know which transport called it.

### Versioning

Two axes, kept apart.

**Wire contract changes → new transport DTO, same handler.** Renaming a field or reshaping a response is a `transport/http/v2/` concern. The SQL is not touched. This is the entire payoff of transport-agnostic handlers: most version bumps never reach them.

**Behaviour changes → new command, new handler file.**

```
features/events/update_event.go       UpdateEventCommand
features/events/update_event_v2.go    UpdateEventV2Command — different SQL, different rules
```

Never a numbered method on a shared type. The deleted `IEventService.Get2AllEvents` was exactly this: a behaviour fork smeared into the interface every version and every transport shared.

Sunsetting v1 is `rm -rf transport/http/v1/events/`. That only works because the adapter holds nothing but decode/encode.

The three transports version by different mechanisms — HTTP by URL path, gRPC by proto package (`eventify.events.v1`), GraphQL not at all (additive schema + `@deprecated`). A shared handler cannot satisfy three schemes at once, so **version lives at the edge**.

### Events and the outbox

A write that publishes to RabbitMQ has two commit points and no way to make them atomic. So it doesn't:

```go
tx, _ := pool.Begin(ctx)
defer tx.Rollback(ctx)

h := events.UpdateEventHandler{db: tx}
res, err := h.Handle(ctx, cmd)          // business write
outbox.Enqueue(ctx, tx, name, ver, evt) // intent to publish — same tx

tx.Commit(ctx)                           // both, or neither
```

The relay publishes and marks rows published. It may publish then crash before marking, so delivery is **at-least-once**. **Every subscriber must be idempotent on `Envelope.MessageID`.**

Published event structs are contracts with other processes. **Never edit one in place.** Add `events/eventcreated/v2/`, dual-publish, migrate consumers, delete v1.

## Commands

```bash
make check              # vet + staticcheck + tests, all modules
make build              # every binary, stripped (-ldflags="-s -w")
make test-unit          # no Docker
make test-integration   # testcontainers; needs Docker
make migrate-up
make migrate-create MODULE=api NAME=add_event_status
make run-http | run-grpc | run-graphql | run-relay | run-subscriber
```

Each module also builds alone: `cd outbox && go build ./...`.

## Testing

| Code under test | Test type | Why |
|---|---|---|
| Feature handler (`internal/features/`) | **Integration**, testcontainers | It contains raw SQL. Mocks cannot validate a query. |
| Transport adapter | **Unit** | Inject a handler func; assert decode/encode/status mapping. |
| Subscriber handler | **Integration** | Raw SQL. Assert idempotency by dispatching the same MessageID twice. |
| Relay | **Unit** | Inject a fake `relay.Publisher`. |

Mirror the package path under `tests/unit/` or `tests/integration/`. Table-driven, sub-tests named with `t.Run`. `-count=1` to bypass the cache. Integration tests spin up a fresh container per suite.

There are **no service mocks**, because there are no service interfaces. `mockgen` now generates exactly one mock: `ITelemetryAdapter`.

## Go standards

Static analysis is not optional — `go vet ./...` and `staticcheck ./...` before every commit. `staticcheck.conf` enables all checks except `ST1003`.

- `strconv.Itoa` / `FormatInt` / `FormatBool` over `fmt.Sprintf` on hot paths — `Sprintf` reflects.
- `strings.Builder` over `+` in loops. `sync.Pool` for request-scoped buffers.
- Run `fieldalignment -fix ./...` on new or modified structs; order fields largest → smallest.
- Release builds are stripped: `-ldflags="-s -w"` (already in `make build`).

## Do not

- Do not put SQL in a transport adapter. Do not put `fiber.Ctx`, `proto.*`, or an HTTP status in a feature handler.
- Do not add a service or repository layer back. If a use case needs two queries, write an unexported helper in its own file.
- Do not edit a published event struct. Add a version.
- Do not pass a `*pgxpool.Pool` to `outbox.Enqueue`. It takes the caller's `pgx.Tx`, or the pattern is pointless.
- Do not ack a message whose handler returned an error. Nack it; let the DLQ catch it.
- Do not use `fmt.Println` for logging; use `platform/logger`.
- Do not embed secrets in source; read them from the environment.
- Do not commit to `main` directly.
