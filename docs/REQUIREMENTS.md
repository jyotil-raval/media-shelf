# media-shelf — Requirements

> Project 2 of the Go Learning Arc · Anime tracking CLI backed by PostgreSQL

---

## Overview

A local CLI tool to track anime — with data pulled from MAL via `mal-updater`, stored in a local PostgreSQL database, and queried via Cobra subcommands.

**Reference language:** TypeScript / Node.js
**Prior project:** [mal-updater](https://github.com/jyotil-raval/mal-updater) — OAuth2, HTTP client, JSON I/O, JWT API

---

## Scope

**In scope:**

- Anime only — no movies, books, or series (current phase)
- Single data source — MAL via `mal-updater` HTTP API
- Local PostgreSQL database via Docker
- CLI interface only — no HTTP server in this project

**Out of scope (current phase):**

- OMDb, OpenLibrary integrations
- Fan-out concurrency (dropped — only one provider for now)
- HTTP server / REST API (Project 3)
- gRPC (Project 3)

---

## Future Scope

The current implementation is intentionally scoped to anime only. The architecture
is designed to accommodate expansion without structural changes.

### Planned Media Type Extensions

| Type   | Source          | Auth    | Notes                        |
| ------ | --------------- | ------- | ---------------------------- |
| Movies | OMDb API        | API key | Register free at omdbapi.com |
| Series | OMDb API        | API key | Same key as movies           |
| Books  | OpenLibrary API | None    | No auth required             |

### What Expansion Requires

- Add a `Provider` interface in `internal/providers/provider.go`
- Add one provider package per source — `internal/providers/omdb/`, `internal/providers/openlibrary/`
- Add `media_type` values — `movie`, `series`, `book` to the application layer
- Fan-out concurrency — fetch from multiple providers simultaneously

### What Does NOT Change on Expansion

- `MediaItem` struct — already has `MediaType` and `SubType` fields
- `Store` interface — already handles any `MediaItem`
- Database schema — Single Table Inheritance supports all media types without migration
- Cobra commands — `shelf list`, `shelf stats`, `shelf export` work unchanged

The `SubType` field already handles cross-type distinctions:

- Anime film → `media_type: anime`, `sub_type: movie`
- Standalone film → `media_type: movie`, `sub_type: ""`

### Concurrency Pattern (Fan-Out / Fan-In)

When multiple providers are active, concurrent fetching replaces sequential calls.
This is the Go equivalent of `Promise.all()`:

```go
resultChan := make(chan result, len(providers)) // buffered — goroutines don't block

for _, p := range providers {
    go func(provider Provider) {         // fan-out — one goroutine per provider
        item, err := provider.GetByID(ctx, id)
        resultChan <- result{item, err}
    }(p)
}

for i := 0; i < len(providers); i++ {
    r := <-resultChan                    // fan-in — collect all results
}
```

Total wait time = `max(all providers)` instead of `sum(all providers)`.

---

## Data Source (Current)

| Source                      | Media | Auth Mechanism                                |
| --------------------------- | ----- | --------------------------------------------- |
| MAL API (via `mal-updater`) | Anime | JWT token from `mal-updater POST /auth/token` |

`media-shelf` never calls the MAL API directly. All MAL data flows through
`mal-updater`'s JWT-protected HTTP API running on `:8080`.

---

## CLI Interface

```bash
# Add anime
shelf add --source mal --id 1535 --status watching
shelf add --source mal --id 16498 --status completed

# List
shelf list
shelf list --type tv --status watching
shelf list --subtype movie
shelf list --status completed --score 8

# Stats
shelf stats

# Export
shelf export --format json --output shelf.json
shelf export --format csv  --output shelf.csv
```

---

## Data Model

### Go Struct

```go
// internal/models/media.go
type MediaItem struct {
    ID        int64  `json:"id"         db:"id"`
    Title     string `json:"title"      db:"title"`
    MediaType string `json:"media_type" db:"media_type"` // always "anime" (current phase)
    SubType   string `json:"sub_type"   db:"sub_type"`   // tv | movie | ova | special
    Source    string `json:"source"     db:"source"`     // always "mal" (current phase)
    SourceID  string `json:"source_id"  db:"source_id"`
    Status    string `json:"status"     db:"status"`     // watching | completed | on_hold | dropped | plan_to
    Score     int    `json:"score"      db:"score"`      // 1–10, nullable
    Progress  int    `json:"progress"   db:"progress"`   // episodes watched
    Total     int    `json:"total"      db:"total"`      // total episodes
    Notes     string `json:"notes"      db:"notes"`
}
```

### Database Schema

```sql
CREATE TABLE IF NOT EXISTS media_items (
    id          SERIAL       PRIMARY KEY,
    title       TEXT         NOT NULL,
    media_type  TEXT         NOT NULL,   -- anime | movie | series | book (future)
    sub_type    TEXT,                    -- tv | movie | ova | special | ona
    source      TEXT         NOT NULL,   -- mal | omdb | openlibrary (future)
    source_id   TEXT,
    status      TEXT         NOT NULL,   -- watching | completed | on_hold | dropped | plan_to
    score       INTEGER,
    progress    INTEGER,
    total       INTEGER,
    notes       TEXT,
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_source ON media_items(source, source_id);
```

### Design Rationale

**Single Table Inheritance** — all media in one table regardless of type. `WHERE status = 'watching'` works across the full shelf without JOINs or UNIONs. Adding movies and books requires zero schema changes.

**`SubType` field** — handles anime movies, OVAs, and specials. Also handles future media subtypes without additional columns.

**UNIQUE INDEX on `(source, source_id)`** — deduplication enforced at the DB level. The application catches the constraint violation and surfaces a clean error.

**`SERIAL` over `INTEGER PRIMARY KEY AUTOINCREMENT`** — PostgreSQL's sequence-backed auto-increment. More robust than SQLite's `AUTOINCREMENT` for concurrent writes.

**`TIMESTAMPTZ` over `DATETIME`** — timezone-aware. Stored as UTC, converted on read.

---

## Project Structure

```
media-shelf/
├── cmd/
│   ├── main.go              ← entry point — wires everything, minimal logic
│   └── shelf/
│       ├── add.go           ← shelf add
│       ├── list.go          ← shelf list
│       ├── stats.go         ← shelf stats
│       └── export.go        ← shelf export
├── internal/
│   ├── config/
│   │   └── constants.go     ← all global constants
│   ├── db/
│   │   ├── db.go            ← Store interface + PostgreSQLStore + error types
│   │   ├── migrations.go    ← schema creation
│   │   └── db_test.go       ← table-driven tests
│   ├── models/
│   │   └── media.go         ← shared MediaItem struct
│   └── providers/
│       └── mal/
│           └── client.go    ← HTTP client for mal-updater API
├── docs/
│   ├── REQUIREMENTS.md      ← this file
│   └── TECHNICAL.md         ← architecture + implementation notes
├── .env                     ← gitignored
├── .env.example
├── docker-compose.yml       ← postgres:16-alpine
├── go.mod
└── go.sum
```

---

## Store Interface

```go
type Store interface {
    Add(ctx context.Context, item models.MediaItem) (int64, error)
    GetByID(ctx context.Context, id int64) (*models.MediaItem, error)
    List(ctx context.Context, filter Filter) ([]models.MediaItem, error)
    Update(ctx context.Context, item models.MediaItem) error
    Delete(ctx context.Context, id int64) error
}

type Filter struct {
    Status    string
    MediaType string
    SubType   string
    MinScore  int
    Sort      string
}
```

`PostgreSQLStore` implements `Store` for production.
Tests use an in-memory mock — no disk, no network, runs in microseconds.

---

## App Struct — Cobra Dependency Injection

```go
type App struct {
    store     db.Store
    malClient *mal.Client
}

func (a *App) Add(ctx context.Context, source, id, status string) error { ... }
func (a *App) List(ctx context.Context, filter db.Filter) error { ... }
func (a *App) Stats(ctx context.Context) error { ... }
func (a *App) Export(ctx context.Context, format, output string) error { ... }
```

Cobra `RunE` is a one-line wrapper:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    return app.Add(cmd.Context(), source, id, status)
}
```

No global state. Every command is independently testable.

---

## Environment Variables

```env
DATABASE_URL=postgres://shelf:shelf@localhost:5432/mediashelf?sslmode=disable
MAL_UPDATER_URL=http://localhost:8080
MAL_UPDATER_TOKEN=<jwt from mal-updater POST /auth/token>
```

---

## Dependencies

```bash
go get github.com/lib/pq              # PostgreSQL driver (pure Go, no cgo)
go get github.com/spf13/cobra         # CLI framework
go get github.com/joho/godotenv       # .env loading
```

No GORM. `database/sql` only — learn what SQL is actually being executed.

---

## Docker

PostgreSQL runs via Docker Compose from day one — no local PostgreSQL installation required.

```bash
docker compose up -d    # start PostgreSQL
docker compose ps       # verify healthy
docker compose down     # stop — data preserved in named volume
docker compose down -v  # stop + wipe data — clean slate
```

---

## Phase Plan

| Phase | What                                                                                     | Testable Artifact                                              |
| ----- | ---------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| **1** | Project skeleton + PostgreSQL schema + Docker compose                                    | `go build ./...` · `media-shelf ready.` · schema in PostgreSQL |
| **2** | Database layer — `Store` interface + `PostgreSQLStore` + `context.Context` + error types | `store.Add()` + `store.List()` work against real DB            |
| **3** | Cobra CLI — `App` struct + all subcommands wired                                         | `shelf --help` shows all subcommands                           |
| **4** | MAL provider — calls `mal-updater` HTTP API                                              | `shelf add --source mal --id 1535` fetches + stores            |
| **5** | Stats + Export — JSON and CSV                                                            | `shelf stats` prints table · `shelf export` writes valid file  |
| **6** | Table-driven tests — `Store` Add, List, duplicate handling                               | `go test ./...` passes                                         |

---

## Go Concepts This Project Teaches

| Concept                                              | Phase |
| ---------------------------------------------------- | ----- |
| `database/sql` — blank import, driver registry       | 2     |
| `context.Context` — `WithTimeout`, `cancel()`        | 2     |
| PostgreSQL `$1` parameter syntax                     | 2     |
| `rows.Close()` and `rows.Err()`                      | 2     |
| Interfaces as contracts — `Store`                    | 2     |
| Pointer vs value receivers                           | 2     |
| Cobra — `RunE`, flags, `init()`                      | 3     |
| Dependency injection via `App` struct                | 3     |
| HTTP client — consuming `mal-updater` as a service   | 4     |
| `encoding/csv`                                       | 5     |
| `errors.Is` vs `==` on wrapped errors                | 5     |
| Table-driven tests — `t.Run`, `t.Error` vs `t.Fatal` | 6     |

---

## Relationship to mal-updater

```
mal-updater (Project 1)             media-shelf (Project 2)
────────────────────────            ──────────────────────────
OAuth2 + PKCE                       Consumes mal-updater API
MAL sync CLI                        Cobra CLI
JWT HTTP API (:8080)                PostgreSQL storage
Docker image                        Docker Compose (DB only)
v1.0.0 tagged + released            Imports mal-updater public packages
```

---

_media-shelf · Requirements · March 2026_
