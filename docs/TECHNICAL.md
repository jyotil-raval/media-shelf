# media-shelf — Technical Documentation

> Architecture details, data flow, and implementation notes for the `media-shelf` project.

---

## Table of Contents

- [Project Overview](#project-overview)
- [Package Structure](#package-structure)
- [Architecture](#architecture)
- [Data Model](#data-model)
- [Data Flow](#data-flow)
- [External Dependencies](#external-dependencies)
- [Configuration Reference](#configuration-reference)
- [Critical Implementation Notes](#critical-implementation-notes)

---

## Project Overview

| Field | Value |
|---|---|
| Language | Go 1.26 · darwin/arm64 (Apple Silicon) |
| Module | `github.com/jyotil-raval/media-shelf` |
| External dependencies | `lib/pq v1.10.9` · `cobra v1.10.2` · `godotenv v1.5.1` |
| Database | PostgreSQL 16 (via Docker) |
| Status | Phase 1 complete |

**Purpose:** Local CLI tool to track anime — fetches data from MAL via `mal-updater`'s HTTP API, stores entries in a local PostgreSQL database, and provides offline-capable list, stats, and export commands.

**Relationship to mal-updater:** `media-shelf` is a consumer of `mal-updater` (Project 1). It never calls the MAL API directly — all MAL data flows through `mal-updater`'s JWT-protected REST API.

---

## Package Structure

```
media-shelf/
├── cmd/
│   ├── main.go                  # Entry point — env, db, migrate, wire commands
│   └── shelf/
│       ├── add.go               # shelf add — fetch from MAL + store locally
│       ├── list.go              # shelf list — filter + display
│       ├── stats.go             # shelf stats — aggregations
│       └── export.go            # shelf export — JSON + CSV
├── internal/
│   ├── config/
│   │   └── constants.go         # All global constants
│   ├── db/
│   │   ├── db.go                # Store interface + PostgreSQLStore + error types
│   │   ├── migrations.go        # Schema creation — IF NOT EXISTS guards
│   │   └── db_test.go           # Table-driven tests
│   ├── models/
│   │   └── media.go             # Shared MediaItem struct — imported by all packages
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

| Package | Key Files | Responsibility |
|---|---|---|
| `cmd/main.go` | `main.go` | Entry point · env load · db open · migrate · wire Cobra commands |
| `cmd/shelf` | `add, list, stats, export` | Cobra subcommands — thin wrappers over `App` methods |
| `internal/models` | `media.go` | Shared `MediaItem` struct — imported by all packages, imports nothing |
| `internal/db` | `db.go, migrations.go` | `Store` interface · `PostgreSQLStore` · error types · schema |
| `internal/providers/mal` | `client.go` | HTTP client for `mal-updater` API — returns `models.MediaItem` |
| `internal/config` | `constants.go` | All global constants |

### Dependency Graph

```
cmd/main.go
    │
    ├── internal/db          ← Store interface + PostgreSQLStore
    │       │
    │       └── internal/models   ← MediaItem struct
    │
    └── cmd/shelf/*          ← Cobra commands
            │
            ├── internal/db
            ├── internal/models
            └── internal/providers/mal  ← calls mal-updater
                    │
                    └── internal/models
```

`internal/models` imports nothing inside this project — it is the foundation. Every other package imports `models`. No circular imports are possible in this graph.

### Why `Store` Interface Over Direct `*sql.DB`

`cmd/shelf/add.go` depends on `db.Store` — not `*sql.DB` directly:

```go
type App struct {
    store     db.Store
    malClient *mal.Client
}
```

In production: `App` gets a real `PostgreSQLStore` backed by PostgreSQL.
In tests: `App` gets an in-memory mock — no disk, no network, runs in microseconds.

The Cobra command never knows which implementation it has. It calls `a.store.Add()` and trusts the contract.

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
│   cmd/shelf/                                                     │
│   App struct — store + malClient                                 │
│                                                                  │
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
    Status    string `json:"status"     db:"status"`     // watching | completed | on_hold | dropped | plan_to
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
    media_type  TEXT         NOT NULL,   -- always "anime"
    sub_type    TEXT,                    -- tv | movie | ova | special
    source      TEXT         NOT NULL,   -- always "mal"
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

### Design Decisions

**Single Table Inheritance** — all anime in one table regardless of subtype. The most common query is `WHERE status = 'watching'` — this works across the full shelf without JOINs or UNIONs.

**`SubType` field** — handles anime movies, OVAs, and specials. An anime movie (`media_type: anime`, `sub_type: movie`) is distinct from a standalone movie (`media_type: movie`). This allows `shelf list --subtype movie` to return only anime films.

**UNIQUE INDEX on `(source, source_id)`** — deduplication enforced at the database level. Attempting to add the same MAL anime twice fails at the constraint — the application catches `ErrDuplicate` and returns a readable message.

**`SERIAL` over `INTEGER PRIMARY KEY AUTOINCREMENT`** — PostgreSQL's `SERIAL` creates an implicit sequence (`media_items_id_seq`) and sets the column default to `nextval()`. Functionally identical to SQLite's `AUTOINCREMENT` but backed by a proper sequence object.

**`TIMESTAMPTZ` over `DATETIME`** — PostgreSQL stores timestamps with timezone awareness. `DATETIME` in SQLite is a plain string with no timezone. `TIMESTAMPTZ` stores UTC internally and converts on read.

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
│  INSERT INTO media_items ...                         │
│                                                      │
│  Duplicate? → ErrDuplicate → readable message        │
│  Success?   → print confirmation                     │
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

| Endpoint | Method | Auth | Purpose |
|---|---|---|---|
| `/anime/:id` | GET | JWT | Fetch full anime details |
| `/anime/search` | GET | JWT | Search anime by query |

JWT token stored in `.env` as `MAL_UPDATER_TOKEN` — issued once from `mal-updater`'s `POST /auth/token`.

### PostgreSQL

`postgres:16-alpine` running via Docker Compose on port `5432`.

---

## Configuration Reference

Environment variables (`.env`):

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `MAL_UPDATER_URL` | Base URL of `mal-updater` HTTP server |
| `MAL_UPDATER_TOKEN` | JWT token for `mal-updater` API auth |

---

## Critical Implementation Notes

**PostgreSQL placeholder syntax**
PostgreSQL uses `$1`, `$2`, `$3` for query parameters — not `?` like SQLite:
```sql
-- WRONG (SQLite style)
WHERE id = ?

-- CORRECT (PostgreSQL style)
WHERE id = $1
```

**`db.Ping()` after `sql.Open()`**
`sql.Open()` never connects — it only validates the driver name. `db.Ping()` forces an immediate connection attempt. Always call it at startup for network databases.

**`godotenv.Load()` is non-fatal**
In Docker, `DATABASE_URL` is injected via environment — no `.env` file exists in the container. `godotenv.Load()` is called without error checking so the app starts cleanly in both local and container environments.

**`rows.Close()` is mandatory**
Always `defer rows.Close()` immediately after a successful `QueryContext`. Leaving rows open leaks the connection back to the pool.

**`rows.Err()` after the loop**
`rows.Next()` returns `false` on both "no more rows" and "iterator error". Check `rows.Err()` after the loop to distinguish them:
```go
for rows.Next() { ... }
return stats, rows.Err()
```

**`errors.Is()` over `==` on wrapped errors**
`fmt.Errorf("...: %w", err)` creates a new error value. `==` checks identity — always returns `false` on wrapped errors. Use `errors.Is()` to traverse the chain:
```go
if errors.Is(err, db.ErrDuplicate) { ... }
```

**No cgo — `lib/pq` is pure Go**
Unlike `go-sqlite3`, `lib/pq` has no C dependency. No `gcc`, no `CGO_ENABLED=1`, standard `go build` works.

**Docker volume persistence**
```bash
docker compose down      # keeps postgres_data volume — data survives
docker compose down -v   # deletes postgres_data volume — clean slate
```

---

*media-shelf · Technical Documentation · March 2026*
