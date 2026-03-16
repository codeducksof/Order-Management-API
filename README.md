# Order Management API

A production-ready REST API for order management built with **Go**, applying **Clean Architecture** principles with JWT authentication, Redis caching, rate limiting, and comprehensive unit tests.

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## What This Project Demonstrates

- **Clean Architecture** — แยก layer ชัดเจน (Handler → Service → Repository → Domain) แต่ละ layer ไม่ผูกติดกัน
- **JWT Authentication** — Register/Login พร้อม token-based auth ทุก protected route
- **Redis Caching** — Cache-first pattern ลด latency ตอนดึง order
- **Rate Limiting** — จำกัด request ต่อ IP ด้วย Token Bucket algorithm
- **Graceful Shutdown** — รอ request ที่ค้างอยู่ให้เสร็จก่อน shutdown
- **Structured Logging** — JSON log พร้อม Request ID ทุก request
- **Health Checks** — Kubernetes-ready `/ready` และ `/live` endpoints
- **Unit Tests** — ครอบคลุม service และ handler layer ด้วย mock

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.23 |
| Web Framework | Gin |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| ORM | GORM |
| Authentication | JWT (HS256) |
| Container | Docker + Docker Compose |

---

## Project Structure

```
cmd/api/
└── main.go                  # Bootstrap: load config, start server

internal/
├── server/
│   ├── server.go            # Server struct, init dependencies, Start/Shutdown
│   └── router.go            # Register all routes and middleware
├── domain/
│   ├── models.go            # Core entities: User, Order
│   ├── repository.go        # DB interfaces
│   └── ports.go             # Cache and External API interfaces
├── handler/                 # HTTP layer: parse request, call service, return response
├── service/                 # Business logic: auth, order management
├── repository/              # Data access: PostgreSQL, Redis, External API (mock)
├── middleware/              # JWT auth, rate limiting, CORS, logging, recovery
├── config/                  # Environment variable management
├── logger/                  # Structured logging setup
└── apperror/                # Standardized error format

docs/                        # API documentation + Architecture guide
```

---

## Quick Start

### Option 1: Docker Compose (Recommended)

```bash
# Clone the repository
git clone <repository-url>
cd order-management-api

# Setup environment
cp .env.example .env
# แก้ JWT_SECRET ใน .env ให้ยาวอย่างน้อย 32 ตัวอักษร

# Start all services (API + PostgreSQL + Redis)
docker-compose up -d

# Verify
curl http://localhost:8080/health
```

### Option 2: Local Development

```bash
# Setup environment
cp .env.example .env

# Start only database and cache
docker-compose up -d postgres redis

# Run API locally
make run
```

---

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/register` | สมัครสมาชิก |
| POST | `/auth/login` | เข้าสู่ระบบ รับ JWT token |

### Orders `(ต้องการ JWT)`

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/orders` | สร้าง order ใหม่ |
| GET | `/api/orders` | ดู order ทั้งหมดของ user |
| GET | `/api/orders/:id` | ดู order ตาม ID |
| PATCH | `/api/orders/:id/status` | อัปเดตสถานะ order |
| DELETE | `/api/orders/:id` | ลบ order (เฉพาะ `pending` หรือ `cancelled`) |

### Health Checks

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | สถานะ API |
| GET | `/health/detail` | สถานะพร้อม DB + Redis latency |
| GET | `/ready` | Kubernetes readiness probe |
| GET | `/live` | Kubernetes liveness probe |

---

## Usage Examples

### 1. Register

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123", "name": "John Doe"}'
# {"token": "eyJhbGci...", "user": {"id": "uuid-...", "email": "user@example.com", "name": "John Doe"}}
```

### 2. Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'
# {"token": "eyJhbGci...", "user": {...}}

TOKEN="eyJhbGci..."
```

### 3. Create Order

```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"customer_name": "Customer A", "total_amount": 199.99}'
# {"id": "uuid-...", "status": "pending", "total_amount": 199.99, ...}
```

### 4. List Orders

```bash
curl "http://localhost:8080/api/orders?limit=10&offset=0" \
  -H "Authorization: Bearer $TOKEN"
# {"orders": [...], "total": 1, "limit": 10, "offset": 0}
```

### 5. Get Order by ID

```bash
curl http://localhost:8080/api/orders/<ORDER_ID> \
  -H "Authorization: Bearer $TOKEN"
```

### 6. Update Order Status

```bash
curl -X PATCH http://localhost:8080/api/orders/<ORDER_ID>/status \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"status": "confirmed"}'
```

Valid statuses: `pending` → `confirmed` → `shipped` → `delivered` / `cancelled`

### 7. Delete Order

```bash
curl -X DELETE http://localhost:8080/api/orders/<ORDER_ID> \
  -H "Authorization: Bearer $TOKEN"
# 204 No Content
```

> สามารถลบได้เฉพาะ order ที่มีสถานะ `pending` หรือ `cancelled` เท่านั้น

---

## Architecture

```
Client Request
     │
     ▼
┌─────────────────────────────────────────┐
│            Middleware                    │  RequestID, Logger, Recovery,
│                                          │  CORS, RateLimit, AuthRequired
├─────────────────────────────────────────┤
│             Handler                      │  Parse request, validate input
├─────────────────────────────────────────┤
│             Service                      │  Business logic
├─────────────────────────────────────────┤
│           Repository                     │  PostgreSQL / Redis / External API
└─────────────────────────────────────────┘
```

> อ่านรายละเอียดเพิ่มเติมได้ที่ [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

---

## Development

```bash
make run            # Run the application
make test           # Run unit tests
make test-coverage  # Run tests with HTML coverage report
make build          # Build binary to ./bin/api
make lint           # Run linter (golangci-lint)
make fmt            # Format code
make docker-up      # Start all Docker services
make docker-down    # Stop all Docker services
```

---

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `GIN_MODE` | `debug` / `release` | `debug` |
| `JWT_SECRET` | JWT signing key (min 32 chars) | **Required** |
| `JWT_EXPIRATION` | Token expiration | `24h` |
| `DATABASE_URL` | PostgreSQL connection string | — |
| `REDIS_URL` | Redis connection string | — |
| `RATE_LIMIT_RPS` | Requests per second per IP | `100` |
| `RATE_LIMIT_BURST` | Burst capacity | `200` |

---

## Error Response Format

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message"
  }
}
```

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Invalid input |
| `UNAUTHORIZED` | 401 | Missing or invalid token |
| `INVALID_CREDENTIALS` | 401 | Wrong email or password |
| `NOT_FOUND` | 404 | Resource not found |
| `USER_EXISTS` | 409 | Email already registered |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |
| (conflict message) | 409 | Cannot delete non-pending/cancelled order |

---

## Production Checklist

- [ ] `GIN_MODE=release`
- [ ] `JWT_SECRET` อย่างน้อย 32 ตัวอักษร
- [ ] ตั้งค่า database credentials ที่ปลอดภัย
- [ ] เปิดใช้ TLS/HTTPS
- [ ] ตั้งค่า rate limit ตามการใช้งานจริง
- [ ] เชื่อมต่อ log aggregation (เช่น Loki, Datadog)
