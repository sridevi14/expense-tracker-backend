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

Copy `.env.example` to `.env` and fill in your values. Store passwords as **plain text** (not URL-encoded) — the application handles encoding internally.

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | _(empty)_ | PostgreSQL password (plain text) |
| `DB_NAME` | `expense_tracker` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode |
| `REDIS_URL` | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | _(empty)_ | Redis password (blank if none) |
| `JWT_SECRET` | _(dev fallback)_ | JWT signing secret |
| `CORS_ORIGIN` | `http://localhost:5173` | Allowed CORS origin |

## Running Locally (without Docker)

### Prerequisites
- Go 1.25+
- PostgreSQL 16+ running
- Redis 7+ running

### 1. Copy and fill in your environment

```bash
cp .env.example .env
# Edit .env with your credentials
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Start the server

Migrations run **automatically** at startup. No separate migration step needed.

```bash
go run cmd/server/main.go
```

The server starts at `http://localhost:8080`.

---

## Running with Docker

The Dockerfile builds a self-contained image. It does **not** pull a Postgres or Redis image — it connects to your existing servers.

### Startup sequence inside the container

1. Waits for PostgreSQL to be reachable (`pg_isready`)
2. Creates the database if it does not already exist
3. Runs all pending goose migrations (idempotent — already-applied migrations are skipped)
4. Starts the API server

### 1. Copy and fill in your environment

```bash
cp .env.example .env
```

Key values to set in `.env`:

```env
DB_HOST=host.docker.internal   # use host.docker.internal on Mac/Windows
                                # use 172.17.0.1 on Linux to reach host Postgres
DB_PASSWORD=your_plain_password
REDIS_URL=host.docker.internal:6379
REDIS_PASSWORD=                 # blank if no Redis auth
JWT_SECRET=a-long-random-secret
CORS_ORIGIN=http://localhost:5173
```

### 2. Build the image

```bash
# Run from the backend/ directory
docker build -t expense-tracker-backend .
```

### 3. Run the container

```bash
docker run \
  --env-file .env \
  -p 8080:8080 \
  expense-tracker-backend
```

The API is now available at `http://localhost:8080`.

### Build binary (without Docker)

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
