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
- [gRPC Integration](#grpc-integration)
- [Data Flow](#data-flow)
- [External Dependencies](#external-dependencies)
- [Configuration Reference](#configuration-reference)
- [Critical Implementation Notes](#critical-implementation-notes)

---

## Project Overview

| Field                 | Value                                                                                                    |
| --------------------- | -------------------------------------------------------------------------------------------------------- |
| Language              | Go 1.26 · darwin/arm64 (Apple Silicon)                                                                   |
| Module                | `github.com/jyotil-raval/media-shelf`                                                                    |
| External dependencies | `lib/pq v1.10.9` · `cobra v1.10.2` · `godotenv v1.5.1` · `mal-updater v1.1.0` · `google.golang.org/grpc` |
| Database              | PostgreSQL 16 (via Docker)                                                                               |
| Status                | Phase 4 complete                                                                                         |

**Purpose:** Local CLI tool to track anime — fetches data from MAL via `mal-updater`'s gRPC `AnimeService`, stores entries in a local PostgreSQL database, and provides offline-capable list, stats, and export commands.

---

## Package Structure

```
media-shelf/
├── cmd/
│   ├── main.go                  # Entry point — env, db, gRPC client, cobra
│   └── shelf/
│       ├── app.go               # App struct + Add(), List(), Stats(), Export()
│       ├── root.go              # Cobra root command
│       ├── add.go               # shelf add
│       ├── list.go              # shelf list
│       ├── stats.go             # shelf stats
│       └── export.go            # shelf export
├── internal/
│   ├── config/
│   │   └── constants.go
│   ├── db/
│   │   ├── db.go                # Open() + ErrNotFound + ErrDuplicate
│   │   ├── filter.go            # Filter struct
│   │   ├── store.go             # Store interface
│   │   ├── postgres.go          # PostgreSQLStore implementation
│   │   └── db_test.go           # Table-driven tests
│   ├── models/
│   │   └── media.go             # Shared MediaItem struct
│   └── providers/
│       └── mal/
│           └── client.go        # gRPC client → mal-updater AnimeService
├── docs/
├── .env
├── .env.example
├── docker-compose.yml           # postgres:16-alpine
├── go.mod
└── go.sum
```

---

## Architecture

### Package Responsibilities

| Package                  | Key Files                                 | Responsibility                                      |
| ------------------------ | ----------------------------------------- | --------------------------------------------------- |
| `cmd/main.go`            | `main.go`                                 | Entry point · env · db · gRPC client · cobra wiring |
| `cmd/shelf/app.go`       | `app.go`                                  | App struct · store + malClient · command methods    |
| `cmd/shelf`              | `root, add, list, stats, export`          | Cobra commands — thin closures over App methods     |
| `internal/models`        | `media.go`                                | Shared `MediaItem` struct — imports nothing         |
| `internal/db`            | `db.go, filter.go, store.go, postgres.go` | Store interface · PostgreSQLStore · error types     |
| `internal/providers/mal` | `client.go`                               | gRPC client for `mal-updater` AnimeService          |

### App Struct — Two Dependencies

```go
type App struct {
    store     db.Store      // PostgreSQL via Store interface
    malClient *mal.Client   // gRPC client → mal-updater :9090
}
```

`store` is an interface — swappable for tests. `malClient` is a concrete gRPC client — connects to `mal-updater` on startup and reuses the connection for all calls.

### Dependency Graph

```
cmd/main.go
    │
    ├── internal/db              ← Store + PostgreSQLStore
    │       └── internal/models
    │
    ├── internal/providers/mal   ← gRPC client
    │       └── mal-updater/proto/animepb  ← imported from mal-updater@v1.1.0
    │
    └── cmd/shelf/*              ← Cobra commands
            ├── internal/db
            ├── internal/models
            └── internal/providers/mal
```

### Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│  External Services                                                   │
│                                                                      │
│  ┌───────────────────────────────┐   ┌──────────────────────────┐   │
│  │   mal-updater (Docker)        │   │   PostgreSQL (Docker)    │   │
│  │   gRPC AnimeService  :9090    │   │   :5432                  │   │
│  │   HTTP REST API      :8080    │   │   media_items table      │   │
│  └──────────────┬────────────────┘   └──────────────┬───────────┘   │
└─────────────────┼────────────────────────────────── ┼───────────────┘
                  │ protobuf (binary)                  │ SQL
                  ▼                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│   internal/providers/mal              internal/db                   │
│   mal.Client                          Store interface                │
│   → grpc.AnimeService.GetAnime()      PostgreSQLStore                │
│   → models.MediaItem                                                 │
└──────────────────────┬────────────────────┬────────────────────────┘
                       │                    │
                       ▼                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│   cmd/shelf/app.go — App{store, malClient}                           │
│   Add()   List()   Stats()   Export()                                │
└──────────────────────────────────────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│   cmd/main.go — Entry point                                          │
│   db.Open() → db.Migrate() → mal.NewClient() → NewApp() → Execute() │
└──────────────────────────────────────────────────────────────────────┘
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

- `RETURNING id` on insert
- `pq.Error` code `23505` → `ErrDuplicate`
- Dynamic WHERE with `$1`, `$2` numbered placeholders
- `rows.Close()` + `rows.Err()` on `List()`
- `var _ Store = (*PostgreSQLStore)(nil)` — compile-time assertion

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

### Dependency Injection via Closures

`RunE` captures `app` from the enclosing constructor function:

```go
func newAddCommand(app *App) *cobra.Command {
    var id, status, source string
    return &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            return app.Add(cmd.Context(), source, id, status) // app captured
        },
    }
}
```

No global state. Every command testable by injecting a mock `App`.

---

## gRPC Integration

### Why gRPC over HTTP for Service-to-Service

|                 | HTTP (REST)              | gRPC                               |
| --------------- | ------------------------ | ---------------------------------- |
| Format          | JSON — human readable    | Protobuf — binary, smaller         |
| Contract        | Implicit                 | Explicit `.proto` file             |
| Code generation | Manual                   | `protoc` generates client + server |
| Best for        | User-facing, curl, Bruno | Service-to-service                 |

### Proto Contract

`media-shelf` imports `mal-updater@v1.1.0` to get the generated proto code:

```go
import pb "github.com/jyotil-raval/mal-updater/proto/animepb"
```

The contract is defined in `mal-updater/proto/anime.proto`:

```protobuf
service AnimeService {
    rpc GetAnime(GetAnimeRequest) returns (AnimeResponse);
    rpc Search(SearchAnimeRequest) returns (SearchAnimeResponse);
    rpc GetList(GetListRequest) returns (GetListResponse);
}
```

### Client Implementation

```go
type Client struct {
    conn  *grpc.ClientConn
    anime pb.AnimeServiceClient
}

func NewClient(target string) (*Client, error) {
    conn, err := grpc.NewClient(target,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    // ...
    return &Client{
        conn:  conn,
        anime: pb.NewAnimeServiceClient(conn),
    }, nil
}

func (c *Client) GetAnime(ctx context.Context, id string) (*models.MediaItem, error) {
    resp, err := c.anime.GetAnime(ctx, &pb.GetAnimeRequest{Id: id})
    // map AnimeResponse → models.MediaItem
}
```

**`insecure.NewCredentials()`** — plaintext TCP for local development. In production use TLS credentials.

**Connection reuse** — the gRPC connection is established once at startup in `cmd/main.go` and shared for all calls. HTTP/2 multiplexing means multiple RPC calls share the same TCP connection efficiently.

**`defer malClient.Close()`** — always close the gRPC connection on exit to release the TCP socket.

---

## Data Flow

```
┌─────────────────────────────────────────────────────┐
│  1 · CLI Start                                       │
│  go run cmd/main.go add --id 1535 --status watching  │
│  Cobra parses flags → App.Add(ctx, source, id, ...)  │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│  2 · Fetch from MAL via gRPC                         │
│                                                      │
│  malClient.GetAnime(ctx, "1535")                     │
│  → AnimeService.GetAnime on mal-updater:9090         │
│  ← AnimeResponse (protobuf binary)                   │
│  → mapped to models.MediaItem                        │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│  3 · Store (internal/db)                             │
│                                                      │
│  item.Status = "watching"                            │
│  store.Add(ctx, item)                                │
│  INSERT INTO media_items ... RETURNING id            │
│                                                      │
│  pq 23505? → ErrDuplicate → readable message         │
│  Success?  → ✓ Added [1]: Death Note (tv)            │
└──────────────────────────────────────────────────────┘
```

---

## External Dependencies

### mal-updater gRPC

`localhost:9090` (configurable via `MAL_UPDATER_GRPC_URL`)

| RPC        | Request                | Purpose                  |
| ---------- | ---------------------- | ------------------------ |
| `GetAnime` | `{id: "1535"}`         | Fetch full anime details |
| `Search`   | `{q: "naruto"}`        | Search anime             |
| `GetList`  | `{status: "watching"}` | Get user's MAL list      |

### PostgreSQL

`postgres:16-alpine` via Docker Compose on port `5432`.

---

## Configuration Reference

| Variable               | Purpose                                                       |
| ---------------------- | ------------------------------------------------------------- |
| `DATABASE_URL`         | PostgreSQL connection string                                  |
| `MAL_UPDATER_GRPC_URL` | gRPC target address (default: `localhost:9090`)               |
| `MAL_UPDATER_URL`      | HTTP base URL (kept for future use)                           |
| `MAL_UPDATER_TOKEN`    | JWT token (not used by gRPC client — no auth interceptor yet) |

---

---

## Testing

### Philosophy

`media-shelf` uses a `MockStore` — an in-memory implementation of the `Store` interface — for all tests. This means:

- No Docker required to run tests
- No real PostgreSQL connection
- Tests run in milliseconds (~0.5s for 11 tests)
- Each test gets a fresh store — no shared state between tests

This is the Store interface payoff: `PostgreSQLStore` and `MockStore` satisfy the same interface. The application code never knows which one it's talking to.

### MockStore

```go
type MockStore struct {
    mu    sync.Mutex
    items map[int64]models.MediaItem
    next  int64
}
```

An in-memory map protected by a mutex. Implements all `Store` methods — `Add`, `GetByID`, `List`, `Update`, `Delete`, `Stats`. Duplicate detection mirrors the real DB constraint: same `source` + `source_id` = `ErrDuplicate`.

### Table-Driven Test Pattern

```go
tests := []struct {
    name    string
    item    models.MediaItem
    wantErr error
}{
    {"valid anime",           validItem,     nil},
    {"duplicate entry",       duplicateItem, ErrDuplicate},
    {"different source ID",   differentItem, nil},
}

store := NewMockStore()
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        _, err := store.Add(ctx, tt.item)
        if err != tt.wantErr {
            t.Errorf("Add() error = %v, wantErr = %v", err, tt.wantErr)
        }
    })
}
```

Adding a new test case = adding one struct to the slice. No new function, no duplicated setup.

`t.Run` creates a named sub-test — run a single case with:

```bash
go test ./internal/db/... -run TestMockStore_Add/duplicate_entry
```

### `t.Fatal` vs `t.Error`

| Function   | Behaviour                       | Use when                                             |
| ---------- | ------------------------------- | ---------------------------------------------------- |
| `t.Errorf` | Marks failed, continues         | Subsequent assertions are still meaningful           |
| `t.Fatalf` | Marks failed, stops immediately | Continuing would panic or produce meaningless output |

```go
items, err := store.List(ctx, tt.filter)
if err != nil {
    t.Fatalf("List() unexpected error: %v", err) // stop — can't check len(nil)
}
if len(items) != tt.wantCount {
    t.Errorf("List() got %d, want %d", len(items), tt.wantCount) // continue
}
```

### Run Tests

```bash
go test ./internal/...        # all packages
go test ./internal/db/... -v  # verbose — shows each sub-test
go test ./internal/db/... -run TestMockStore_List  # single test
```

---

## Critical Implementation Notes

**PostgreSQL `$1` placeholder syntax**

```sql
WHERE id = $1  -- correct (PostgreSQL)
WHERE id = ?   -- wrong (SQLite syntax)
```

**`db.Ping()` forces connection at startup**
`sql.Open()` never connects. `db.Ping()` validates the connection immediately.

**gRPC connection lifecycle**

```go
malClient, err := mal.NewClient(grpcTarget)  // establish once
defer malClient.Close()                       // release on exit
```

**gRPC has no auth interceptor yet**
The gRPC server accepts all connections without authentication. Production would use gRPC interceptors (equivalent of HTTP middleware) to validate tokens.

**`float64` from proto numeric fields**
`AnimeResponse.NumEpisodes` is `int32` in proto — cast directly:

```go
Total: int(resp.NumEpisodes)
```

**Import path for proto code**
`media-shelf` imports the generated proto from `mal-updater` as a versioned module dependency — not a local copy. Upgrading `mal-updater` requires a `go get` bump.

**`rows.Close()` and `rows.Err()`**

```go
defer rows.Close()
for rows.Next() { ... }
return items, rows.Err()
```

**`errors.Is()` over `==`**

```go
if errors.Is(err, db.ErrDuplicate) { ... }
```

---

_media-shelf · Technical Documentation · March 2026_

---

## Stats + Export

### Stats Query

```go
SELECT COALESCE(sub_type, 'unknown'), status, COUNT(*)
FROM media_items
GROUP BY sub_type, status
ORDER BY sub_type, status
```

`COALESCE` handles `NULL` sub_type values — returns `'unknown'` instead of `NULL` so the output is always readable.

`StatRow` holds each aggregated result:

```go
type StatRow struct {
    SubType string
    Status  string
    Count   int
}
```

### Export — `io.Writer` Pattern

Both `json.NewEncoder` and `csv.NewWriter` accept any `io.Writer`. `*os.File` implements `io.Writer`. This means one file handle works for both formats — no duplication:

```go
file, _ := os.Create(output)
defer file.Close()

switch format {
case "json":
    enc := json.NewEncoder(file)
    enc.SetIndent("", "  ")
    enc.Encode(items)
case "csv":
    w := csv.NewWriter(file)
    defer w.Flush()
    w.Write([]string{"id", "title", ...}) // header
    for _, item := range items {
        w.Write([]string{...})
    }
}
```

`csv.Writer.Flush()` must be called — it buffers writes internally and only flushes to the underlying `io.Writer` on `Flush()`. Without it, the file may be incomplete.
