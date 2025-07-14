# Include .env file if it exists
-include .env

# Default values for database connection (override these in .env file)
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_NAME ?= eventify

.PHONY: run test test-unit test-integration test-coverage mock swagger migrate-up migrate-down migrate-create clean

# Go parameters
BINARY_NAME=eventify
MAIN_FILE= cmd/main.go
MIGRATION_DIR=internal/database/migrations

# Tools
MOCKGEN=go run github.com/golang/mock/mockgen
SWAG=swag
MIGRATE=migrate

# Database URL for migrations
DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Default command
all: clean swagger mock test run

# Run the application
run:
	@echo "Running application..."
	cd cmd && go run .

# Run all tests (unit + integration)
test: test-unit test-integration
	@echo "All tests completed."

# Run only unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v -count=1 ./tests/unit/... -cover

# Run only integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -count=1 ./tests/integration/... -cover

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -count=1 ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

# Run unit tests with coverage report
test-unit-coverage:
	@echo "Running unit tests with coverage..."
	go test -v -count=1 -run UnitTests ./tests/unit/... -coverprofile=unit-coverage.out
	go tool cover -html=unit-coverage.out

# Run integration tests with coverage report
test-integration-coverage:
	@echo "Running integration tests with coverage..."
	go test -v -count=1 -run IntegrationTests ./tests/integration/... -coverprofile=integration-coverage.out
	go tool cover -html=integration-coverage.out

# Generate mocks
mock:
	@echo "Generating mocks..."
	$(MOCKGEN) -source=internal/service/event_service.go -destination=internal/service/mocks/event_service_mock.go -package=mocks
	$(MOCKGEN) -source=internal/service/user_service.go -destination=internal/service/mocks/user_service_mock.go -package=mocks
	$(MOCKGEN) -source=internal/service/role_service.go -destination=internal/service/mocks/role_service_mock.go -package=mocks
	$(MOCKGEN) -source=internal/service/permission_service.go -destination=internal/service/mocks/permission_service_mock.go -package=mocks

# Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	$(SWAG) init -g $(MAIN_FILE) -o docs && \
	go run docs/scripts/add_xtags.go

# Create a new migration
migrate-create:
	@read -p "Enter migration name: " name; \
	$(MIGRATE) create -ext sql -dir $(MIGRATION_DIR) -seq $$name

# Run migrations up
migrate-up:
	@echo "Running migrations up..."
	@echo "Using database URL: $(DB_URL)"
	$(MIGRATE) -path $(MIGRATION_DIR) -database "$(DB_URL)" up

# Run migrations down
migrate-down:
	@echo "Running migrations down..."
	@echo "Using database URL: $(DB_URL)"
	$(MIGRATE) -path $(MIGRATION_DIR) -database "$(DB_URL)" down

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go install github.com/golang/mock/mockgen@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -rf docs
	go clean

# Build the application
build:
	@echo "Building application..."
	go build -o $(BINARY_NAME) $(MAIN_FILE)

# Show current configuration
config:
	@echo "Current configuration:"
	@echo "  DB_USER: $(DB_USER)"
	@echo "  DB_HOST: $(DB_HOST)"
	@echo "  DB_PORT: $(DB_PORT)"
	@echo "  DB_NAME: $(DB_NAME)"
	@echo "  DB_URL:  $(DB_URL)"

# Help command
help:
	@echo "Available commands:"
	@echo "  make run                    - Run the application"
	@echo "  make test                   - Run all tests (unit + integration)"
	@echo "  make test-unit              - Run only unit tests"
	@echo "  make test-integration       - Run only integration tests"
	@echo "  make test-coverage          - Run all tests with coverage report"
	@echo "  make test-unit-coverage     - Run unit tests with coverage report"
	@echo "  make test-integration-coverage - Run integration tests with coverage report"
	@echo "  make mock                   - Generate mocks"
	@echo "  make swagger                - Generate Swagger documentation"
	@echo "  make migrate-create         - Create a new migration"
	@echo "  make migrate-up             - Run migrations up"
	@echo "  make migrate-down           - Run migrations down"
	@echo "  make deps                   - Install dependencies"
	@echo "  make clean                  - Clean build artifacts"
	@echo "  make build                  - Build the application"
	@echo "  make config                 - Show current configuration"
	@echo "  make all                    - Run clean, swagger, mock, test, and run"
