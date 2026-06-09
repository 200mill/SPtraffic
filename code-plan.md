# SPtraffic 코드 설계 계획

> plans.md의 기능 계획을 바탕으로 실제 코드 구조와 구현 방법을 기술한다.
> 기술 스택: Go (백엔드) + React + TypeScript (프론트엔드) + PostgreSQL + Redis

---

## 1. 디렉토리 구조

```
sptraffic/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go              # 엔트리포인트: 서버 초기화, 라우터 등록
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handler/
│   │   │   │   ├── terminal.go      # GET /api/terminals, /api/terminals/:id
│   │   │   │   ├── search.go        # GET /api/search
│   │   │   │   ├── schedule.go      # GET /api/schedule/:route_id
│   │   │   │   └── region.go        # GET /api/regions
│   │   │   ├── middleware/
│   │   │   │   ├── cors.go
│   │   │   │   └── ratelimit.go
│   │   │   └── router.go            # 라우트 등록
│   │   ├── domain/
│   │   │   ├── terminal.go          # Terminal, Station 타입 정의
│   │   │   ├── route.go             # Route 타입 정의
│   │   │   ├── schedule.go          # Schedule 타입 정의
│   │   │   └── search.go            # SearchQuery, SearchResult 타입
│   │   ├── repository/
│   │   │   ├── terminal_repo.go     # DB CRUD for terminals/stations
│   │   │   ├── route_repo.go        # DB CRUD for routes
│   │   │   └── schedule_repo.go     # DB CRUD for schedules
│   │   ├── service/
│   │   │   ├── search_service.go    # 경로 탐색 로직 (직행 + 환승)
│   │   │   └── schedule_service.go  # 시간표 조회 로직
│   │   ├── ingestion/
│   │   │   ├── client/
│   │   │   │   └── datagokr.go      # data.go.kr HTTP 클라이언트 (rate limit, retry)
│   │   │   ├── parser/
│   │   │   │   ├── express_bus.go   # 고속버스 API 응답 파싱
│   │   │   │   ├── intercity_bus.go # 시외버스 API 응답 파싱
│   │   │   │   └── rail.go          # 철도 API 응답 파싱
│   │   │   └── scheduler.go         # 일일 폴링 스케줄러 (cron)
│   │   ├── cache/
│   │   │   └── redis.go             # Redis 클라이언트 래퍼, 검색 결과 캐싱
│   │   └── config/
│   │       └── config.go            # 환경변수 로딩 (viper 또는 os.Getenv)
│   ├── migrations/
│   │   ├── 001_create_terminals.sql
│   │   ├── 002_create_routes.sql
│   │   └── 003_create_schedules.sql
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── Map/
│   │   │   │   ├── MapView.tsx       # Leaflet 지도 컨테이너
│   │   │   │   ├── TerminalMarker.tsx
│   │   │   │   └── RouteLine.tsx     # 검색 결과 폴리라인
│   │   │   ├── Search/
│   │   │   │   ├── SearchForm.tsx    # 출발지·도착지·날짜 입력
│   │   │   │   └── SearchResults.tsx # 결과 리스트
│   │   │   └── Schedule/
│   │   │       └── SchedulePopup.tsx # 마커 클릭 시 팝업
│   │   ├── hooks/
│   │   │   ├── useSearch.ts          # 검색 API 호출 및 상태 관리
│   │   │   └── useTerminals.ts       # 터미널 목록 로딩
│   │   ├── api/
│   │   │   └── client.ts             # fetch 래퍼, 백엔드 API 호출
│   │   ├── types/
│   │   │   └── index.ts              # Terminal, Route, Schedule, SearchResult 타입
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
│
├── docker-compose.yml               # postgres, redis, backend, frontend
└── .env.example                     # 환경변수 템플릿
```

---

## 2. 데이터베이스 스키마

