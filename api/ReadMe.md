# Eventify - Multi-Protocol API

A modern event management system supporting HTTP REST, gRPC, and GraphQL APIs with shared business logic and maximum component reuse.

## Project Structure

```
/Users/talabi/repo/bezaeel/go/context/eventify/
├── api/
│   ├── http/                    # HTTP REST endpoints
│   │   ├── controllers/         # HTTP controllers
│   │   ├── middlewares/        # HTTP middlewares
│   │   └── routes/             # HTTP route definitions
│   ├── grpc/                   # gRPC endpoints
│   │   ├── handlers/           # gRPC handlers
│   │   ├── interceptors/       # gRPC interceptors
│   │   └── proto/              # Protocol buffer definitions
│   └── graphql/                # GraphQL endpoints
│       ├── resolvers/          # GraphQL resolvers
│       ├── schemas/            # GraphQL schema definitions
│       └── directives/         # GraphQL directives
├── internal/
│   ├── domain/                 # Shared domain models
│   ├── service/                # Shared business logic
│   ├── repository/             # Shared data access layer
│   └── shared/                 # Shared utilities
│       ├── auth/               # Authentication & authorization
│       ├── config/             # Configuration management
│       └── constants/          # Shared constants
├── pkg/                        # Shared packages
│   ├── database/               # Database connection
│   ├── logger/                 # Logging utilities
│   └── telemetry/              # Observability & telemetry
├── cmd/
│   ├── http-server/            # HTTP server binary
│   ├── grpc-server/            # gRPC server binary
│   └── graphql-server/         # GraphQL server binary
└── tests/
    ├── integration/
    │   ├── http/               # HTTP integration tests
    │   ├── grpc/               # gRPC integration tests
    │   └── graphql/            # GraphQL integration tests
    └── unit/
        ├── http/               # HTTP unit tests
        ├── grpc/               # gRPC unit tests
        └── graphql/            # GraphQL unit tests
```

## Architecture Overview

### Shared Components

All three API protocols (HTTP, gRPC, GraphQL) share the same core components:

- **Domain Models**: Single source of truth for data structures
- **Business Logic**: Shared service layer for all business operations
- **Data Access**: Unified repository pattern for database operations
- **Authentication**: Common JWT-based authentication system
- **Telemetry**: Unified observability and monitoring
- **Configuration**: Centralized configuration management

### API-Specific Components

Each API type has its own transport layer:

- **HTTP**: Controllers, middlewares, and route definitions
- **gRPC**: Handlers, interceptors, and protocol buffer definitions
- **GraphQL**: Resolvers, schemas, and directives

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL
- Docker (for development)

### Running the Servers

#### HTTP Server (REST API)
```bash
cd cmd/http-server
go run main.go
# Server starts on :3000
```

#### gRPC Server
```bash
cd cmd/grpc-server
go run main.go
# Server starts on :50051
```

#### GraphQL Server
```bash
cd cmd/graphql-server
go run main.go
# Server starts on :8080
```

### Database Setup

```bash
# Run migrations
go run cmd/http-server/main.go migrate

# Seed initial data
go run cmd/http-server/main.go seed
```

## API Documentation

### HTTP REST API
- **Swagger UI**: `http://localhost:3000/docs`
- **Base URL**: `http://localhost:3000/api/v1`

### gRPC API
- **Reflection**: Available for development tools
- **Port**: `:50051`

### GraphQL API
- **Playground**: `http://localhost:8080`
- **Endpoint**: `http://localhost:8080/query`

## Development

### Running Tests

```bash
# All tests
go test ./tests/...

# HTTP tests only
go test ./tests/unit/http/...
go test ./tests/integration/http/...

# gRPC tests only
go test ./tests/unit/grpc/...
go test ./tests/integration/grpc/...

# GraphQL tests only
go test ./tests/unit/graphql/...
go test ./tests/integration/graphql/...
```

### Code Generation

#### gRPC
```bash
# Generate protobuf code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/grpc/proto/*.proto
```

#### GraphQL
```bash
# Generate GraphQL code (when using gqlgen)
go run github.com/99designs/gqlgen generate
```

## Key Features

### Maximum Component Reuse
- **Single Business Logic**: All APIs use the same service layer
- **Shared Domain Models**: Consistent data structures across protocols
- **Unified Authentication**: Same JWT system for all APIs
- **Common Telemetry**: Unified observability and monitoring

### Protocol-Specific Optimizations
- **HTTP**: RESTful design with JSON responses
- **gRPC**: High-performance binary protocol
- **GraphQL**: Flexible querying with schema introspection

### Scalability
- **Independent Deployment**: Each API can be deployed separately
- **Different Scaling**: Each API can scale independently
- **API Gateway Ready**: Can be fronted by an API gateway

## Contributing

1. Follow the existing code structure
2. Add tests for new features
3. Update documentation
4. Ensure all APIs remain consistent

## License

[Add your license here]
