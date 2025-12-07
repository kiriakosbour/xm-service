# XM Company Service

A production-ready Go microservice for managing companies with REST API, PostgreSQL, Kafka events, and JWT authentication.

## Features

- **CRUD Operations**: Create, Read, Update (Patch), Delete companies
- **JWT Authentication**: Protected endpoints for mutations
- **Kafka Events**: Event publishing on create, update, and delete operations
- **PostgreSQL**: Persistent storage with proper constraints
- **Docker Support**: Full Docker and Docker Compose setup
- **Health Checks**: Liveness and readiness endpoints
- **Graceful Shutdown**: Proper handling of OS signals
- **Comprehensive Tests**: Unit tests, handler tests, and integration tests

## Company Entity

| Field       | Type    | Constraints                                                      |
|-------------|---------|------------------------------------------------------------------|
| ID          | UUID    | Required, auto-generated                                         |
| Name        | String  | Required, max 15 chars, unique                                   |
| Description | String  | Optional, max 3000 chars                                         |
| Employees   | Integer | Required, >= 0                                                   |
| Registered  | Boolean | Required                                                         |
| Type        | Enum    | Required: Corporations, NonProfit, Cooperative, Sole Proprietorship |

## Quick Start

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Make (optional)

### Using Docker Compose (Recommended)

```bash
# Start all services (app, PostgreSQL, Kafka, Zookeeper)
docker-compose up -d --build

# View logs
docker-compose logs -f app

# Stop all services
docker-compose down
```

The API will be available at `http://localhost:8080`

### Local Development

```bash
# Start external services only
docker-compose up -d db kafka zookeeper

# Install dependencies
go mod download

# Run the application
go run ./cmd/server

# Or use Make
make run
```

### Environment Variables

| Variable               | Default                                              | Description                |
|------------------------|------------------------------------------------------|----------------------------|
| SERVER_PORT            | :8080                                                | HTTP server port           |
| DB_URL                 | postgres://user:pass@localhost:5432/xm?sslmode=disable | PostgreSQL connection URL |
| KAFKA_BROKERS          | localhost:9092                                       | Kafka broker addresses     |
| KAFKA_TOPIC            | company-events                                       | Kafka topic for events     |
| KAFKA_ENABLED          | true                                                 | Enable/disable Kafka       |
| JWT_SECRET             | your-256-bit-secret-key-here                         | JWT signing secret         |

## API Endpoints

### Health Checks

```bash
# Liveness probe
GET /health/live

# Readiness probe (includes DB check)
GET /health/ready
```

### Public Endpoints

```bash
# Get a company by ID
GET /companies/{id}
```

### Protected Endpoints (Require JWT)

All mutation endpoints require an `Authorization: Bearer <token>` header.

```bash
# Create a company
POST /companies
Content-Type: application/json

{
  "name": "Acme Corp",
  "description": "A sample company",
  "employees": 100,
  "registered": true,
  "type": "Corporations"
}

# Update a company (partial update)
PATCH /companies/{id}
Content-Type: application/json

{
  "employees": 150,
  "registered": false
}

# Delete a company
DELETE /companies/{id}
```

## API Examples

### Create a Company

```bash
curl -X POST http://localhost:8080/companies \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "name": "TechStartup",
    "description": "An innovative tech company",
    "employees": 50,
    "registered": true,
    "type": "Corporations"
  }'
```

### Get a Company

```bash
curl http://localhost:8080/companies/550e8400-e29b-41d4-a716-446655440000
```

### Update a Company

```bash
curl -X PATCH http://localhost:8080/companies/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "employees": 75
  }'
```

### Delete a Company

```bash
curl -X DELETE http://localhost:8080/companies/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer your-token"
```

## Testing

```bash
# Run unit tests
make test

# Run with coverage
make test-coverage

# Run integration tests (requires running database)
TEST_DB_URL="postgres://xm_user:xm_password@localhost:5432/xm_test?sslmode=disable" \
  go test -v -tags=integration ./tests/...

# Run linter
make lint
```

## Project Structure

```
xm-company-service/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── core/
│   │   ├── domain.go         # Domain models and validation
│   │   └── ports.go          # Interface definitions
│   ├── handler/
│   │   ├── http.go           # HTTP handlers
│   │   └── health.go         # Health check handlers
│   ├── middleware/
│   │   └── auth.go           # JWT authentication middleware
│   ├── platform/
│   │   ├── kafka/
│   │   │   └── producer.go   # Kafka event producer
│   │   └── postgres/
│   │       └── repository.go # PostgreSQL repository
│   └── service/
│       └── company.go        # Business logic
├── migrations/
│   └── 001_init.sql          # Database migrations
├── tests/
│   └── integration_test.go   # Integration tests
├── .golangci.yml             # Linter configuration
├── docker-compose.yml        # Docker Compose configuration
├── Dockerfile                # Multi-stage Docker build
├── go.mod                    # Go module definition
├── Makefile                  # Build and development commands
└── README.md                 # This file
```

## Architecture

The project follows a clean/hexagonal architecture pattern:

- **Core**: Domain models, business rules, and port interfaces
- **Service**: Business logic implementation
- **Handler**: HTTP transport layer
- **Platform**: Infrastructure implementations (PostgreSQL, Kafka)

## Kafka Events

Events are published to Kafka on mutations:

- `CompanyCreated`: When a new company is created
- `CompanyUpdated`: When a company is updated
- `CompanyDeleted`: When a company is deleted

Event format:
```json
{
  "type": "CompanyCreated",
  "payload": { /* company object */ },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Production Considerations

1. **JWT Authentication**: The current implementation is a mock. In production, implement proper JWT validation with signature verification.

2. **Secrets Management**: Use a proper secrets management solution (Vault, AWS Secrets Manager, etc.) instead of environment variables for sensitive data.

3. **Logging**: Consider adding structured logging (zerolog, zap) for better observability.

4. **Metrics**: Add Prometheus metrics for monitoring.

5. **Rate Limiting**: Implement rate limiting for API endpoints.

6. **HTTPS**: Use TLS in production.

## License

MIT
