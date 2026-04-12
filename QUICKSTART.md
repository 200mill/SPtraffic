# SPtraffic Quickstart Guide

This guide walks you through getting the system running from scratch, including obtaining the required API keys.

---

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (includes Docker Compose)
- A [data.go.kr](https://www.data.go.kr) account (free, takes ~5 minutes)

---

## Step 1 — Get a data.go.kr API Key

SPtraffic pulls its timetable data from Korea's public data portal. You need to register and apply for access to three APIs.

### 1-1. Create an account

1. Go to **https://www.data.go.kr**
2. Click **회원가입** (Sign up) in the top-right corner
3. Complete registration with a Korean phone number or email

### 1-2. Apply for API access

You need access to **three services**. Apply for each one:

| Service name | Link | 제공기관 |
|---|---|---|
| 고속버스 운행정보 조회서비스 | [15098522](https://www.data.go.kr/data/15098522/openapi.do) | 국토교통부 |
| 시외버스 운행정보 조회서비스 | [15098541](https://www.data.go.kr/data/15098541/openapi.do) | 국토교통부 |
| 열차 운행정보 조회서비스 | [15098552](https://www.data.go.kr/data/15098552/openapi.do) | 국토교통부 |

For each service:
1. Search the service name on data.go.kr
2. Open the service detail page
3. Click **활용신청** (Apply for use)
4. Fill in purpose of use (e.g. "개인 프로젝트 — 시간표 조회 앱 개발")
5. Submit — approval is usually instant or within a few minutes

### 1-3. Get your key

1. Go to **마이페이지 → 개발계정** (My Page → Developer Account)
2. Find one of the services you applied for
3. Copy the **일반 인증키 (Decoding)** — this is your `DATAGOKR_API_KEY`

> **Note:** All three services share the same API key. You only need to copy it once.

---

## Step 2 — Clone and Configure

```bash
git clone https://github.com/yourname/sptraffic.git
cd sptraffic
```

Copy the example environment file:

```bash
cp .env.example .env
```

Open `.env` and fill in the required values:

```env
# Paste your data.go.kr key here
DATAGOKR_API_KEY=your_actual_key_here

# Set a secure database password
DB_PASSWORD=a-strong-password

# Set a secret for the admin endpoints
ADMIN_KEY=a-random-secret-string
```

Everything else can stay as the default for local development.

---

## Step 3 — Start the Stack

```bash
bash quickstart.sh
```

This script will:
- Check Docker is installed
- Build the backend (Go) and frontend (React) images
- Start PostgreSQL, Redis, backend, and frontend
- Wait for the backend to become healthy
- Trigger the first data ingest automatically (if your API key is set)

Or start manually:

```bash
docker compose up --build
```

Services come up at:

| Service | URL |
|---|---|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080/api |
| Health check | http://localhost:8080/health |

---

## Step 4 — Load Timetable Data

If `quickstart.sh` didn't trigger the ingest (e.g. you set the API key after starting), run it manually:

```bash
curl -X POST http://localhost:8080/api/admin/ingest \
  -H "X-Admin-Key: your-ADMIN_KEY-value"
```

Expected response: `{"message":"ingest started"}`

The ingest runs in the background and takes a few minutes depending on API response times. Check progress with:

```bash
curl http://localhost:8080/api/admin/status \
  -H "X-Admin-Key: your-ADMIN_KEY-value"
```

After the ingest completes, **data refreshes automatically every day at 02:00 KST**.

---

## Step 5 — Verify It Works

Check the health endpoint:

```bash
curl http://localhost:8080/health
# {"db":"ok","redis":"ok"}
```

Check that terminals were loaded:

```bash
curl "http://localhost:8080/api/terminals?q=서울"
```

You should see a list of terminals with "서울" in their name.

---

## Stopping and Restarting

```bash
# Stop containers (data is preserved)
docker compose down

# Stop and delete all data
docker compose down -v

# Restart
docker compose up -d
```

---

## Troubleshooting

### Backend exits immediately on startup

Check the logs:

```bash
docker compose logs backend
```

Common causes:
- **`database ping failed`** — DB is still starting up. Wait a few seconds and retry.
- **`migrations failed`** — Check the DB password in `.env` matches.

### Ingest returns 403 Forbidden

The `X-Admin-Key` header doesn't match `ADMIN_KEY` in `.env`. If `ADMIN_KEY` is empty, the admin endpoints are disabled entirely.

### Ingest returns 409 Conflict

An ingest is already running. Wait for it to finish, then check `/api/admin/status`.

### No terminals found after ingest

The data.go.kr parsers use field names based on the documented API spec. If the live API returns different field names, the parser will silently skip records. Check the backend logs for `[ingestion]` lines to see how many records were upserted.

### `docker compose` command not found

Make sure you have Docker Compose V2. Run `docker compose version` (not `docker-compose`). If needed, update Docker Desktop.

---

## Development Setup (without Docker)

### Backend only

```bash
# Start DB + Redis via Docker, run backend locally
docker compose up db redis -d

cd backend
cp ../.env .env          # viper reads .env automatically
make run
```

### Frontend only

```bash
cd frontend
npm install
npm run dev              # starts on http://localhost:5173
                         # proxies /api → localhost:8080
```

---

## Environment Variables Reference

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATAGOKR_API_KEY` | Yes | — | data.go.kr API key |
| `DB_PASSWORD` | Yes | `changeme` | PostgreSQL password |
| `ADMIN_KEY` | Recommended | — | Secret for admin endpoints; empty = disabled |
| `DB_HOST` | No | `localhost` | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_NAME` | No | `sptraffic` | Database name |
| `DB_USER` | No | `sptraffic` | Database user |
| `REDIS_ADDR` | No | `localhost:6379` | Redis address |
| `SERVER_PORT` | No | `8080` | Backend HTTP port |
| `FRONTEND_PORT` | No | `3000` | Frontend port |
| `CORS_ORIGIN` | No | `http://localhost:5173` | Allowed CORS origin |
| `MIGRATIONS_DIR` | No | `./migrations` | Path to SQL migrations |
