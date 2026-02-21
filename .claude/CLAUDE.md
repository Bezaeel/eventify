# Eventify – Claude Code Instructions

## Project Overview

**Module:** `eventify` (Go 1.24.2)
Multi-transport Go service exposing REST (Fiber v2), gRPC, and GraphQL (gqlgen) APIs with OpenTelemetry tracing, GORM/PostgreSQL persistence, and JWT auth.

## Directory Structure

```
cmd/
  http-server/     # Fiber REST entry point
  grpc-server/     # gRPC entry point
  graphql-server/  # GraphQL entry point
api/
  http/controllers/  # HTTP handlers (versioned under v1/)
  grpc/handlers/     # gRPC service handlers
  grpc/proto/        # Protobuf definitions
  graphql/           # resolvers, schemas, generated code
internal/
  domain/            # Pure domain models (no framework deps)
  repository/database/ # GORM repository implementations + migrations
  service/           # Business logic; mocks live in service/mocks/
  shared/            # auth, config, constants
pkg/
  database/          # Postgres connection helper
  logger/            # Structured logger
  telemetry/         # OTel configurator + adapter (mocks in telemetry/mocks/)
tests/
  unit/              # Unit tests (http, grpc, graphql, helpers)
  integration/       # Integration tests via testcontainers
docs/                # Swagger JSON/YAML + generation scripts
```

## Essential Commands

```bash
# Run servers
make run-http
make run-grpc
make run-graphql

# Testing
make test                     # unit + integration
make test-unit                # go test ./tests/unit/...
make test-integration         # go test ./tests/integration/...
make test-coverage            # full coverage HTML report
make test-unit-coverage
make test-integration-coverage

# Code generation
make mock                     # regenerate all gomock mocks
make swagger                  # swag init + add_xtags.go post-processor

# Database migrations
make migrate-up
make migrate-down
make migrate-create           # prompts for migration name

# Build
make build                    # go build -o eventify cmd/http-server/main.go
make clean

# Install toolchain
make deps                     # mockgen, swag, golang-migrate
```

## Static Analysis

`staticcheck.conf` enables all checks except ST1003 (naming conventions):
```
checks = ["all", "-ST1003"]
```

Run manually:
```bash
staticcheck ./...
go vet ./...
```

Permitted via `.claude/settings.local.json`: `go build`, `go vet`, `staticcheck`.

## Key Conventions

- **Dependency injection** – services receive interfaces, not concrete types; mocks are generated with `mockgen`.
- **Error responses** – use `pkg/ErrorResponse.go` patterns; do not invent new response shapes.
- **Telemetry** – wrap spans via `pkg/telemetry`; use the adapter interface so it stays mockable.
- **Config** – loaded through `internal/shared/config` via `github.com/spf13/viper` + `.env`.
- **Auth** – JWT via `github.com/golang-jwt/jwt/v5`; middleware in `api/http/middlewares/`.
- **Migrations** – sequential SQL files in `internal/repository/database/migrations/`; always create via `make migrate-create`.
- **GraphQL codegen** – schema edits go in `api/graphql/schemas/`; regenerate with `gqlgen generate` (config: `gqlgen.yml`).

## Go Performance Checklist

Apply these on all new or modified code paths. Check off each item before opening a PR.

### String / Integer Formatting
- [ ] **`strconv` over `fmt.Sprintf` on hot paths** – `fmt.Sprintf` uses reflection; prefer:
  - `strconv.Itoa(n)` instead of `fmt.Sprintf("%d", n)`
  - `strconv.FormatInt(n, 10)` for `int64`
  - `strconv.FormatBool(b)` instead of `fmt.Sprintf("%t", b)`
  - `strconv.AppendInt(buf, n, 10)` for zero-alloc append patterns

### Binary / Container Size
- [ ] **Strip debug symbols for production builds** – add `-ldflags="-s -w"` to the build command:
  ```bash
  go build -ldflags="-s -w" -o eventify cmd/http-server/main.go
  ```
  `-s` removes the symbol table; `-w` removes DWARF debug info. Reduces binary size and container image layer.
- [ ] Confirm `Makefile` `build` target uses stripped flags before shipping a release image.

### Struct Field Alignment
- [ ] **Run `fieldalignment` on new/modified structs** to eliminate padding waste:
  ```bash
  go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
  fieldalignment -fix ./...
  ```
  Order fields largest → smallest (pointer/int64 → int32 → int16 → int8/bool → smaller) to minimise struct size and reduce GC pressure.
- [ ] Verify with `go vet -vettool=$(which fieldalignment) ./...` in CI.

### General
- [ ] Avoid unnecessary heap allocations in request-handling loops (profile with `go test -bench . -benchmem`).
- [ ] Prefer `strings.Builder` or `bytes.Buffer` over repeated string concatenation (`+`) in loops.
- [ ] Use `sync.Pool` for short-lived, frequently allocated objects (e.g. byte slices, request-scoped buffers).

## Testing Guidelines

### Which test type to write

| Code under test | Test type | Location |
|---|---|---|
| Uses raw SQL queries (repository layer) | Integration test | `tests/integration/` |
| Everything else (services, handlers, resolvers) | Unit test with mocks | `tests/unit/` |

**Raw SQL → Integration test** – any function that constructs or executes a raw SQL string (e.g. `db.Raw(...)`, `db.Exec(...)`, hand-written query strings) must be covered by an integration test that runs against a real Postgres instance via `testcontainers-go`. Mocks cannot validate query correctness.

**No raw SQL → Unit test with mocks** – service methods, HTTP/gRPC/GraphQL handlers, and any logic that calls through an interface should use `gomock`-generated mocks. Run `make mock` to regenerate after any interface change.

### Rules

- Mirror the package path of the code under test inside `tests/unit/` or `tests/integration/`.
- Always run `make mock` after changing a service interface before running tests.
- Use `-count=1` to bypass the test cache (already set in Makefile targets).
- Table-driven tests are preferred; name sub-tests with `t.Run`.
- Integration tests must not depend on external state – spin up a fresh container per test suite and tear it down after.

## Do Not

- Do not commit to `main` directly; always open a PR.
- Do not add framework-specific types to `internal/domain/`.
- Do not skip `make mock` after interface changes – stale mocks cause confusing test failures.
- Do not use `fmt.Println` for logging; use the structured logger in `pkg/logger`.
- Do not embed secrets in source; use `.env` (git-ignored) loaded by Viper.
