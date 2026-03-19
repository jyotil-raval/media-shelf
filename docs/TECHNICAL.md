# media-shelf — Technical Documentation

> Architecture details, data flow, and implementation notes for the `media-shelf` project.

---

## Table of Contents

- [Project Overview](#project-overview)
- [Package Structure](#package-structure)
- [Architecture](#architecture)
- [Data Model](#data-model)
- [Database Layer](#database-layer)
- [Data Flow](#data-flow)
- [External Dependencies](#external-dependencies)
- [Configuration Reference](#configuration-reference)
- [Critical Implementation Notes](#critical-implementation-notes)

---

## Project Overview

| Field                 | Value                                                  |
| --------------------- | ------------------------------------------------------ |
| Language              | Go 1.26 · darwin/arm64 (Apple Silicon)                 |
| Module                | `github.com/jyotil-raval/media-shelf`                  |
| External dependencies | `lib/pq v1.10.9` · `cobra v1.10.2` · `godotenv v1.5.1` |
| Database              | PostgreSQL 16 (via Docker)                             |
| Status                | Phase 2 complete                                       |

**Purpose:** Local CLI tool to track anime — fetches data from MAL via `mal-updater`'s HTTP API, stores entries in a local PostgreSQL database, and provides offline-capable list, stats, and export commands.

---

## Package Structure

```
media-shelf/
├── cmd/
│   ├── main.go                  # Entry point — env, db, migrate, wire commands
│   └── shelf/
│       ├── add.go               # shelf add
│       ├── list.go              # shelf list
│       ├── stats.go             # shelf stats
│       └── export.go            # shelf export
├── internal/
│   ├── config/
│   │   └── constants.go         # All global constants
│   ├── db/
│   │   ├── db.go                # Open() + ErrNotFound + ErrDuplicate
│   │   ├── filter.go            # Filter struct for List() queries
│   │   ├── store.go             # Store interface
│   │   ├── postgres.go          # PostgreSQLStore implementation
│   │   └── db_test.go           # Table-driven tests
│   ├── models/
│   │   └── media.go             # Shared MediaItem struct
│   └── providers/
│       └── mal/
│           └── client.go        # Calls mal-updater HTTP API
├── docs/
├── .env                         # gitignored
├── .env.example
├── docker-compose.yml           # postgres:16-alpine + named volume
├── go.mod
└── go.sum
```

---

## Architecture

### Package Responsibilities

| Package                  | Key Files                                 | Responsibility                                           |
| ------------------------ | ----------------------------------------- | -------------------------------------------------------- |
| `cmd/main.go`            | `main.go`                                 | Entry point · env load · db open · migrate · wire Cobra  |
| `cmd/shelf`              | `add, list, stats, export`                | Cobra subcommands — thin wrappers over `App` methods     |
| `internal/models`        | `media.go`                                | Shared `MediaItem` struct — foundation, imports nothing  |
| `internal/db`            | `db.go, filter.go, store.go, postgres.go` | Store interface · PostgreSQLStore · error types · Filter |
| `internal/providers/mal` | `client.go`                               | HTTP client for `mal-updater` API                        |
| `internal/config`        | `constants.go`                            | All global constants                                     |

### Why the Store Interface Exists

`App` depends on `db.Store` — not `*sql.DB` directly:

```go
type App struct {
    store     db.Store      // interface — not tied to PostgreSQL
    malClient *mal.Client
}
```

**In production:** inject `PostgreSQLStore` — talks to real PostgreSQL.
**In tests:** inject an in-memory mock — no disk, no Docker, microsecond execution.

This is the Open-Closed Principle in practice:

- Open for extension — swap PostgreSQL for any other database
- Closed for modification — zero changes to commands, handlers, or tests

### Dependency Graph

```
cmd/main.go
    │
    ├── internal/db          ← Store interface + PostgreSQLStore
    │       └── internal/models
    │
    └── cmd/shelf/*          ← Cobra commands
            ├── internal/db
            ├── internal/models
            └── internal/providers/mal
                    └── internal/models
```

`internal/models` imports nothing inside this project. No circular imports possible.

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│  External                                                        │
│                                                                  │
│  ┌──────────────────────────┐   ┌──────────────────────────┐    │
│  │   mal-updater HTTP API   │   │   PostgreSQL (Docker)    │    │
│  │   :8080                  │   │   :5432                  │    │
│  │   GET /anime/:id         │   │   media_items table      │    │
│  └──────────────┬───────────┘   └──────────────┬───────────┘    │
└─────────────────┼────────────────────────────── ┼───────────────┘
                  │                               │
                  ▼                               ▼
┌─────────────────────────────────────────────────────────────────┐
│   internal/providers/mal          internal/db                   │
│   Client.GetAnime(id)             Store interface                │
│   → models.MediaItem              PostgreSQLStore                │
└──────────────────────┬────────────────────┬────────────────────┘
                       │                    │
                       ▼                    ▼
┌─────────────────────────────────────────────────────────────────┐
│   cmd/shelf/ — App struct                                        │
│   add.go    list.go    stats.go    export.go                     │
└──────────────────────────────────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│   cmd/main.go                                                    │
│   Entry point · env · db · migrate · Cobra root command          │
└──────────────────────────────────────────────────────────────────┘
```

---

## Data Model

### MediaItem Struct

```go
type MediaItem struct {
    ID        int64  `json:"id"         db:"id"`
    Title     string `json:"title"      db:"title"`
    MediaType string `json:"media_type" db:"media_type"` // always "anime"
    SubType   string `json:"sub_type"   db:"sub_type"`   // tv | movie | ova | special
    Source    string `json:"source"     db:"source"`     // always "mal"
    SourceID  string `json:"source_id"  db:"source_id"`
    Status    string `json:"status"     db:"status"`
    Score     int    `json:"score"      db:"score"`
    Progress  int    `json:"progress"   db:"progress"`
    Total     int    `json:"total"      db:"total"`
    Notes     string `json:"notes"      db:"notes"`
}
```

### Database Schema

```sql
CREATE TABLE IF NOT EXISTS media_items (
    id          SERIAL       PRIMARY KEY,
    title       TEXT         NOT NULL,
    media_type  TEXT         NOT NULL,
    sub_type    TEXT,
    source      TEXT         NOT NULL,
    source_id   TEXT,
    status      TEXT         NOT NULL,
    score       INTEGER,
    progress    INTEGER,
    total       INTEGER,
    notes       TEXT,
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_source ON media_items(source, source_id);
```

---

## Database Layer

### Error Types

```go
var (
    ErrNotFound  = errors.New("not found")
    ErrDuplicate = errors.New("duplicate entry")
)
```

Sentinel errors — always compared with `errors.Is()`, never `==`.

### Filter Struct

```go
type Filter struct {
    Status    string
    MediaType string
    SubType   string
    MinScore  int
    Sort      string // "title" | "score" | "updated_at"
}
```

Zero value means no filters — `store.List(ctx, db.Filter{})` returns everything.

### Store Interface

```go
type Store interface {
    Add(ctx context.Context, item models.MediaItem) (int64, error)
    GetByID(ctx context.Context, id int64) (*models.MediaItem, error)
    List(ctx context.Context, filter Filter) ([]models.MediaItem, error)
    Update(ctx context.Context, item models.MediaItem) error
    Delete(ctx context.Context, id int64) error
}
```

### PostgreSQLStore — Key Implementation Details

**`RETURNING id` on insert:**

```go
query := `INSERT INTO media_items (...) VALUES (...) RETURNING id`
err := s.db.QueryRowContext(ctx, query, ...).Scan(&id)
```

PostgreSQL returns the generated ID in the same statement — no second query needed.

**Duplicate detection via `pq.Error` code `23505`:**

```go
var pqErr *pq.Error
if errors.As(err, &pqErr) && pqErr.Code == "23505" {
    return 0, fmt.Errorf("add item: %w", ErrDuplicate)
}
```

`23505` is PostgreSQL's error code for unique constraint violation. Wrapping it as `ErrDuplicate` keeps the internal DB error from leaking to callers.

**Dynamic WHERE clause with numbered placeholders:**

```go
argIdx := 1
if filter.Status != "" {
    conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
    args = append(args, filter.Status)
    argIdx++
}
```

PostgreSQL requires `$1`, `$2`, `$3` — not `?`. `argIdx` tracks the current placeholder number as conditions are added.

**Compile-time interface assertion:**

```go
var _ Store = (*PostgreSQLStore)(nil)
```

Fails to build if `PostgreSQLStore` stops satisfying `Store`. Catches missing methods at the declaration site, not deep in application code.

---

## Data Flow

```
┌─────────────────────────────────────────────────────┐
│  1 · CLI Start                                       │
│  shelf add --source mal --id 1535 --status watching  │
│  Cobra parses flags → App.Add(ctx, source, id, ...)  │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│  2 · Fetch from MAL (internal/providers/mal)         │
│                                                      │
│  GET mal-updater:8080/anime/1535                     │
│  Authorization: Bearer <MAL_UPDATER_TOKEN>           │
│  → models.MediaItem                                  │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│  3 · Store (internal/db)                             │
│                                                      │
│  store.Add(ctx, item)                                │
│  INSERT INTO media_items ... RETURNING id            │
│                                                      │
│  pq error 23505? → ErrDuplicate → readable message  │
│  Success?         → return id                        │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│  Done                                                │
│  ✓ Added: Death Note                                 │
└──────────────────────────────────────────────────────┘
```

---

## External Dependencies

### mal-updater HTTP API

`http://localhost:8080` (configurable via `MAL_UPDATER_URL`)

| Endpoint        | Method | Auth | Purpose                  |
| --------------- | ------ | ---- | ------------------------ |
| `/anime/:id`    | GET    | JWT  | Fetch full anime details |
| `/anime/search` | GET    | JWT  | Search anime by query    |

### PostgreSQL

`postgres:16-alpine` running via Docker Compose on port `5432`.

---

## Configuration Reference

| Variable            | Purpose                               |
| ------------------- | ------------------------------------- |
| `DATABASE_URL`      | PostgreSQL connection string          |
| `MAL_UPDATER_URL`   | Base URL of `mal-updater` HTTP server |
| `MAL_UPDATER_TOKEN` | JWT token for `mal-updater` API auth  |

---

## Critical Implementation Notes

**PostgreSQL placeholder syntax**
Use `$1`, `$2`, `$3` — not `?`:

```sql
WHERE id = $1  -- correct
WHERE id = ?   -- wrong — SQLite syntax, fails in PostgreSQL
```

**`db.Ping()` after `sql.Open()`**
`sql.Open()` never connects. `db.Ping()` forces a real connection attempt at startup.

**`godotenv.Load()` is non-fatal**
In Docker, env vars are injected via environment — no `.env` file exists in the container.

**`rows.Close()` is mandatory**
Always `defer rows.Close()` immediately after a successful `QueryContext`. Leaving rows open leaks the connection back to the pool.

**`rows.Err()` after the loop**
`rows.Next()` returns `false` on both "no more rows" and "iterator error". Always check `rows.Err()` after the loop to distinguish them:

```go
for rows.Next() { ... }
return items, rows.Err()
```

**`errors.Is()` over `==` on wrapped errors**

```go
if errors.Is(err, db.ErrDuplicate) { ... }  // correct — traverses chain
if err == db.ErrDuplicate { ... }            // wrong — fails on wrapped errors
```

**No cgo — `lib/pq` is pure Go**
No `gcc`, no `CGO_ENABLED=1`, standard `go build` works.

**Docker volume persistence**

```bash
docker compose down      # keeps postgres_data — data survives
docker compose down -v   # deletes postgres_data — clean slate
```

---

_media-shelf · Technical Documentation · March 2026_
