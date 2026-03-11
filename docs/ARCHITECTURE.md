# อธิบายระบบ Order Management API

## ภาพรวม

ระบบนี้เป็น REST API สำหรับจัดการ Order สร้างด้วย Go ใช้แนวคิด **Clean Architecture** คือแบ่งโค้ดออกเป็น layer ชัดเจน แต่ละ layer รู้จักแค่ layer ที่อยู่ถัดไป ไม่ข้ามกัน

```
Request → Middleware → Handler → Service → Repository → Database/Cache
```

---

## โครงสร้างโฟลเดอร์

```
cmd/api/                 ← จุดเริ่มต้น app เท่านั้น 
└── main.go              ← จุดเริ่มต้นของ app (bootstrap)

internal/ 
├── server/              ← โครงสร้าง server และ routes
│   ├── server.go        ← Server struct, init ทุกอย่าง, Start/Shutdown
│   └── router.go        ← ลงทะเบียน routes ทั้งหมด
├── domain/              ← models และ interfaces หัวใจของระบบ
│   ├── models.go        ← โครงสร้างข้อมูล (User, Order)
│   ├── repository.go    ← interface ของ DB
│   └── ports.go         ← interface ของ Cache และ External API
├── handler/             ← รับ HTTP request ส่ง response
│   ├── auth_handler.go  ← รับ HTTP request ของ auth
│   ├── order_handler.go ← รับ HTTP request ของ order
│   └── health_handler.go← health check endpoints
├── service/             ← business logic ทั้งหมด
│   ├── auth_service.go  ← business logic ของ auth
│   └── order_service.go ← business logic ของ order
├── repository/              ← ติดต่อ DB และ Cache
│   ├── user_repository.go   ← query DB ของ user
│   ├── order_repository.go  ← query DB ของ order
│   ├── cache_repository.go  ← อ่าน/เขียน Redis
│   └── external_api.go      ← เรียก External API (mock)
├── middleware/          ← ด่านกรอง request ก่อนถึง handler 
│   ├── auth.go          ← ตรวจสอบ JWT token
│   ├── ratelimit.go     ← จำกัดจำนวน request ต่อ IP
│   ├── cors.go          ← จัดการ CORS headers
│   ├── logger.go        ← log ทุก request
│   ├── recovery.go      ← ดัก panic ไม่ให้ server พัง
│   └── request_id.go    ← ใส่ unique ID ให้ทุก request
├── config/              ← อ่านค่า environment variables
│   └── config.go        ← อ่าน environment variables
├── logger/              ← setup การ log
│   └── logger.go        ← setup structured logging
└── apperror/            ← format error response แบบ standard
    └── errors.go        ← standard error format
```

---

## ไล่ทีละไฟล์

### 1. `cmd/api/main.go` — จุดเริ่มต้น

ทำหน้าที่แค่ 3 อย่าง:
1. โหลด config จาก environment variables
2. สร้าง server ด้วย `server.New(cfg)`
3. รอ signal Ctrl+C แล้ว shutdown อย่าง graceful

```
main.go ไม่รู้จัก DB, Redis, หรือ business logic โดยตรง
ทุกอย่างซ่อนอยู่ใน server.New()
```

---

### 2. `internal/server/server.go` — ศูนย์กลาง init

`server.New()` ทำตามลำดับนี้:

```
initDatabase()     → เชื่อมต่อ PostgreSQL, ตั้งค่า connection pool
autoMigrate()      → สร้าง table users, orders อัตโนมัติถ้ายังไม่มี
initRedis()        → เชื่อมต่อ Redis, ping ทดสอบ
สร้าง Repositories → userRepo, orderRepo, cacheRepo
สร้าง Services     → authSvc, orderSvc
สร้าง Handlers     → authHandler, orderHandler, healthHandler
setupRouter()      → ลงทะเบียน routes
สร้าง http.Server  → กำหนด port, timeout
```

มี method สำคัญ 2 ตัว:
- `Start()` — เปิดรับ request
- `Shutdown()` — ปิด HTTP server, DB, Redis อย่าง graceful

---

### 3. `internal/server/router.go` — routes ทั้งหมด

ลงทะเบียน middleware และ routes:

```
Global middleware (ทุก request ผ่านหมด):
  RequestID   → ใส่ X-Request-ID header ให้ทุก request
  Recovery    → ดัก panic ป้องกัน server crash
  Logger      → log method, path, status, latency
  CORS        → อนุญาต cross-origin request
  RateLimit   → จำกัด request ต่อ IP (default 100 req/s)

Routes:
  /health, /health/detail, /ready, /live  → ไม่ต้อง auth
  /auth/register, /auth/login             → ไม่ต้อง auth
  /api/orders (ทุก endpoint)              → ต้องมี JWT token
```

