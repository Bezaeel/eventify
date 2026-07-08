---
name: add-endpoint-version
description: Version an endpoint correctly — decide whether the change is a wire-contract change (new DTO only) or a behaviour change (new command + handler). Use when asked to add /v2 of an endpoint, change a response shape, deprecate a field, or evolve a gRPC/GraphQL surface.
---

# Version an endpoint

**Decide which axis changed before you touch a file.** Getting this wrong is how `Get2AllEvents` came to exist: a behaviour fork was welded onto the interface that every version and every transport shared.

## The decision

```
Did the SQL / the rules / the side effects change?
├── NO  → wire contract only  → new DTO in a new transport version dir.
│                               The feature handler is NOT touched.
└── YES → behaviour           → new command type + new handler file.
                                Old handler stays, untouched, serving old callers.
```

Field renamed, field dropped from a response, response re-enveloped, param made optional → **contract only**.
Different filter, extra write, new validation rule, different rows returned → **behaviour**.

## Case A: contract only (the common case)

Copy the adapter, change the DTO, map to the **same** command.

```
api/internal/transport/http/
  v1/events/update_event.go   → UpdateEventRequestV1 → events.UpdateEventCommand
  v2/events/update_event.go   → UpdateEventRequestV2 → events.UpdateEventCommand
```

Nothing under `internal/features/` changes. No new SQL. No new tests for the handler — only a unit test for the new adapter's decode/encode.

## Case B: behaviour changed

New command, new handler, new file. Both live side by side.

```
api/internal/features/events/
  update_event.go      UpdateEventCommand    (v1 callers)
  update_event_v2.go   UpdateEventV2Command  (v2 callers)
```

```go
// update_event_v2.go
type UpdateEventV2Command struct{ /* ... */ }
type UpdateEventV2Handler struct{ db postgres.Querier }
func (h UpdateEventV2Handler) Handle(ctx context.Context, cmd UpdateEventV2Command) (UpdateEventV2Result, error)
```

Never:

```go
// WRONG — the fork is now visible to every version and every transport
type IEventService interface {
	GetAllEvents(ctx) []Event
	Get2AllEvents(ctx) []Event   // what is "2"?
}
```

Each handler gets its own integration test. Deleting v2 later is deleting one file.

## Per-transport mechanics

**HTTP** — version in the path. One directory per version under `transport/http/`.

```go
app.Group("/api/v1/events")
app.Group("/api/v2/events")
```

**gRPC** — version in the *proto package*, never the URL.

```proto
package eventify.events.v2;
option go_package = "eventify/api/internal/transport/grpc/eventsv2";
```

Generate into a new Go package. The v1 service keeps serving; register both on the server.

**GraphQL** — **do not create `/v2/graphql`.** Evolve the schema additively:

```graphql
type Event {
  organizer  String @deprecated(reason: "Use organiser. Removed after 2026-12-01.")
  organiser  String
}
```

Add fields, never remove them without a deprecation window. A `/v2` GraphQL endpoint defeats the point of the schema.

## Sunsetting

Because a v1 adapter holds only decode/encode:

```bash
rm -rf api/internal/transport/http/v1/events/
```

Then delete the v1 route registration. If deleting a version requires touching `internal/features/`, the version boundary leaked — fix that first.

Announce with `Deprecation` / `Sunset` response headers before removing.

## Checklist

- [ ] Classified the change: contract-only or behaviour
- [ ] Contract-only → `internal/features/` untouched (verify with `git diff --stat`)
- [ ] Behaviour → new command type, new file; no numbered methods on a shared type
- [ ] GraphQL change is additive with `@deprecated`, not a new endpoint
- [ ] gRPC change bumps the proto package, not a URL
- [ ] Old version still has passing tests
