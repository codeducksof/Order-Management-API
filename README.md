# Order Management API

A production-ready REST API for order management built with Go, featuring clean architecture, JWT authentication, Redis caching, and comprehensive testing.

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## Features

- **Clean Architecture**: Domain-driven design with clear separation of concerns
- **JWT Authentication**: Secure token-based authentication with configurable expiration
- **Redis Caching**: Optimized read performance with cache-first pattern
- **PostgreSQL**: Robust data persistence with GORM ORM
- **Rate Limiting**: Per-IP rate limiting to prevent abuse
- **Graceful Shutdown**: Proper handling of SIGTERM/SIGINT signals
- **Structured Logging**: JSON logging with request ID correlation
- **Health Checks**: Kubernetes-ready liveness and readiness probes
- **API Documentation**: OpenAPI/Swagger specification
- **Docker Support**: Multi-stage builds with Docker Compose

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.21+ |
| Web Framework | Gin |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| ORM | GORM |
| Authentication | JWT (HS256) |
| Container | Docker |

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go           # Application entry point
├── internal/
│   ├── apperror/             # Standardized error handling
│   ├── config/               # Configuration management
│   ├── domain/               # Domain models and interfaces
│   ├── handler/              # HTTP handlers (controllers)
│   ├── logger/               # Structured logging
│   ├── middleware/           # HTTP middleware
│   ├── mocks/                # Test mocks
│   ├── repository/           # Data access layer
│   └── service/              # Business logic layer
├── docs/                     # API documentation
├── .env.example              # Environment variables template
├── .golangci.yml             # Linter configuration
├── docker-compose.yml        # Docker services
├── Dockerfile                # Multi-stage Docker build
├── Makefile                  # Build automation
└── README.md
```

## Quick Start

### Prerequisites

- **Docker** and **Docker Compose** (recommended)
- Or: **Go 1.21+**, **PostgreSQL 16**, **Redis 7**

### Option 1: Docker Compose (Recommended)

```bash
# Clone the repository
git clone <repository-url>
cd order-management-api

# Create environment file
cp .env.example .env

# Edit .env and set a secure JWT_SECRET (minimum 32 characters)
# JWT_SECRET=your-secure-secret-key-at-least-32-characters

# Start all services
docker-compose up -d

# Verify the API is running
curl http://localhost:8080/health
```

### Option 2: Local Development

```bash
# Install dependencies
go mod download

# Set up environment
cp .env.example .env
# Edit .env with your database and Redis connection strings

# Run the application
make run

# Or build and run
make build
./bin/api
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `GIN_MODE` | Gin mode (debug/release) | `debug` |
| `JWT_SECRET` | JWT signing key (min 32 chars) | **Required** |
| `JWT_EXPIRATION` | Token expiration duration | `24h` |
| `DATABASE_URL` | PostgreSQL connection string | `localhost:5432` |
| `DATABASE_MAX_OPEN_CONNS` | Max open DB connections | `25` |
| `DATABASE_MAX_IDLE_CONNS` | Max idle DB connections | `5` |
| `REDIS_URL` | Redis connection string | `localhost:6379` |
| `RATE_LIMIT_RPS` | Requests per second per IP | `100` |
| `RATE_LIMIT_BURST` | Burst capacity | `200` |
| `SERVER_READ_TIMEOUT` | HTTP read timeout | `15s` |
| `SERVER_WRITE_TIMEOUT` | HTTP write timeout | `15s` |
| `SERVER_SHUTDOWN_TIMEOUT` | Graceful shutdown timeout | `30s` |

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/register` | Register new user |
| POST | `/auth/login` | Login and get JWT |

### Orders (Requires Authentication)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/orders` | Create new order |
| GET | `/api/orders` | List user's orders |
| GET | `/api/orders/:id` | Get order by ID |
| PATCH | `/api/orders/:id/status` | Update order status |

### Health Checks

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Basic health check |
| GET | `/health/detail` | Detailed health with dependencies |
| GET | `/ready` | Kubernetes readiness probe |
| GET | `/live` | Kubernetes liveness probe |

## Usage Examples

### Register a User

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "John Doe"
  }'
```

### Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

Save the `token` from the response for authenticated requests.

### Create an Order

```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <YOUR_TOKEN>" \
  -d '{
    "customer_name": "Customer A",
    "total_amount": 199.99
  }'
```

### List Orders

```bash
curl -X GET "http://localhost:8080/api/orders?limit=10&offset=0" \
  -H "Authorization: Bearer <YOUR_TOKEN>"
```

### Update Order Status

```bash
curl -X PATCH http://localhost:8080/api/orders/<ORDER_ID>/status \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <YOUR_TOKEN>" \
  -d '{
    "status": "confirmed"
  }'
```

Valid statuses: `pending`, `confirmed`, `shipped`, `delivered`, `cancelled`

## Development

### Available Make Commands

```bash
make help           # Show all available commands
make build          # Build the application
make run            # Run the application
make test           # Run all tests
make test-coverage  # Run tests with coverage report
make lint           # Run linter
make fmt            # Format code
make check          # Run all checks (fmt, vet, lint, test)
make docker-up      # Start Docker services
make docker-down    # Stop Docker services
make install-tools  # Install development tools
make generate-mocks # Generate test mocks
make swagger        # Generate Swagger docs
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration
```

### Code Quality

```bash
# Install development tools
make install-tools

# Run linter
make lint

# Format code
make fmt

# Run all checks
make check
```

## Architecture

### Clean Architecture Layers

```
┌─────────────────────────────────────────┐
│              Handlers                    │  HTTP Layer
│         (Request/Response)               │
├─────────────────────────────────────────┤
│              Services                    │  Business Logic
│         (Use Cases)                      │
├─────────────────────────────────────────┤
│             Domain                       │  Core Entities
│      (Models & Interfaces)               │
├─────────────────────────────────────────┤
│           Repositories                   │  Data Access
│     (DB, Cache, External APIs)           │
└─────────────────────────────────────────┘
```

### Request Flow

```
Client → Middleware → Handler → Service → Repository → Database
                                    ↓
                                  Cache
                                    ↓
                              External API
```

## Production Deployment

### Environment Checklist

- [ ] Set `GIN_MODE=release`
- [ ] Set secure `JWT_SECRET` (min 32 characters)
- [ ] Configure proper database credentials
- [ ] Set up TLS/HTTPS
- [ ] Configure appropriate rate limits
- [ ] Set up log aggregation
- [ ] Configure health check monitoring

### Kubernetes

Use the health endpoints for Kubernetes probes:

```yaml
livenessProbe:
  httpGet:
    path: /live
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

## API Documentation

OpenAPI/Swagger documentation is available at `docs/swagger.json`.

You can import this file into:
- [Swagger Editor](https://editor.swagger.io/)
- [Postman](https://www.postman.com/)
- [Insomnia](https://insomnia.rest/)

## Error Responses

All error responses follow a consistent format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "detail": "Additional details (optional)"
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Invalid input data |
| `UNAUTHORIZED` | 401 | Missing or invalid authentication |
| `INVALID_CREDENTIALS` | 401 | Wrong email or password |
| `TOKEN_EXPIRED` | 401 | JWT token has expired |
| `NOT_FOUND` | 404 | Resource not found |
| `USER_EXISTS` | 409 | Email already registered |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make check`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