---

### 4. `internal/domain/` — แกนกลางของระบบ

**models.go** — โครงสร้างข้อมูล:

```go
User {
  ID, Email, Password (hash), Name, CreatedAt, UpdatedAt
}

Order {
  ID, UserID, CustomerName, TotalAmount,
  Status (pending/confirmed/shipped/delivered/cancelled),
  ExternalRef, CreatedAt, UpdatedAt
}
```

**repository.go** — interface บอกว่า DB ต้องทำอะไรได้บ้าง (ไม่บอกว่าทำยังไง):

```go
UserRepository  → Create, GetByEmail, GetByID
OrderRepository → Create, GetByID, GetByUserID, UpdateStatus
```

**ports.go** — interface สำหรับ dependencies อื่น:

```go
CacheRepository    → Get, Set, Delete (generic)
OrderCache         → GetOrder, SetOrder, DeleteOrder
ExternalAPIClient  → CreateOrderRef
```

> ทำไมต้องเป็น interface? เพราะ service layer ใช้ interface ไม่ใช่ struct โดยตรง
> ทำให้ test ได้ง่ายโดยใส่ mock แทน DB จริงได้เลย

---

### 5. `internal/middleware/` — ด่านกรอง request

**auth.go** — ตรวจสอบ JWT:
1. เช็ค `Authorization: Bearer <token>` header
2. ถอดรหัส JWT ด้วย secret key
3. ดึง `user_id` จาก claims แล้วเก็บไว้ใน context
4. Handler ดึง user_id ออกมาได้ด้วย `middleware.GetUserID(c)`

**ratelimit.go** — จำกัด request ต่อ IP:
- เก็บ `rate.Limiter` แยกต่าง IP ใน map
- ใช้ Token Bucket algorithm (จาก `golang.org/x/time/rate`)
- ถ้าเกิน limit ตอบ `429 Too Many Requests` พร้อม `Retry-After: 1`

**cors.go** — จัดการ CORS:
- ใส่ headers ที่ browser ต้องการเพื่ออนุญาต cross-origin request
- จัดการ preflight request (HTTP OPTIONS)

**recovery.go** — ดัก panic:
- ถ้าโค้ดที่ไหนเกิด panic middleware นี้จะดักไว้
- ตอบ `500 Internal Server Error` แทนที่ server จะพังทั้งตัว

**logger.go** — log ทุก request:
- บันทึก method, path, status code, latency, request ID

**request_id.go** — ใส่ unique ID:
- สร้าง UUID ให้ทุก request เพื่อ trace ปัญหาได้ง่าย

---

### 6. `internal/handler/` — รับ-ส่ง HTTP

Handler ทำหน้าที่แค่:
1. Validate request body (binding)
2. เรียก service
3. ตอบ JSON กลับ

ไม่มี business logic ที่นี่เลย

**auth_handler.go:**
- `POST /auth/register` → เรียก `authService.Register()`
- `POST /auth/login` → เรียก `authService.Login()`

**order_handler.go:**
- `POST /api/orders` → ดึง userID จาก context, เรียก `orderService.Create()`
- `GET /api/orders` → รับ query params `limit`, `offset`, เรียก `orderService.GetByUserID()`
- `GET /api/orders/:id` → เรียก `orderService.GetByID()`
- `PATCH /api/orders/:id/status` → validate status, เรียก `orderService.UpdateStatus()`

**health_handler.go:**
- `GET /health` → ตอบ `{"status": "ok"}` เสมอ
- `GET /health/detail` → ping DB + Redis แล้วรายงานสถานะ
- `GET /ready` → เช็ค DB พร้อมรับ traffic ไหม
- `GET /live` → เช็คว่า process ยังมีชีวิตอยู่ไหม (สำหรับ Kubernetes)

---

### 7. `internal/service/` — business logic

**auth_service.go:**

Register:
```
1. เช็คว่า email ซ้ำไหม
2. hash password ด้วย bcrypt
3. สร้าง User พร้อม UUID
4. บันทึกลง DB
5. สร้าง JWT token
6. คืน user + token
```

Login:
```
1. หา user จาก email
2. เทียบ password กับ hash (bcrypt.CompareHashAndPassword)
3. สร้าง JWT token
4. คืน user + token
```

JWT token มี claims:
- `user_id` — UUID ของ user
- `email` — email ของ user
- `exp` — วันหมดอายุ (default 24 ชั่วโมง)
- algorithm HS256

