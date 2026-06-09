# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

전국 시외/고속버스 및 간선철도 시간표 조회 시스템 — a timetable query system for nationwide intercity/express buses and intercity rail in Korea.

## Tech Stack

- **Backend**: Go 1.22, Gin, pgx/v5 (PostgreSQL), go-redis/v9, robfig/cron
- **Database**: PostgreSQL (plain coordinates, no PostGIS)
- **Cache**: Redis (5-minute TTL on search results)
- **Frontend**: React + TypeScript + Vite + Tailwind CSS + Leaflet + Zustand
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
make lint        # golangci-lint run ./...

# Run a single test package
go test ./internal/service/...

# Frontend (from frontend/)
npm install
npm run dev      # Vite dev server on :5173
npm run build    # production build to dist/

# Full stack (dev — includes docker-compose.override.yml)
docker compose up --build

# Full stack (production-like — skips override)
docker compose -f docker-compose.yml up --build -d

# DB shell / backend shell
make shell-db       # psql -U sptraffic -d sptraffic
make shell-backend  # sh in the backend container
```

## Backend Architecture (`backend/`)

```
cmd/server/main.go          — entry point: load config, migrate DB, wire all layers
internal/config/            — viper env var loading (.env file + OS env)
internal/db/migrate.go      — SQL migration runner (schema_migrations tracking table)
internal/domain/            — pure Go types: Terminal, Route, Schedule, SearchQuery/Result, DepartureInfo
internal/repository/        — pgx PostgreSQL implementations behind interfaces
internal/service/
  search_service.go         — direct + 1-transfer + 2-transfer route search
  schedule_service.go       — per-route timetable retrieval
internal/cache/redis.go     — Redis wrapper; 5-min TTL; SearchKey() builds cache keys
internal/ingestion/
  client/datagokr.go        — rate-limited HTTP client (5 req/s, exponential backoff, 3 retries)
  parser/{express_bus,intercity_bus,rail,util}.go — JSON → domain types
  scheduler.go              — daily 02:00 KST cron: truncate all tables, then re-ingest
internal/api/
  router.go                 — Gin router; global rate limit 30 req/s per IP (burst 60)
  handler/{terminal,search,schedule,region,health,admin}.go
  middleware/{cors,ratelimit,adminkey}.go
migrations/001–003_*.sql    — applied by db.Migrate() on startup (tracked in schema_migrations)
```

## Frontend Architecture (`frontend/`)

```
src/main.tsx              — React entry point
src/App.tsx               — top-level layout: header + left panel (search/results) + map
src/store/index.ts        — Zustand global state: terminals, searchResults, selectedResult,
                            selectedTerminal, searching, searchError
src/api/client.ts         — typed fetch wrappers; VITE_API_BASE env var sets the API host
src/types/index.ts        — shared TS interfaces (Terminal, SearchResult, Leg, etc.)
                            + minsToTime() / minsToLabel() helpers
src/hooks/
  useSearch.ts            — calls api.search(), writes results to store
  useTerminals.ts         — loads terminal list for map markers
src/components/
  Search/{SearchForm,SearchResults,TerminalInput}.tsx
  Map/{MapView,TerminalMarker,RouteLine}.tsx
  Schedule/SchedulePopup.tsx
```

The frontend talks to the backend via the same origin (Vite dev proxy / nginx in production). `VITE_API_BASE` can override the base URL.

## API Endpoints

| Method | Path | Notes |
|--------|------|-------|
| GET | `/health` | Pings DB + Redis; 503 if either is down |
| GET | `/api/terminals` | `?q=name` autocomplete (≤20); `?region=` filter; plain list with `?limit=&offset=` |
| GET | `/api/terminals/:code` | Single terminal by code |
| GET | `/api/terminals/:code/departures` | `?after=HH:MM` (default: now KST); `?limit=` (default 20, max 60) |
| GET | `/api/search` | `?from=CODE&to=CODE&date=YYYY-MM-DDTHH:MM&mode=all&max_legs=1\|2\|3` |
| GET | `/api/schedule/:route_id` | Full timetable for a route |
| GET | `/api/regions` | Distinct region codes |
| POST | `/api/admin/ingest` | Requires `X-Admin-Key` header; triggers full refresh |
| GET | `/api/admin/status` | Requires `X-Admin-Key`; shows last run time + error |

## Key Design Decisions

- **Schedule times** are stored as `departure_mins` / `arrival_mins` (integer minutes since midnight, 0–1439) — not `TIME` — to avoid timezone complexity. All time math wraps with `+24*60` for overnight routes.
- **`days_of_week` bitmask**: bit 0 = Monday … bit 6 = Sunday; 127 = every day. `Schedule.RunsOn(weekday)` checks this bitmask.
- **Daily full refresh**: ingestion truncates schedules → routes → terminals, then re-inserts. Routes without an API-provided code are inserted without `ON CONFLICT` handling.
- **Transfer search**: `max_legs=2` → `oneTransfer` (indexes leg-1 by dest, intersects leg-2 origins); `max_legs=3` → `twoTransfer` (enumerates mid1 from leg-1, mid2 from leg-3, finds connecting leg-2). Min transfer wait = 30 min; max results = 20.
- **Admin key** (`ADMIN_KEY` env var): if empty, admin endpoints return 403 Forbidden.
- **`docker-compose.override.yml`**: automatically merged in `docker compose up` for local dev (exposes DB/Redis to host, sets `GIN_MODE=debug`). For production-like runs use `-f docker-compose.yml` explicitly.
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
| `VITE_API_BASE` | (same origin) | Frontend: override API host (e.g. `http://localhost:8080`) |
