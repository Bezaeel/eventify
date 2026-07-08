-include .env
export

# Workspace-level orchestrator. Each module is buildable on its own; these
# targets fan out across all of them.

MODULES     := platform events outbox api subscribers
BUILD_FLAGS := -ldflags="-s -w"
BIN_DIR     := bin

DB_USER     ?= postgres
DB_PASSWORD ?= postgres
DB_HOST     ?= localhost
DB_PORT     ?= 5432
DB_NAME     ?= eventify
DB_URL      := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Migrations live beside the module that owns their tables. They are applied
# against one database, in dependency order.
MIGRATION_DIRS := api/internal/migrations outbox/migrations subscribers/migrations

.PHONY: help build test test-unit test-integration vet staticcheck check \
        migrate-up migrate-down migrate-create mock swagger clean deps \
        run-http run-grpc run-graphql run-relay run-subscriber

help:
	@echo "build              Build every binary into $(BIN_DIR)/ (stripped)"
	@echo "check              vet + staticcheck + test across all modules"
	@echo "test-unit          Unit tests (no Docker required)"
	@echo "test-integration   Integration tests (testcontainers; needs Docker)"
	@echo "migrate-up/down    Apply/revert migrations for every module"
	@echo "migrate-create     Scaffold a migration: make migrate-create MODULE=api NAME=add_foo"
	@echo "run-http|grpc|graphql|relay|subscriber   Run a server"

## ---- build -----------------------------------------------------------------
# -s strips the symbol table, -w strips DWARF. Required for release images.
build:
	@mkdir -p $(BIN_DIR)
	go build $(BUILD_FLAGS) -o $(BIN_DIR)/http-server    ./api/cmd/http-server
	go build $(BUILD_FLAGS) -o $(BIN_DIR)/grpc-server    ./api/cmd/grpc-server
	go build $(BUILD_FLAGS) -o $(BIN_DIR)/graphql-server ./api/cmd/graphql-server
	go build $(BUILD_FLAGS) -o $(BIN_DIR)/outbox-relay   ./outbox/cmd/relay
	go build $(BUILD_FLAGS) -o $(BIN_DIR)/subscriber     ./subscribers/cmd/subscriber

## ---- quality ---------------------------------------------------------------
vet:
	@for m in $(MODULES); do echo "vet $$m"; (cd $$m && go vet ./...) || exit 1; done

staticcheck:
	@for m in $(MODULES); do echo "staticcheck $$m"; (cd $$m && staticcheck ./...) || exit 1; done

fieldalignment:
	@for m in $(MODULES); do (cd $$m && go vet -vettool=$$(which fieldalignment) ./...) || exit 1; done

check: vet staticcheck test

## ---- tests -----------------------------------------------------------------
test: test-unit test-integration

test-unit:
	@for m in $(MODULES); do echo "unit: $$m"; (cd $$m && go test -count=1 -short ./...) || exit 1; done

test-integration:
	@for m in $(MODULES); do echo "integration: $$m"; (cd $$m && go test -count=1 -run Integration ./...) || exit 1; done

test-coverage:
	go test -count=1 ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

## ---- codegen ---------------------------------------------------------------
mock:
	go run github.com/golang/mock/mockgen -source=platform/telemetry/telemetryAdapter.go \
		-destination=platform/telemetry/mocks/telemetry_adapter_mock.go -package=mocks

swagger:
	cd api && swag init -g cmd/http-server/main.go -o docs && go run docs/scripts/add_xtags.go

## ---- migrations ------------------------------------------------------------
migrate-up:
	@for d in $(MIGRATION_DIRS); do echo "migrate up: $$d"; migrate -path $$d -database "$(DB_URL)" up || exit 1; done

migrate-down:
	@for d in $(MIGRATION_DIRS); do echo "migrate down: $$d"; migrate -path $$d -database "$(DB_URL)" down 1 || exit 1; done

# usage: make migrate-create MODULE=api NAME=add_event_status
migrate-create:
	@test -n "$(MODULE)" || (echo "MODULE is required, e.g. MODULE=api" && exit 1)
	@test -n "$(NAME)"   || (echo "NAME is required, e.g. NAME=add_event_status" && exit 1)
	@dir=$$( [ "$(MODULE)" = "api" ] && echo api/internal/migrations || echo $(MODULE)/migrations ); \
	migrate create -ext sql -dir $$dir -seq $(NAME)

## ---- run -------------------------------------------------------------------
run-http:       ; go run ./api/cmd/http-server
run-grpc:       ; go run ./api/cmd/grpc-server
run-graphql:    ; go run ./api/cmd/graphql-server
run-relay:      ; go run ./outbox/cmd/relay
run-subscriber: ; go run ./subscribers/cmd/subscriber

## ---- housekeeping ----------------------------------------------------------
deps:
	go work sync
	go install github.com/golang/mock/mockgen@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest

# Deliberately narrow. The previous `clean` ran `rm -rf docs`, which deleted
# docs/scripts/add_xtags.go — a hand-written file that `make swagger` then
# tried to run. Only generated artefacts are removed here.
clean:
	rm -rf $(BIN_DIR) coverage.out coverage.html
	rm -f api/docs/docs.go api/docs/swagger.json api/docs/swagger.yaml
	go clean
