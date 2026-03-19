# media-shelf — Technical Documentation

> Architecture details, data flow, and implementation notes for the `media-shelf` project.

---

## Table of Contents

- [Project Overview](#project-overview)
- [Package Structure](#package-structure)
- [Architecture](#architecture)
- [Data Model](#data-model)
- [Database Layer](#database-layer)
- [CLI Layer](#cli-layer)
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
| Status                | Phase 3 complete                                       |

**Purpose:** Local CLI tool to track anime — fetches data from MAL via `mal-updater`'s HTTP API, stores entries in a local PostgreSQL database, and provides offline-capable list, stats, and export commands.

---

## Package Structure

```
media-shelf/
├── cmd/
│   ├── main.go                  # Entry point — env, db, migrate, wire commands
│   └── shelf/
│       ├── app.go               # App struct + all command method stubs
│       ├── root.go              # Cobra root command — registers subcommands
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
| `cmd/shelf/app.go`       | `app.go`                                  | App struct · dependency container · all command methods  |
| `cmd/shelf`              | `root, add, list, stats, export`          | Cobra commands — thin wrappers over App methods          |
| `internal/models`        | `media.go`                                | Shared `MediaItem` struct — foundation, imports nothing  |
| `internal/db`            | `db.go, filter.go, store.go, postgres.go` | Store interface · PostgreSQLStore · error types · Filter |
| `internal/providers/mal` | `client.go`                               | HTTP client for `mal-updater` API                        |
| `internal/config`        | `constants.go`                            | All global constants                                     |

### Cobra Dependency Injection Pattern

```go
// cmd/main.go — wiring
store := db.NewPostgreSQLStore(database)   // real DB in production
app   := shelf.NewApp(store)               // inject store into App
root  := shelf.NewRootCommand(app)         // inject app into commands
root.Execute()

// cmd/shelf/app.go — App owns dependencies
type App struct {
    store     db.Store       // interface — not tied to PostgreSQL
    malClient *mal.Client    // added in Phase 4
}

// cmd/shelf/add.go — command is a thin closure wrapper
RunE: func(cmd *cobra.Command, args []string) error {
    return app.Add(cmd.Context(), source, id, status)
}
```

`app` is captured by the closure — `RunE` never needs it in its signature. Each command independently testable by injecting a mock store into `App`.

### Why `RunE` Over `Run`

`Run` ignores errors — a failed DB write silently exits with code `0`. `RunE` returns an `error` — Cobra prints it and exits with a non-zero code. Always use `RunE` for commands that can fail.

### Dependency Graph

```
cmd/main.go
    │
    ├── internal/db          ← Store interface + PostgreSQLStore
    │       └── internal/models
    │
    └── cmd/shelf/*          ← Cobra commands via App
            ├── internal/db
            ├── internal/models
            └── internal/providers/mal  (Phase 4)
                    └── internal/models
```

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│  External                                                        │
│                                                                  │
│  ┌──────────────────────────┐   ┌──────────────────────────┐    │
│  │   mal-updater HTTP API   │   │   PostgreSQL (Docker)    │    │
│  │   :8080                  │   │   :5432                  │    │
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
│   cmd/shelf/app.go — App struct                                  │
│   Add()   List()   Stats()   Export()                            │
└──────────────────────────────────────────────────────────────────┘
                       │
              ┌────────┴────────┐
              ▼                 ▼
┌─────────────────┐   ┌──────────────────────────────────────────┐
│  cmd/shelf/     │   │  cmd/main.go                             │
│  root.go        │   │  Entry point · wires store → app → cobra │
│  add.go         │   └──────────────────────────────────────────┘
│  list.go        │
│  stats.go       │
│  export.go      │
└─────────────────┘
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

### PostgreSQLStore Key Details

- `RETURNING id` on insert — one statement, no second query
- `pq.Error` code `23505` → `ErrDuplicate`
- Dynamic WHERE clause with `$1`, `$2` numbered placeholders
- `rows.Close()` + `rows.Err()` on `List()`
- `RowsAffected() == 0` → `ErrNotFound` on `Update()` and `Delete()`
- `var _ Store = (*PostgreSQLStore)(nil)` — compile-time interface assertion

---

## CLI Layer

### Command Structure

```
shelf
├── add     --id (required) --status (required) --source (default: mal)
├── list    --status --type --subtype --score --sort
├── stats
└── export  --format (default: json) --output (default: shelf.json)
```

### Flag Binding Pattern

Cobra flags bind directly to local variables via `StringVar` / `IntVar`. The closure captures them by reference — by the time `RunE` executes, the flags are already populated:

```go
var status string
cmd.Flags().StringVar(&status, "status", "", "Watch status")

RunE: func(cmd *cobra.Command, args []string) error {
    // status is already set by Cobra before RunE runs
    return app.List(cmd.Context(), db.Filter{Status: status})
}
```

### `cmd.Context()` Propagation

`RunE` passes `cmd.Context()` to every `App` method. This context carries the signal from OS interrupts (Ctrl+C). When the user cancels, the context is cancelled — any in-flight DB query or HTTP call respects it automatically.

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
│  Success?         → print confirmation               │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│  Done · ✓ Added: Death Note                          │
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
Use `$1`, `$2` — not `?` (SQLite syntax):

```sql
WHERE id = $1  -- correct
WHERE id = ?   -- wrong
```

**`db.Ping()` after `sql.Open()`**
`sql.Open()` never connects. `db.Ping()` forces a real connection attempt at startup.

**`godotenv.Load()` is non-fatal**
In Docker, env vars are injected via environment — no `.env` file in container.

**`rows.Close()` is mandatory**
Always `defer rows.Close()` after a successful `QueryContext`.

**`rows.Err()` after the loop**

```go
for rows.Next() { ... }
return items, rows.Err()
```

**`errors.Is()` over `==`**

```go
if errors.Is(err, db.ErrDuplicate) { ... }  // correct
if err == db.ErrDuplicate { ... }            // wrong on wrapped errors
```

**Closure captures in Cobra commands**
`RunE` captures `app` and flag variables from the enclosing `newXxxCommand()` function. This is how commands access the store without global state.

**`RunE` over `Run`**
`Run` ignores errors. `RunE` returns them. Always use `RunE`.

**No cgo — `lib/pq` is pure Go**
Standard `go build` — no C compiler needed.

**Docker volume persistence**

```bash
docker compose down      # data preserved
docker compose down -v   # clean slate
```

---

_media-shelf · Technical Documentation · March 2026_