```sql
-- 001_create_terminals.sql
CREATE TABLE terminals (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(20) UNIQUE NOT NULL,  -- data.go.kr 제공 코드
    name        VARCHAR(100) NOT NULL,
    type        VARCHAR(10) NOT NULL CHECK (type IN ('bus', 'rail')),
    region_code VARCHAR(10) NOT NULL,          -- 광역시도 코드
    lat         DOUBLE PRECISION NOT NULL,
    lon         DOUBLE PRECISION NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_terminals_region ON terminals(region_code);
-- PostGIS 사용 시: ALTER TABLE terminals ADD COLUMN geom GEOMETRY(Point, 4326);

-- 002_create_routes.sql
CREATE TABLE routes (
    id          SERIAL PRIMARY KEY,
    code        VARCHAR(30) UNIQUE,
    type        VARCHAR(15) NOT NULL CHECK (type IN ('express', 'intercity', 'rail')),
    origin_id   INT NOT NULL REFERENCES terminals(id),
    dest_id     INT NOT NULL REFERENCES terminals(id),
    operator    VARCHAR(50),
    created_at  TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_routes_origin ON routes(origin_id);
CREATE INDEX idx_routes_dest   ON routes(dest_id);

-- 003_create_schedules.sql
CREATE TABLE schedules (
    id             SERIAL PRIMARY KEY,
    route_id       INT NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    departure_time TIME NOT NULL,
    arrival_time   TIME,
    duration_min   INT,
    days_of_week   SMALLINT NOT NULL DEFAULT 127, -- bitmask: Mon=1 ... Sun=64
    valid_from     DATE,
    valid_until    DATE,
    created_at     TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_schedules_route ON schedules(route_id);
CREATE INDEX idx_schedules_dep   ON schedules(departure_time);
```

---

## 3. 핵심 Go 타입 및 인터페이스

```go
// internal/domain/terminal.go
type TerminalType string
const (
    Bus  TerminalType = "bus"
    Rail TerminalType = "rail"
)

type Terminal struct {
    ID         int          `db:"id"   json:"id"`
    Code       string       `db:"code" json:"code"`
    Name       string       `db:"name" json:"name"`
    Type       TerminalType `db:"type" json:"type"`
    RegionCode string       `db:"region_code" json:"regionCode"`
    Lat        float64      `db:"lat"  json:"lat"`
    Lon        float64      `db:"lon"  json:"lon"`
}

// internal/domain/search.go
type SearchQuery struct {
    FromCode string    // 출발 터미널 코드
    ToCode   string    // 도착 터미널 코드
    Date     time.Time
    Mode     string    // "bus" | "rail" | "all"
    MaxLegs  int       // 최대 환승 횟수 (1 or 2)
}

type Leg struct {
    From          Terminal
    To            Terminal
    DepartureTime time.Time
    ArrivalTime   time.Time
    RouteType     string
    Operator      string
}

type SearchResult struct {
    Legs         []Leg
    TotalMinutes int
    Transfers    int  // 환승 횟수 = len(Legs) - 1
}

// internal/repository/terminal_repo.go
type TerminalRepository interface {
    FindByCode(ctx context.Context, code string) (*Terminal, error)
    FindByRegion(ctx context.Context, regionCode string) ([]Terminal, error)
    Upsert(ctx context.Context, t *Terminal) error
}
```

---

## 4. 경로 탐색 알고리즘 (search_service.go)

```
SearchService.Search(query SearchQuery) ([]SearchResult, error)
  │
  ├─ 1) 직행 탐색
  │     SQL: SELECT s.* FROM schedules s
  │           JOIN routes r ON s.route_id = r.id
  │          WHERE r.origin_id = $from AND r.dest_id = $to
  │            AND s.departure_time >= $after
  │          ORDER BY s.departure_time
  │
  ├─ 2) 1회 환승 탐색
  │     a) FROM → mid: 출발지에서 갈 수 있는 모든 노선 조회
  │     b) mid → TO: 도착지로 오는 모든 노선 조회
  │     c) 두 집합의 교집합(mid 터미널)에서 조합 생성
  │     d) leg1.arrival + 30min <= leg2.departure 조건 필터
  │     e) 결과를 TotalMinutes 기준 정렬
  │
  └─ 3) 결과 병합 → []SearchResult 반환 (직행 우선, 환승 순)
```

**환승 탐색 복잡도 제어**
- mid 후보를 권역(region) 단위로 1차 필터링 (전체 탐색 방지)
- 결과 수가 많을 경우 상위 20개만 반환
- Redis에 `search:{from}:{to}:{date}` 키로 5분 캐싱

---

## 5. data.go.kr 클라이언트 (ingestion/client/datagokr.go)

