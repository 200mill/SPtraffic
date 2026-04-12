#!/usr/bin/env bash
set -euo pipefail

# ── colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()    { echo -e "${CYAN}[info]${NC}  $*"; }
success() { echo -e "${GREEN}[ok]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[warn]${NC}  $*"; }
error()   { echo -e "${RED}[error]${NC} $*"; exit 1; }

# ── banner ────────────────────────────────────────────────────────────────────
echo -e "${CYAN}"
echo "  ╔═══════════════════════════════╗"
echo "  ║       SPtraffic Quickstart    ║"
echo "  ╚═══════════════════════════════╝"
echo -e "${NC}"

# ── 1. prerequisites ──────────────────────────────────────────────────────────
info "Checking prerequisites..."

command -v docker  >/dev/null 2>&1 || error "Docker is not installed. See https://docs.docker.com/get-docker/"
command -v docker compose >/dev/null 2>&1 || \
  docker compose version >/dev/null 2>&1   || \
  error "Docker Compose v2 is not installed."

success "Docker $(docker --version | awk '{print $3}' | tr -d ',')"

# ── 2. .env setup ─────────────────────────────────────────────────────────────
if [ ! -f .env ]; then
  info ".env not found — copying from .env.example"
  cp .env.example .env
  warn "Please edit .env and set DATAGOKR_API_KEY, DB_PASSWORD, and ADMIN_KEY, then re-run this script."
  exit 0
fi

success ".env found"

# Warn about unfilled placeholders
if grep -qE '^DATAGOKR_API_KEY=your_key_here' .env; then
  warn "DATAGOKR_API_KEY is not set — data ingestion will be disabled."
fi
if grep -qE '^ADMIN_KEY=change-this-secret' .env; then
  warn "ADMIN_KEY is still the default — change it before exposing to the internet."
fi

# ── 3. build + start ──────────────────────────────────────────────────────────
info "Building and starting all services..."
docker compose up --build -d

# ── 4. wait for backend health ────────────────────────────────────────────────
info "Waiting for backend to become healthy..."
BACKEND_PORT=$(grep -E '^SERVER_PORT=' .env | cut -d= -f2)
BACKEND_PORT=${BACKEND_PORT:-8080}

MAX_TRIES=30
TRIES=0
until curl -sf "http://localhost:${BACKEND_PORT}/health" >/dev/null 2>&1; do
  TRIES=$((TRIES + 1))
  if [ "$TRIES" -ge "$MAX_TRIES" ]; then
    error "Backend did not become healthy after ${MAX_TRIES} attempts. Run 'docker compose logs backend' to investigate."
  fi
  sleep 2
done

success "Backend is healthy at http://localhost:${BACKEND_PORT}"

# ── 5. trigger ingest (if API key is set) ─────────────────────────────────────
ADMIN_KEY=$(grep -E '^ADMIN_KEY=' .env | cut -d= -f2)
API_KEY=$(grep -E '^DATAGOKR_API_KEY=' .env | cut -d= -f2)

if [ -n "$API_KEY" ] && [ "$API_KEY" != "your_key_here" ] && \
   [ -n "$ADMIN_KEY" ] && [ "$ADMIN_KEY" != "change-this-secret" ]; then
  info "Triggering initial data ingest..."
  HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "http://localhost:${BACKEND_PORT}/api/admin/ingest" \
    -H "X-Admin-Key: ${ADMIN_KEY}")
  if [ "$HTTP_STATUS" = "202" ]; then
    success "Ingest started in background (runs daily at 02:00 KST after this)."
  else
    warn "Ingest returned HTTP ${HTTP_STATUS} — check logs with: docker compose logs backend"
  fi
else
  warn "Skipping ingest — set DATAGOKR_API_KEY and ADMIN_KEY in .env to enable."
fi

# ── 6. done ───────────────────────────────────────────────────────────────────
FRONTEND_PORT=$(grep -E '^FRONTEND_PORT=' .env | cut -d= -f2)
FRONTEND_PORT=${FRONTEND_PORT:-3000}

echo ""
echo -e "${GREEN}  All services are running!${NC}"
echo ""
echo "  Frontend  →  http://localhost:${FRONTEND_PORT}"
echo "  API       →  http://localhost:${BACKEND_PORT}/api"
echo "  Health    →  http://localhost:${BACKEND_PORT}/health"
echo ""
echo "  Useful commands:"
echo "    docker compose logs -f        # tail all logs"
echo "    docker compose logs backend   # backend logs only"
echo "    make shell-db                 # open a psql shell"
echo "    make down                     # stop containers"
echo ""