**order_service.go:**

Create:
```
1. สร้าง Order พร้อม UUID, status = "pending"
2. บันทึกลง DB
3. เรียก External API เพื่อขอ reference number
4. Cache order ใน Redis (TTL 5 นาที)
5. คืน order
```

GetByID (Cache-first pattern):
```
1. ลองดึงจาก Redis ก่อน → ถ้าเจอคืนเลย (เร็ว)
2. ถ้าไม่มีใน cache → ดึงจาก DB
3. เช็คว่า order เป็นของ user คนนี้ไหม
4. เก็บใน Redis สำหรับครั้งถัดไป
5. คืน order
```

UpdateStatus:
```
1. ดึง order จาก DB
2. เช็คว่าเป็นของ user คนนี้ไหม
3. อัปเดต status ใน DB
4. ลบ cache เพื่อให้ครั้งถัดไปดึงข้อมูลใหม่จาก DB
5. คืน order ที่อัปเดตแล้ว
```

---

### 8. `internal/repository/` — ติดต่อ DB และ Cache

**user_repository.go / order_repository.go:**
- implement interface จาก `domain/repository.go`
- ใช้ GORM เป็น ORM สำหรับ query PostgreSQL
- ไม่มี business logic ที่นี่ แค่ CRUD

**cache_repository.go:**
- implement `CacheRepository` และ `OrderCache` พร้อมกันใน struct เดียว
- ใช้ Redis เก็บ Order เป็น JSON
- key format: `order:<id>` เช่น `order:abc-123`
- TTL 300 วินาที (5 นาที)

**external_api.go:**
- Mock implementation ของ External API
- ในระบบจริงจะเรียก HTTP ไปยัง service ภายนอก
- ตอนนี้สร้าง reference number แบบ random

---

## Flow ของ request จริงๆ

### ตัวอย่าง: `GET /api/orders/:id`

```
1. Request เข้ามา
   ↓
2. Middleware: RequestID → ใส่ UUID ให้ request
   ↓
3. Middleware: Logger → เริ่มจับเวลา
   ↓
4. Middleware: RateLimit → เช็ค IP ว่าเกิน limit ไหม
   ↓
5. Middleware: AuthRequired → ตรวจ JWT, ดึง user_id ใส่ context
   ↓
6. Handler: OrderHandler.GetOrder()
   - ดึง user_id จาก context
   - ดึง order_id จาก URL param
   ↓
7. Service: OrderService.GetByID()
   - ลองดึงจาก Redis (cache hit → จบที่นี่)
   - ถ้าไม่มี → ดึงจาก PostgreSQL
   - เช็คว่า order เป็นของ user คนนี้ไหม
   - เก็บใน Redis
   ↓
8. ตอบ JSON กลับ + Logger บันทึก latency
```

---

## เทคโนโลยีและเหตุผลที่เลือก

| เทคโนโลยี | เหตุผล |
|-----------|--------|
| **Gin** | Web framework ที่เร็วและนิยมใน Go |
| **PostgreSQL** | Relational DB เหมาะกับข้อมูล order ที่ต้องการ consistency |
| **GORM** | ORM ช่วยลดโค้ด SQL boilerplate |
| **Redis** | Cache ช่วยลด latency ตอนดึง order ที่ถูกเรียกบ่อย |
| **JWT** | Stateless authentication ไม่ต้องเก็บ session ใน server |
| **bcrypt** | Hash password แบบ slow hash ป้องกัน brute force |
| **Token Bucket** | Rate limiting algorithm ที่ยืดหยุ่น รองรับ burst ได้ |
| **Docker** | Deploy ง่าย environment เหมือนกันทุกที่ |

---

## สิ่งที่ทำให้โค้ดนี้ production-ready

1. **Graceful Shutdown** — รอให้ request ที่ค้างอยู่เสร็จก่อนค่อยปิด server
2. **Connection Pool** — จำกัดจำนวน DB connections ไม่ให้ล้น
3. **Cache-first** — ลด load ที่ DB โดยดึงจาก Redis ก่อน
4. **Rate Limiting** — ป้องกัน abuse ต่อ IP
5. **Structured Logging** — log เป็น JSON พร้อม request ID ใช้ trace ปัญหาได้
6. **Health Checks** — Kubernetes ใช้ `/ready` และ `/live` ตรวจสอบสถานะ
7. **Recovery Middleware** — ดัก panic ป้องกัน server crash
8. **Interface-based design** — test ได้ง่ายโดยใช้ mock แทน dependency จริง
