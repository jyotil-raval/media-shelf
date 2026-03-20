# media-shelf

A local CLI tool to track your anime — with data pulled from MAL via gRPC, stored in PostgreSQL, and queried via subcommands.

---

## What It Does

`media-shelf` is a CLI tool that lets you add, list, filter, and export your anime watchlist locally. It fetches anime data from MyAnimeList via `mal-updater`'s gRPC API, stores everything in a local PostgreSQL database, and provides fast, offline-capable queries through a clean command interface.

---

## How It Works

1. You run `shelf add --source mal --id 1535 --status watching`
2. `media-shelf` calls `mal-updater`'s gRPC `AnimeService.GetAnime` to fetch full anime details
3. The entry is stored in your local PostgreSQL database
4. `shelf list`, `shelf stats`, and `shelf export` query your local database — no network required

---

## Prerequisites

- Go 1.26 or higher
- Docker Desktop — for PostgreSQL
- A running `mal-updater` instance — gRPC server on `:9090`
  - Repo: [github.com/jyotil-raval/mal-updater](https://github.com/jyotil-raval/mal-updater)
  - Run: `docker compose up -d` from `mal-updater/`

---

## Setup

### Install

```bash
git clone https://github.com/jyotil-raval/media-shelf.git
cd media-shelf
go mod tidy
```

### Configure

```bash
cp .env.example .env
```

Open `.env` and fill in:

```env
DATABASE_URL=postgres://shelf:shelf@localhost:5432/mediashelf?sslmode=disable
MAL_UPDATER_GRPC_URL=localhost:9090
MAL_UPDATER_URL=http://localhost:8080
MAL_UPDATER_TOKEN=
```

### Start PostgreSQL

```bash
docker compose up -d
docker compose ps   # wait for healthy
```

### Start mal-updater (required for add command)

```bash
cd ../mal-updater
docker compose up -d   # starts HTTP :8080 + gRPC :9090
```

### Run

```bash
go run cmd/main.go --help
```

---

## Usage

```bash
# Add anime from MAL (requires mal-updater gRPC on :9090)
go run cmd/main.go add --source mal --id 1535 --status watching
go run cmd/main.go add --source mal --id 16498 --status completed

# List your shelf (offline — queries local PostgreSQL)
go run cmd/main.go list
go run cmd/main.go list --status watching
go run cmd/main.go list --subtype movie
go run cmd/main.go list --status completed --score 8
go run cmd/main.go list --sort title

# Stats
go run cmd/main.go stats

# Export
go run cmd/main.go export --format json --output shelf.json
go run cmd/main.go export --format csv  --output shelf.csv
```

---

## Project Structure

```
media-shelf/
├── cmd/
│   ├── main.go              ← entry point — wires store + gRPC client + cobra
│   └── shelf/
│       ├── app.go           ← App struct + Add(), List(), Stats(), Export()
│       ├── root.go          ← Cobra root command
│       ├── add.go           ← shelf add
│       ├── list.go          ← shelf list
│       ├── stats.go         ← shelf stats
│       └── export.go        ← shelf export
├── internal/
│   ├── config/              ← constants
│   ├── db/
│   │   ├── db.go            ← Open() + sentinel errors
│   │   ├── filter.go        ← Filter struct
│   │   ├── store.go         ← Store interface
│   │   ├── postgres.go      ← PostgreSQLStore implementation
│   │   └── db_test.go       ← table-driven tests
│   ├── models/
│   │   └── media.go         ← shared MediaItem struct
│   └── providers/
│       └── mal/
│           └── client.go    ← gRPC client → mal-updater AnimeService
├── docs/
├── .env                     ← gitignored
├── .env.example
├── docker-compose.yml       ← PostgreSQL
├── go.mod
└── go.sum
```

---

## Tech

- Go standard library — `database/sql`, `encoding/json`, `encoding/csv`
- [`lib/pq`](https://github.com/lib/pq) — PostgreSQL driver (pure Go)
- [`cobra`](https://github.com/spf13/cobra) — CLI framework
- [`godotenv`](https://github.com/joho/godotenv) — `.env` loading
- [`google.golang.org/grpc`](https://pkg.go.dev/google.golang.org/grpc) — gRPC client
- [`mal-updater v1.1.0`](https://github.com/jyotil-raval/mal-updater) — proto contract + AnimeService
- [Docker](https://www.docker.com) — PostgreSQL via `postgres:16-alpine`

---

## Phase Progress

| Phase | Description                                                          | Status |
| ----- | -------------------------------------------------------------------- | ------ |
| 1     | Project skeleton + PostgreSQL schema + Docker compose                | ✅     |
| 2     | Database layer — `Store` interface + `PostgreSQLStore` + error types | ✅     |
| 3     | Cobra CLI — `App` struct + all subcommands wired                     | ✅     |
| 4     | MAL provider — gRPC client → `mal-updater` AnimeService              | ✅     |
| 5     | Stats + Export — JSON and CSV                                        | 🔜     |
| 6     | Table-driven tests                                                   | 🔜     |

---

## Related Projects

- [mal-updater](https://github.com/jyotil-raval/mal-updater) — Project 1 · MAL sync CLI + HTTP + gRPC API

---

## License

[MIT](./LICENSE)
