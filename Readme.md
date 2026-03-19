# media-shelf

A local CLI tool to track your anime — with data pulled from MAL, stored in PostgreSQL, and queried via subcommands.

---

## What It Does

`media-shelf` is a CLI tool that lets you add, list, filter, and export your anime watchlist locally. It fetches anime data from MyAnimeList via `mal-updater` (Project 1), stores everything in a local PostgreSQL database, and provides fast, offline-capable queries through a clean command interface.

---

## How It Works

1. You run `shelf add --source mal --id 1535 --status watching`
2. `media-shelf` calls `mal-updater`'s HTTP API to fetch full anime details
3. The entry is stored in your local PostgreSQL database
4. `shelf list`, `shelf stats`, and `shelf export` query your local database — no network required

---

## Prerequisites

- Go 1.26 or higher
- Docker Desktop — for PostgreSQL
- A running `mal-updater` instance — for fetching MAL data
  - Repo: [github.com/jyotil-raval/mal-updater](https://github.com/jyotil-raval/mal-updater)

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

Open `.env` and fill in your credentials:

```env
DATABASE_URL=postgres://shelf:shelf@localhost:5432/mediashelf?sslmode=disable
MAL_UPDATER_URL=http://localhost:8080
MAL_UPDATER_TOKEN=<jwt from mal-updater POST /auth/token>
```

### Start PostgreSQL

```bash
docker compose up -d
docker compose ps   # wait for healthy
```

### Run

```bash
go run cmd/main.go --help
```

---

## Usage

```bash
# Add anime from MAL
shelf add --source mal --id 1535 --status watching
shelf add --source mal --id 16498 --status completed

# List your shelf
shelf list
shelf list --type tv --status watching
shelf list --subtype movie
shelf list --status completed --score 8
shelf list --sort title

# Stats
shelf stats

# Export
shelf export --format json --output shelf.json
shelf export --format csv  --output shelf.csv
```

---

## Project Structure

```
media-shelf/
├── cmd/
│   ├── main.go              ← entry point — wires everything
│   └── shelf/
│       ├── app.go           ← App struct + all command methods
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
│           └── client.go    ← calls mal-updater HTTP API
├── docs/                    ← technical documentation
├── .env                     ← gitignored
├── .env.example
├── docker-compose.yml       ← PostgreSQL
├── go.mod
└── go.sum
```

---

## Tech

- Go standard library — `database/sql`, `encoding/json`, `encoding/csv`, `net/http`
- [`lib/pq`](https://github.com/lib/pq) — PostgreSQL driver (pure Go, no cgo)
- [`cobra`](https://github.com/spf13/cobra) — CLI framework with subcommands
- [`godotenv`](https://github.com/joho/godotenv) — `.env` file loading
- [Docker](https://www.docker.com) — PostgreSQL via `postgres:16-alpine`

---

## Phase Progress

| Phase | Description                                                          | Status |
| ----- | -------------------------------------------------------------------- | ------ |
| 1     | Project skeleton + PostgreSQL schema + Docker compose                | ✅     |
| 2     | Database layer — `Store` interface + `PostgreSQLStore` + error types | ✅     |
| 3     | Cobra CLI — `App` struct + all subcommands wired                     | ✅     |
| 4     | MAL provider — calls `mal-updater` HTTP API                          | 🔜     |
| 5     | Stats + Export — JSON and CSV                                        | 🔜     |
| 6     | Table-driven tests                                                   | 🔜     |

---

## Related Projects

- [mal-updater](https://github.com/jyotil-raval/mal-updater) — Project 1 · MAL sync CLI + JWT-protected HTTP API

---

## License

[MIT](./LICENSE)
