# Expense Tracker — Backend

Go REST API built with Gin, PostgreSQL, and Redis following the Handler → Service → Repository architecture.

## Tech Stack

| Technology | Purpose |
|---|---|
| Go + Gin | API server |
| PostgreSQL 16 | Database (UUID primary keys) |
| pgx v5 | PostgreSQL driver (no ORM) |
| squirrel | SQL query builder |
| goose | Database migrations |
| Redis 7 | Rate limiting + summary caching |
| JWT (golang-jwt) | Authentication tokens |
| bcrypt | Password hashing |

## Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go           # Entry point — wires up all dependencies
├── internal/
│   ├── config/
│   │   └── config.go         # Loads config from environment variables
│   ├── middleware/
│   │   ├── cors.go           # CORS headers
│   │   ├── jwt.go            # JWT token validation
│   │   ├── logger.go         # Structured request logging
│   │   ├── ratelimiter.go    # Redis-backed per-IP rate limiting
│   │   ├── recovery.go       # Panic recovery
│   │   └── requestid.go      # Attaches UUID to every request
│   ├── handler/
│   │   ├── auth.go           # POST /auth/register, POST /auth/login
│   │   ├── category.go       # GET/POST /categories
│   │   └── expense.go        # GET/POST/PUT/DELETE /expenses, GET /expenses/summary
│   ├── service/
│   │   ├── auth.go           # Register, login, JWT generation
│   │   ├── category.go       # Category business logic
│   │   └── expense.go        # Expense CRUD + Redis cache invalidation
│   ├── repository/
│   │   ├── user.go           # User SQL queries
│   │   ├── category.go       # Category SQL queries + default seeding
│   │   └── expense.go        # Expense SQL queries with filtering
│   ├── model/
│   │   ├── user.go           # User struct
│   │   ├── category.go       # Category struct
│   │   └── expense.go        # Expense + filter + summary structs
│   └── response/
│       └── response.go       # Standardized JSON response helpers
└── migrations/
    ├── 001_create_users.sql
    ├── 002_create_categories.sql
    └── 003_create_expenses.sql
```

## Architecture

Every request flows through this exact chain — no layers are skipped:

```
HTTP Request
  → Middleware (Recovery → RequestID → Logger → CORS → RateLimiter → JWTAuth)
  → Handler   (parse request body, validate input, call service)
  → Service   (business logic, orchestration)
  → Repository (SQL via squirrel + pgx)
  → PostgreSQL
```

## API Endpoints

### Auth (no authentication required)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/auth/register` | Create account, returns JWT + user |
| POST | `/api/auth/login` | Login, returns JWT + user |

**Register request:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "secret123"
}
```

**Login request:**
```json
{
  "email": "john@example.com",
  "password": "secret123"
}
```

**Auth response:**
```json
{
  "data": {
    "token": "<jwt>",
    "user": { "id": "uuid", "email": "...", "name": "..." }
  }
}
```

### Categories (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/categories` | List all categories for current user |
| POST | `/api/categories` | Create a new category |

### Expenses (JWT required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/expenses` | List expenses with optional filters |
| POST | `/api/expenses` | Create a new expense |
| PUT | `/api/expenses/:id` | Update an expense |
| DELETE | `/api/expenses/:id` | Delete an expense |
| GET | `/api/expenses/summary` | Get totals by category + overall |

**Expense list query params:**
- `category_id` — filter by category UUID
- `start_date` — filter from date (YYYY-MM-DD)
- `end_date` — filter to date (YYYY-MM-DD)
- `cursor` — cursor for next page (from `next_cursor` in response)
- `limit` — page size (default: 20)

**Create/update expense request:**
```json
{
  "category_id": "uuid",
  "amount": 150.00,
  "description": "Lunch at cafe",
  "expense_date": "2026-03-02"
}
```

### Response Format

Every response uses one of these structured formats:

```json
// Success (single item)
{ "data": { ... } }

// Success (list with cursor pagination)
{ "data": [...], "next_cursor": "2026-03-01T10:00:00Z" }

// Error
{ "error": { "code": "VALIDATION_REQUIRED", "message": "amount is required" } }
```

## Environment Variables

Copy `.env.example` to `.env` and set the values:

```env
PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=expense_user
DB_PASSWORD=expense_pass
DB_NAME=expense_tracker
DB_SSLMODE=disable
REDIS_URL=localhost:6379
JWT_SECRET=change-this-to-a-random-secret
CORS_ORIGIN=http://localhost:5173
```

## Running Locally

### 1. Start PostgreSQL and Redis

```bash
# from the project root
docker compose up -d
```

### 2. Run migrations

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest

goose -dir migrations postgres \
  "postgres://expense_user:expense_pass@localhost:5432/expense_tracker?sslmode=disable" up
```

### 3. Start the server

```bash
go run cmd/server/main.go
```

The server starts at `http://localhost:8080`.

### Build binary

```bash
go build -o expense-tracker ./cmd/server
./expense-tracker
```

## Database Schema

### users
```sql
id            UUID PRIMARY KEY DEFAULT gen_random_uuid()
email         VARCHAR(255) UNIQUE NOT NULL
name          VARCHAR(255) NOT NULL
password_hash VARCHAR(255) NOT NULL
created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

### categories
```sql
id         UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
name       VARCHAR(100) NOT NULL
icon       VARCHAR(50) NOT NULL DEFAULT ''
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

Default categories are automatically seeded on user registration:
`Food`, `Transport`, `Shopping`, `Bills`, `Entertainment`, `Health`, `Other`

### expenses
```sql
id           UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
category_id  UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT
amount       DECIMAL(12,2) NOT NULL CHECK (amount > 0)
description  VARCHAR(500) NOT NULL DEFAULT ''
expense_date DATE NOT NULL DEFAULT CURRENT_DATE
created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

## Redis Usage

| Key pattern | Purpose | TTL |
|---|---|---|
| `rate_limit:<ip>` | Per-IP request counter (max 100/min) | 60s |
| `summary:<user_id>*` | Cached expense summary per user/date range | 5 min |

Summary cache is invalidated automatically on any expense create, update, or delete.

## Middleware Chain

Applied globally to all routes in this order:

1. **Recovery** — catches panics, returns 500
2. **RequestID** — generates UUID, sets `X-Request-ID` header
3. **Logger** — logs method, path, status, latency
4. **CORS** — sets `Access-Control-Allow-*` headers
5. **RateLimiter** — Redis counter per client IP, 100 req/min
