# Eventify

Eventify is a Go-based REST API for event management with role-based access control (RBAC).

## Features

- 🔐 JWT Authentication
- 👥 Role-Based Access Control (RBAC)
- 📅 Event Management
- 📚 Swagger Documentation
- 🔄 Database Migrations
- ✅ Unit and Integration Tests

## Prerequisites

- Go 1.19 or higher
- PostgreSQL 12 or higher
- Make

## Project Structure

```
.
├── cmd/                    # Application entrypoints
├── docs/                   # Swagger documentation
├── internal/               # Private application code
│   ├── api/               # API handlers and middleware
│   ├── auth/              # Authentication logic
│   ├── config/            # Configuration
│   ├── database/          # Database migrations
│   ├── domain/            # Domain models
│   └── service/           # Business logic
├── pkg/                    # Public libraries
├── test/                  # Integration tests
├── .env                   # Environment variables (create from .env.example)
├── go.mod                 # Go modules
└── Makefile              # Build automation
```

## Getting Started

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd eventify
   ```

2. Install dependencies:
   ```bash
   make deps
   ```

3. Create and configure your environment variables:
   ```bash
   cp .env.example .env
   # Edit .env with your database credentials and other configurations
   ```

4. Run database migrations:
   ```bash
   make migrate-up
   ```

5. Generate Swagger documentation:
   ```bash
   make swagger
   ```

6. Run the application:
   ```bash
   make run
   ```

The API will be available at `http://localhost:3000` and Swagger documentation at `http://localhost:3000/swagger/`.

## Available Make Commands

- `make run` - Run the application
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage report
- `make mock` - Generate mocks for testing
- `make swagger` - Generate Swagger documentation
- `make migrate-create` - Create a new migration
- `make migrate-up` - Run migrations up
- `make migrate-down` - Run migrations down
- `make deps` - Install dependencies
- `make clean` - Clean build artifacts
- `make build` - Build the application
- `make config` - Show current configuration
- `make all` - Run clean, swagger, mock, test, and run

## Environment Variables

Create a `.env` file in the root directory with the following variables:

```env
# Database Configuration
DB_USER=postgres
DB_PASSWORD=your_password
DB_HOST=localhost
DB_PORT=5432
DB_NAME=eventify

# JWT Configuration
JWT_SECRET=your_jwt_secret

# Server Configuration
PORT=3000
```

## API Documentation

The API documentation is available through Swagger UI at `http://localhost:3000/swagger/` when the application is running.

### Main Endpoints

- **Authentication**
  - POST `/api/v1/auth/login` - User login
  - POST `/api/v1/auth/register` - User registration

- **Events**
  - GET `/api/v1/events` - List all events
  - POST `/api/v1/events` - Create a new event
  - GET `/api/v1/events/{id}` - Get event details
  - PUT `/api/v1/events/{id}` - Update an event
  - DELETE `/api/v1/events/{id}` - Delete an event

- **Admin**
  - GET `/api/v1/admin/roles` - List all roles
  - POST `/api/v1/admin/roles` - Create a new role
  - POST `/api/v1/admin/roles/assign` - Assign role to user
  - POST `/api/v1/admin/permissions/assign` - Assign permission to role

## Testing

The project includes both unit and integration tests. All tests run without cache to ensure fresh results every time. You can run them separately or together:

### Run All Tests
```bash
make test
```

### Run Unit Tests Only
```bash
make test-unit
```

### Run Integration Tests Only
```bash
make test-integration
```

### Generate Coverage Reports
```bash
# All tests coverage
make test-coverage

# Unit tests coverage
make test-unit-coverage

# Integration tests coverage
make test-integration-coverage
```

Coverage reports will be generated in HTML format and automatically opened in your default browser.

Note: All test commands use the `-count=1` flag to bypass the test cache and ensure fresh test runs.

## Contributing

1. Create a new branch for your feature
2. Make your changes
3. Run tests and ensure they pass
4. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