```go
type Client struct {
    apiKey     string
    httpClient *http.Client   // timeout 설정 포함
    limiter    *rate.Limiter  // golang.org/x/time/rate — 초당 최대 N 요청
}

// 재시도 로직: 최대 3회, 1s → 2s → 4s backoff
func (c *Client) Get(ctx context.Context, endpoint string, params url.Values) ([]byte, error)
```

**수집 흐름 (scheduler.go)**
```
매일 02:00 KST (cron)
  ├─ FetchTerminals()    → upsert terminals 테이블
  ├─ FetchExpressBus()   → upsert routes + schedules (type=express)
  ├─ FetchIntercityBus() → upsert routes + schedules (type=intercity)
  └─ FetchRail()         → upsert routes + schedules (type=rail)
```

---

## 6. 프론트엔드 핵심 흐름

```
App
 ├─ useTerminals()          → GET /api/terminals → 마커 데이터
 │
 ├─ SearchForm
 │    └─ 사용자 입력 → useSearch(query)
 │                        └─ GET /api/search?from=&to=&date=&mode=
 │
 ├─ MapView (Leaflet)
 │    ├─ TerminalMarker[]   → 터미널 위치 표시
 │    │    └─ click → GET /api/schedule/:route_id → SchedulePopup
 │    └─ RouteLine[]        → SearchResult.Legs 기반 폴리라인
 │
 └─ SearchResults
      └─ 결과 카드 리스트 (총 소요시간, 환승 정보, 각 구간 시각)
```

**상태 관리**: React Context 또는 Zustand (전역 상태: 선택된 경로, 지도 포커스)

---

## 7. 환경변수 (.env.example)

```
# data.go.kr
DATAGOKR_API_KEY=your_key_here

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_NAME=sptraffic
DB_USER=sptraffic
DB_PASSWORD=changeme

# Redis
REDIS_ADDR=localhost:6379

# Server
SERVER_PORT=8080
CORS_ORIGIN=http://localhost:5173
```

---

## 8. Docker Compose 구성

```yaml
# docker-compose.yml (개요)
services:
  db:
    image: postgis/postgis:16-3.4
    environment: { POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD }
    volumes: [pgdata:/var/lib/postgresql/data]

  redis:
    image: redis:7-alpine
    volumes: [redisdata:/data]

  backend:
    build: ./backend
    depends_on: [db, redis]
    env_file: .env
    ports: ["8080:8080"]

  frontend:
    build: ./frontend
    ports: ["5173:5173"]         # 개발 모드
    environment: { VITE_API_BASE: http://localhost:8080 }
```

---

## 9. 구현 순서 (코드 작성 우선순위)

1. **DB 마이그레이션** — `migrations/` SQL 파일 작성 및 적용
2. **도메인 타입** — `internal/domain/` 구조체 및 인터페이스 정의
3. **레포지토리** — PostgreSQL CRUD 구현 (`pgx` 또는 `sqlx`)
4. **ingestion 클라이언트** — data.go.kr 연동, 파싱, upsert
5. **폴링 스케줄러** — `robfig/cron` 라이브러리로 일일 수집 등록
6. **검색 서비스** — 직행 쿼리 → 1회 환승 탐색 순으로 구현
7. **HTTP 핸들러** — Gin 라우터 + 핸들러 연결
8. **Redis 캐싱** — 검색 결과 캐싱 레이어 추가
9. **프론트엔드** — Leaflet 지도 → 검색 UI → 결과 표시 순

---

## 10. 주요 외부 패키지

| 패키지 | 용도 |
|--------|------|
| `github.com/gin-gonic/gin` | HTTP 라우터 |
| `github.com/jackc/pgx/v5` | PostgreSQL 드라이버 |
| `github.com/redis/go-redis/v9` | Redis 클라이언트 |
| `github.com/robfig/cron/v3` | 폴링 스케줄러 |
| `golang.org/x/time/rate` | API 호출 rate limiter |
| `github.com/spf13/viper` | 환경변수/설정 로딩 |
| `leaflet` (npm) | OSM 지도 렌더링 |
| `react-leaflet` (npm) | React용 Leaflet 래퍼 |
| `zustand` (npm) | 프론트엔드 상태 관리 |
| `axios` 또는 `ky` (npm) | 백엔드 API 호출 |
