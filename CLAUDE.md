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
events/       eventify/events       wire contracts + name constants + RoutingKey
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

messageID := uuid.New()
evt := events.EventCreated{MessageID: messageID, ID: res.EventID, /* ... */}

h := events.UpdateEventHandler{db: tx}
res, err := h.Handle(ctx, cmd)                                    // business write
outbox.Enqueue(ctx, tx, events.EventCreatedName, messageID, evt)  // intent to publish — same tx

tx.Commit(ctx)                                                    // both, or neither
```

The caller mints `messageID`, stamps it into the payload, and hands the same value to `Enqueue`. One identity: the consumer deduplicates on what it reads out of the payload, and an operator finds the row by the same key.

The relay claims queued rows, hands each to the **processor** that declares its payload type, and records the outcome. It may publish then crash before recording, so delivery is **at-least-once**. **Every subscriber must be idempotent on the payload's `MessageID`.**

There is no envelope. The payload publishes bare, under routing key `events.RoutingKey(payloadType)`; the routing key a message arrived on is what identifies its contract.

#### Processors

`outbox/processors` holds two kinds, both satisfying `IOutboxProcessor`, so the relay's loop cannot tell them apart:

- **`Generic`** publishes the stored bytes unchanged. Most events want this. No type parameter — it never looks inside the payload.
- **`Base[T]`** decodes the payload into `T` and hands it to a process function, for an event that must enrich it, call a service, or write a second row before it is safe to publish.

Dispatch is on the **declared** `PayloadType` (`events.EventCreatedName`), never on `reflect.TypeOf(payload).String()`. A Go type name is not a stable identifier: the row is written by one binary and read by another, possibly after a refactor renamed the struct. Rows already queued would stop matching and the backlog would be poisoned.

#### Message lifecycle

`QUEUED → COMPLETED`, or `→ POISONED` (no processor claims the payload type), or `→ EXCEEDED` (failed `MaxAttempts` times).

Attempts are spent one per poll with no backoff, so a broker outage lasting `MaxAttempts` poll intervals exceeds the backlog. **This is deliberate.** The outbox stopping is a thing to alert on, not to retry forever. Both terminal states are cleared by hand once the cause is fixed:

```sql
UPDATE outbox_messages SET status = 1, attempts = 0 WHERE status IN (2, 4);  -- POISONED, EXCEEDED
```

`outbox/tests/integration/recovery_test.go` runs that query verbatim. If you change the status constants, it fails — which is the point.

#### Versioning

**Events are not versioned in their type. The pipeline is versioned.** Producer and consumer deploy together.

That trade has one edge the pipeline cannot cover: during a rollout, messages published by the old producer are still in RabbitMQ when the new consumer starts reading. So **field changes must be additive**. Adding a field is safe — an old message leaves it zero. Renaming one, changing its type, or removing one silently zeroes it on every message already queued, and nothing fails loudly.

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
| Relay + processors | **Integration** | It drives `FOR UPDATE SKIP LOCKED` and real transactions. Inject a fake `processors.Publisher`, but keep the real database. |
| `platform/apperrors`, registries | **Unit** | Pure. |

Transport `Handlers` structs hold **function values**, not concrete handler types, precisely so a unit test can inject a stub:

```go
v1events.Handlers{ Update: func(ctx, cmd) (Result, error) { return Result{}, apperrors.New(apperrors.NotFound, "") } }
```

Production wiring passes method values (`update.Handle`), and `transport/http.NewApp` hands the *same* value to v1 and v2.

Mirror the package path under `tests/unit/` or `tests/integration/`. Table-driven, sub-tests named with `t.Run`. `-count=1` to bypass the cache. Integration tests spin up a fresh container per suite and guard with `testsupport.SkipUnlessDocker(t)`, so `make test-unit` (which passes `-short`) needs no Docker.

Name integration tests `TestIntegration…` — `make test-integration` selects them with `-run Integration`.

There are **no service mocks**, because there are no service interfaces. `mockgen` now generates exactly one mock: `ITelemetryAdapter`.

## Go standards

Static analysis is not optional — `go vet ./...` and `staticcheck ./...` before every commit. `staticcheck.conf` enables all checks except `ST1003`.

**Build every module standalone, not just the workspace.** `go.work` resolves a missing `require` from a sibling module, so `go build eventify/...` stays green while a module is unbuildable in Docker and CI. Only `cd <module> && GOWORK=off go build ./...` catches it.

- `strconv.Itoa` / `FormatInt` / `FormatBool` over `fmt.Sprintf` on hot paths — `Sprintf` reflects.
- `strings.Builder` over `+` in loops. `sync.Pool` for request-scoped buffers.
- Run `fieldalignment -fix ./...` on new or modified structs; order fields largest → smallest.
- Release builds are stripped: `-ldflags="-s -w"` (already in `make build`).

## Do not

- Do not put SQL in a transport adapter. Do not put `fiber.Ctx`, `proto.*`, or an HTTP status in a feature handler.
- Do not add a service or repository layer back. If a use case needs two queries, write an unexported helper in its own file.
- Do not rename, retype, or remove a field on a published event struct. Additive changes only — messages from the old producer are still in flight during a rollout.
- Do not dispatch on `reflect.TypeOf(payload).String()`. The payload type is a declared constant; a reflected name is a wire contract no compiler checks.
- Do not pass a `*pgxpool.Pool` to `outbox.Enqueue`. It takes the caller's `pgx.Tx`, or the pattern is pointless.
- Do not mint a second message ID inside `Enqueue`. The caller stamps one into the payload and passes the same value.
- Do not ack a message whose handler returned an error. Nack it; let the DLQ catch it.
- Do not use `fmt.Println` for logging; use `platform/logger`.
- Do not embed secrets in source; read them from the environment.
- Do not commit to `main` directly.
- Do not leave this file describing code that no longer exists. See **Keeping this file true**.

## Keeping this file true

This file, `.claude/agents/*.md`, and `.claude/skills/*/SKILL.md` are read *before* the code is. When they describe a design that no longer exists, they do not merely fail to help — they actively mislead, and an agent will implement the design it read rather than the one that is there.

This has already happened once. `events/` lost its versioned subpackages, its envelope type, and its two-argument routing-key helper; for the length of a refactor, every skill still instructed agents to create versioned event packages that the module no longer supported.

**So this loop is mandatory, not advisory.**

### The trigger

A change under any of these paths is a **context-affecting change**:

| Path | What it can invalidate |
|---|---|
| `events/**` | contract shape, versioning policy, routing keys |
| `outbox/**` | enqueue signature, processors, message lifecycle, recovery SQL |
| `subscribers/internal/handler/**` | handler interface, registry, idempotency key |
| `platform/amqp/**`, `platform/postgres/**` | topology, publisher, `Querier` |
| `go.work`, any `go.mod` | module graph, dependency direction |
| `**/migrations/*.sql` | any schema this file documents |

### The loop

After a context-affecting change, **before reporting the work complete**:

1. **Detect.** For each doc file, grep it for identifiers you changed — removed types, renamed functions, altered signatures, dropped columns. A doc naming a symbol that no longer compiles is stale, full stop.
2. **Reconcile.** Update the doc to describe what is now true. Do not paper over it: if a design was replaced, say what replaced it. Delete the guidance that no longer applies rather than leaving it beside its successor — two descriptions of one thing is worse than one wrong description, because now nobody knows which is current.
3. **Verify.** Every code block in a doc must compile against the current tree, and every SQL snippet must run against the current schema. If you cannot verify it, do not write it. Snippets that guard a real procedure belong in a test — `outbox/tests/integration/recovery_test.go` is the pattern.
4. **Report.** Say which docs you updated and why. A silent doc edit is as bad as a silent behaviour change.

The `/sync-context` skill performs steps 1–3 across all of `.claude/` and this file.

**A task that changed a contract and left the docs describing the old one is not complete.** Verify with:

```bash
rg -n -f .claude/retired-symbols.txt CLAUDE.md .claude/agents .claude/skills
```

Empty output is the pass condition. `.claude/retired-symbols.txt` holds one pattern per retired identifier; the patterns live in a data file rather than inline so the check cannot match its own text. **When you retire a symbol, add it to that file.** That is what makes this a regression test for the docs rather than a one-off cleanup.
