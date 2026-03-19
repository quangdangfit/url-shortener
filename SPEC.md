Build a URL Shortener with Analytics system using Go and ScyllaDB.

## Overview
A URL shortening service where every redirect is tracked as an analytics event.
ScyllaDB runs via Docker. No frontend needed — REST API only.

## Tech Stack
- Language: Go 1.22+
- Database: ScyllaDB (Docker)
- Driver: github.com/scylladb/gocql (ScyllaDB fork, not gocql)
- HTTP: net/http with chi router
- Config: environment variables via godotenv
- No ORM

## Project Structure
url-shortener/
├── docker-compose.yml
├── .env.example
├── Makefile
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── db/
│   │   ├── scylla.go        # session, init, health check
│   │   └── migrations.go    # create keyspace + tables
│   ├── handler/
│   │   ├── shorten.go       # POST /shorten
│   │   ├── redirect.go      # GET /:code
│   │   └── stats.go         # GET /stats/:code
│   ├── repository/
│   │   ├── url_repo.go
│   │   └── click_repo.go
│   ├── service/
│   │   ├── shortener.go
│   │   └── analytics.go
│   └── model/
│       ├── url.go
│       └── click.go
├── benchmark/
│   ├── write_bench_test.go
│   └── read_bench_test.go
└── scripts/
└── seed.go              # seed fake data for benchmarking

## ScyllaDB Schema

### Keyspace
```cql
CREATE KEYSPACE IF NOT EXISTS urlshortener
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': 1
};
```

### Table 1: urls
Store the mapping from short code to original URL.
```cql
CREATE TABLE IF NOT EXISTS urlshortener.urls (
    code        TEXT PRIMARY KEY,
    original    TEXT,
    created_at  TIMESTAMP,
    expires_at  TIMESTAMP  -- nullable, NULL means no expiry
);
```

### Table 2: clicks
Store every click event. Partition by (code, bucket) where bucket = YYYY-MM-DD
to avoid unbounded partition growth.
```cql
CREATE TABLE IF NOT EXISTS urlshortener.clicks (
    code        TEXT,
    bucket      TEXT,         -- 'YYYY-MM-DD'
    clicked_at  TIMESTAMP,
    click_id    UUID,
    country     TEXT,
    device      TEXT,         -- 'mobile' | 'desktop' | 'unknown'
    referer     TEXT,
    PRIMARY KEY ((code, bucket), clicked_at, click_id)
) WITH CLUSTERING ORDER BY (clicked_at DESC)
  AND default_time_to_live = 7776000;  -- 90 days TTL
```

### Table 3: click_counts (counter table)
Fast counter for total clicks without scanning clicks table.
```cql
CREATE TABLE IF NOT EXISTS urlshortener.click_counts (
    code        TEXT,
    bucket      TEXT,
    total       COUNTER,
    PRIMARY KEY (code, bucket)
);
```

## API Endpoints

### POST /shorten
Request:
```json
{ "url": "https://example.com/very/long/path", "ttl_days": 30 }
```
Response:
```json
{ "code": "xK9a2", "short_url": "http://localhost:8080/xK9a2", "expires_at": "..." }
```
- Generate a 6-character alphanumeric code (Base62)
- Check collision, regenerate if exists
- ttl_days is optional, omit means no expiry

### GET /:code
- Look up original URL from urls table
- If not found → 404
- If expired → 410 Gone
- If found → 301 redirect to original URL
- Async write click event to clicks table (do NOT block the redirect)
- Async increment click_counts counter
- Parse User-Agent to detect device (mobile/desktop)
- Use X-Forwarded-For or RemoteAddr for country (just store raw IP, no geo lookup needed)

### GET /stats/:code
Response:
```json
{
  "code": "xK9a2",
  "original_url": "https://example.com/...",
  "total_clicks": 1200,
  "clicks_by_day": [
    { "date": "2024-03-01", "count": 340 },
    { "date": "2024-03-02", "count": 420 }
  ],
  "created_at": "..."
}
```
- total_clicks: sum from click_counts table (fast)
- clicks_by_day: query click_counts per bucket for last 30 days
- Return last 30 days only

### GET /health
```json
{ "status": "ok", "scylla": "ok" }
```

## docker-compose.yml
Run ScyllaDB with:
- Single node (no cluster needed for dev)
- Expose port 9042
- Named volume for persistence
- Health check using cqlsh

Also add a scylla-init service that:
- Waits for ScyllaDB to be healthy
- Runs the CQL schema (keyspace + tables)

## Makefile targets
- make dev         → run the API locally
- make docker-up   → docker compose up -d
- make docker-down → docker compose down
- make migrate     → run migrations manually
- make seed        → run scripts/seed.go to insert fake data
- make bench       → go test ./benchmark/... -bench=. -benchtime=10s
- make lint        → golangci-lint run

## Benchmark (benchmark/ directory)

### write_bench_test.go
Benchmark concurrent writes:
- BenchmarkSingleWrite: 1 goroutine writing clicks
- BenchmarkConcurrentWrites: 100 goroutines writing clicks simultaneously
- BenchmarkBatchInsert: insert 1000 clicks in tight loop

### read_bench_test.go
Benchmark reads:
- BenchmarkRedirect: simulate GET /:code (read urls table)
- BenchmarkStats: simulate GET /stats/:code (read click_counts)
- BenchmarkStatsRange: query clicks_by_day for 30 days

### scripts/seed.go
Insert realistic fake data:
- 1000 short codes
- Each code has random clicks spread across last 90 days
- Realistic distribution (some codes have 10 clicks, some have 100k)
- Print progress every 10k inserts

## Error Handling
- Wrap all ScyllaDB errors with context
- Return proper HTTP status codes
- Log errors with slog (Go standard library)
- Never expose internal errors to client

## Configuration (.env)
SCYLLA_HOSTS=localhost:9042
SCYLLA_KEYSPACE=urlshortener
SCYLLA_CONSISTENCY=LOCAL_QUORUM
SERVER_PORT=8080
BASE_URL=http://localhost:8080

## Important Implementation Notes
- Use ScyllaDB's gocql fork: github.com/scylladb/gocql — NOT the original gocql
- Redirect writes (click events) must be async — never block the 301 response
- Use a goroutine pool or buffered channel for async writes, not raw goroutines
- click_counts uses COUNTER type — must use UPDATE, not INSERT
- Bucket = date string "YYYY-MM-DD" in UTC
- Code generation: Base62 using chars [0-9a-zA-Z], exactly 6 chars
- Session should use connection pooling (NumConns: 4 per host)
- Add proper indexes — ScyllaDB does not support secondary indexes well, design queries around primary keys only
- All timestamps stored and returned in UTC
