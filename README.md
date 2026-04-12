# SPtraffic

전국 시외·고속버스 및 간선철도 시간표 조회 시스템

A timetable search system for nationwide intercity/express buses and KTX/rail in Korea, powered by the [data.go.kr](https://www.data.go.kr) public API.

---

## Features

- **Timetable search** — find direct and transfer routes between any two terminals/stations
- **Up to 2 transfers** — direct, 1-transfer, or 2-transfer journeys
- **Overnight routes** — correctly handles schedules crossing midnight
- **Interactive map** — OpenStreetMap with terminal markers and route polylines
- **Schedule popup** — click any terminal on the map to see upcoming departures
- **Transport modes** — express bus, intercity bus, KTX/rail, or mixed
- **Daily auto-refresh** — data ingested from data.go.kr every day at 02:00 KST

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.22, Gin, pgx/v5 |
| Database | PostgreSQL 16 |
| Cache | Redis 7 (5-min TTL) |
| Frontend | React 18, TypeScript, Vite, Leaflet |
| Infra | Docker Compose |

---

## Quick Start

### Prerequisites

- Docker and Docker Compose
- A [data.go.kr](https://www.data.go.kr) API key

### 1. Configure environment

```bash
cp .env.example .env
```

Edit `.env` and set at minimum:

```env
DATAGOKR_API_KEY=your_key_here
DB_PASSWORD=changeme
ADMIN_KEY=change-this-secret
```

### 2. Start the stack

```bash
make up
# or: docker compose up --build
```

Services:
- Frontend → http://localhost:3000
- Backend API → http://localhost:8080
- PostgreSQL → localhost:5432
- Redis → localhost:6379

### 3. Trigger initial data ingest

```bash
curl -X POST http://localhost:8080/api/admin/ingest \
  -H "X-Admin-Key: <your ADMIN_KEY>"
```

Data refreshes automatically every night at 02:00 KST after that.

---

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Service health (DB + Redis) |
| GET | `/api/terminals` | List terminals (`?q=name`, `?region=`, `?limit=`, `?offset=`) |
| GET | `/api/terminals/:code` | Single terminal by code |
| GET | `/api/terminals/:code/departures` | Upcoming departures from a terminal (`?after=HH:MM`, `?limit=`) |
| GET | `/api/search` | Route search (see params below) |
| GET | `/api/schedule/:route_id` | Full timetable for a route |
| GET | `/api/regions` | List of region codes |
| POST | `/api/admin/ingest` | Trigger full data refresh (requires `X-Admin-Key`) |
| GET | `/api/admin/status` | Last ingest time and error (requires `X-Admin-Key`) |

### Search parameters

| Param | Required | Example | Description |
|-------|----------|---------|-------------|
| `from` | Yes | `BUS001` | Departure terminal code |
| `to` | Yes | `BUS002` | Arrival terminal code |
| `date` | No | `2026-04-12T09:00` | Earliest departure (defaults to now) |
| `mode` | No | `all` | `all` / `bus` / `rail` |
| `max_legs` | No | `2` | `1` = direct only, `2` = 1 transfer, `3` = 2 transfers |

---

## Development

### Run infrastructure only

```bash
docker compose up db redis -d
```

### Run backend locally

```bash
cd backend
make tidy   # first time only
make run
```

### Backend commands

```bash
make run    # go run ./cmd/server
make build  # compile to ./server binary
make test   # go test ./...
make tidy   # go mod tidy
make lint   # golangci-lint run ./...
```

### Docker commands (from repo root)

```bash
make up            # build + start all services (dev)
make up-prod       # start without dev overrides
make down          # stop containers (keeps data)
make down-volumes  # stop containers + delete all data
make logs          # tail all logs
make shell-db      # psql into the database
make shell-backend # shell into the backend container
```

---

## Project Structure

```
SPtraffic/
├── backend/
│   ├── cmd/server/main.go          # Entry point
│   ├── internal/
│   │   ├── api/                    # Gin router, handlers, middleware
│   │   ├── cache/                  # Redis wrapper
│   │   ├── config/                 # Viper env config
│   │   ├── db/                     # Migration runner
│   │   ├── domain/                 # Pure Go types
│   │   ├── ingestion/              # data.go.kr client + parsers + cron scheduler
│   │   ├── repository/             # PostgreSQL repositories
│   │   └── service/                # Search and schedule business logic
│   └── migrations/                 # SQL migration files
├── frontend/
│   └── src/
│       ├── api/                    # API client
│       ├── components/             # React components (Map, Search, Schedule)
│       ├── hooks/                  # useTerminals, useSearch
│       ├── store/                  # Zustand global state
│       └── types/                  # TypeScript interfaces
├── docker-compose.yml
├── .env.example
└── Makefile
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATAGOKR_API_KEY` | — | data.go.kr API key; ingestion disabled if empty |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `sptraffic` | Database name |
| `DB_USER` | `sptraffic` | Database user |
| `DB_PASSWORD` | — | Database password |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `SERVER_PORT` | `8080` | HTTP listen port |
| `CORS_ORIGIN` | `http://localhost:5173` | Allowed CORS origin |
| `MIGRATIONS_DIR` | `./migrations` | Path to SQL migration files |
| `ADMIN_KEY` | — | Secret for `X-Admin-Key` header; empty = admin endpoints disabled |

---

## Data Sources

All data is fetched from [공공데이터포털 (data.go.kr)](https://www.data.go.kr):

| Service | Link | Dataset ID |
|---------|------|------------|
| 고속버스 운행정보 조회서비스 | [15098522](https://www.data.go.kr/data/15098522/openapi.do) | Express bus terminals & schedules |
| 시외버스 운행정보 조회서비스 | [15098541](https://www.data.go.kr/data/15098541/openapi.do) | Intercity bus terminals & schedules |
| 열차 운행정보 조회서비스 | [15098552](https://www.data.go.kr/data/15098552/openapi.do) | KTX/rail stations & schedules |

An API key from data.go.kr is required. Register at https://www.data.go.kr and apply for access to each service above.

---

## License

MIT
