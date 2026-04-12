# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Instructions for CLAUDE
* If you trying to reptivie tasks or things to explore, do not use inlcuded serve-agent, use "opencode_run". Provider is github-copilot, and Model is GPT-4o.
* If you need debug or build, do not build in this computer, user remote server. Remote server address is 192.168.117.128, user name is "etx", password is literal "null".

## Project Overview

전국 시외/고속버스 및 간선철도 시간표 조회 시스템 — a timetable query system for nationwide intercity/express buses and intercity rail in Korea.

## Tech Stack

- **Backend**: Go 1.22, Gin, pgx/v5 (PostgreSQL), go-redis/v9, robfig/cron
- **Database**: PostgreSQL (plain coordinates, no PostGIS)
- **Cache**: Redis (5-minute TTL on search results)
- **Frontend**: React + TypeScript + Leaflet (not yet built)
- **Infra**: Docker Compose (`docker-compose.yml` at repo root)

## Common Commands

```bash
# Start infrastructure only (DB + Redis)
docker compose up db redis -d

# Run backend (from backend/)
make run         # go run ./cmd/server
make build       # compile to ./server
make tidy        # go mod tidy
make test        # go test ./...

# Full stack
docker compose up --build
```

## Backend Architecture (`backend/`)

```
cmd/server/main.go          — entry point: load config, migrate DB, wire all layers
internal/config/            — viper env var loading (.env file + OS env)
internal/db/migrate.go      — SQL migration runner (schema_migrations tracking table)
internal/domain/            — pure Go types: Terminal, Route, Schedule, SearchQuery/Result
internal/repository/        — pgx PostgreSQL implementations behind interfaces
internal/service/
  search_service.go         — direct + 1-transfer route search (BFS-style)
  schedule_service.go       — per-route timetable retrieval
internal/cache/redis.go     — Redis wrapper; 5-min TTL; SearchKey() builds cache keys
internal/ingestion/
  client/datagokr.go        — rate-limited HTTP client (5 req/s, exponential backoff, 3 retries)
  parser/{express_bus,intercity_bus,rail,util}.go — JSON → domain types
  scheduler.go              — daily 02:00 KST cron: truncate all tables, then re-ingest
internal/api/
  router.go                 — Gin router + middleware wiring
  handler/{terminal,search,schedule,region,health,admin}.go
  middleware/{cors,ratelimit,adminkey}.go
migrations/001–003_*.sql    — applied by db.Migrate() on startup (tracked in schema_migrations)
```

## API Endpoints

| Method | Path | Notes |
|--------|------|-------|
| GET | `/health` | Pings DB + Redis; 503 if either is down |
| GET | `/api/terminals` | `?q=name` for autocomplete, `?region=` to filter |
| GET | `/api/terminals/:code` | Single terminal by code |
| GET | `/api/search` | `?from=CODE&to=CODE&date=YYYY-MM-DDTHH:MM&mode=all&max_legs=2` |
| GET | `/api/schedule/:route_id` | Full timetable for a route |
| GET | `/api/regions` | Distinct region codes |
| POST | `/api/admin/ingest` | Requires `X-Admin-Key` header; triggers full refresh |
| GET | `/api/admin/status` | Requires `X-Admin-Key`; shows last run time + error |

## Key Design Decisions

- **Schedule times** are stored as `departure_mins` / `arrival_mins` (integer minutes since midnight) — not `TIME` — to avoid timezone complexity.
- **Daily full refresh**: ingestion truncates schedules → routes → terminals, then re-inserts. Routes without an API-provided code are inserted without ON CONFLICT handling.
- **Transfer search** (`oneTransfer`): indexes leg-1 routes by destination ID, intersects with leg-2 origins. Min transfer wait = 30 min; max results returned = 20.
- **Admin key** (`ADMIN_KEY` env var): if empty, admin endpoints return 403 Forbidden.
- **data.go.kr parsers** use confirmed field names from the official spec (verified 2026-04-12). See dataset links in README § Data Sources.

## Environment Variables

Copy `.env.example` → `.env`. Viper reads `.env` automatically on startup (silently ignored in Docker where env vars are injected directly).

| Variable | Default | Description |
|----------|---------|-------------|
| `DATAGOKR_API_KEY` | — | data.go.kr key; ingestion disabled if empty |
| `DB_HOST/PORT/NAME/USER/PASSWORD` | localhost/5432/sptraffic/sptraffic/— | PostgreSQL |
| `REDIS_ADDR` | localhost:6379 | Redis |
| `SERVER_PORT` | 8080 | HTTP listen port |
| `CORS_ORIGIN` | http://localhost:5173 | Allowed CORS origin |
| `MIGRATIONS_DIR` | ./migrations | Path to SQL migration files |
| `ADMIN_KEY` | — | Secret for `X-Admin-Key` header; empty = disabled |
