# URL Shortener

[![CI](https://github.com/quangdangfit/url-shortener/actions/workflows/ci.yml/badge.svg)](https://github.com/quangdangfit/url-shortener/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/quangdangfit/url-shortener/branch/main/graph/badge.svg)](https://codecov.io/gh/quangdangfit/url-shortener)

A high-performance URL shortening service with click analytics, built with Go and ScyllaDB.

## Tech Stack

- **Language:** Go 1.24+
- **HTTP Framework:** [Fiber](https://gofiber.io/)
- **Database:** [ScyllaDB](https://www.scylladb.com/) (Cassandra-compatible)
- **Driver:** [scylladb/gocql](https://github.com/scylladb/gocql)
- **Config:** Environment variables via [godotenv](https://github.com/joho/godotenv)

## Architecture

```
cmd/
└── api/main.go              # Fiber server entrypoint
└── migrate/main.go          # Database migration runner
internal/
├── config/                  # Environment configuration
├── db/                      # ScyllaDB session & migrations
├── handler/                 # HTTP handlers (shorten, redirect, stats)
├── model/                   # Data models (URL, Click)
├── repository/              # ScyllaDB data access layer
└── service/                 # Business logic (shortener, analytics)
benchmark/                   # Performance benchmarks
scripts/                     # Seed data generator
```

## Getting Started

### Prerequisites

- Go 1.24+
- Docker & Docker Compose

### Run ScyllaDB

```bash
make docker-up
```

Wait for ScyllaDB to be healthy, then run migrations:

```bash
make migrate
```

### Run the API

```bash
cp .env.example .env
make dev
```

The server starts at `http://localhost:8080`.

## API Endpoints

### POST /shorten

Create a short URL.

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/very/long/path", "ttl_days": 30}'
```

```json
{
  "code": "xK9a2b",
  "short_url": "http://localhost:8080/xK9a2b",
  "expires_at": "2024-04-01T00:00:00Z"
}
```

- Generates a 6-character Base62 code with collision detection
- `ttl_days` is optional; omit for no expiry

### GET /:code

Redirect to the original URL.

```bash
curl -L http://localhost:8080/xK9a2b
```

- Returns `301 Moved Permanently`
- Returns `404` if not found, `410 Gone` if expired
- Records click analytics asynchronously (does not block redirect)

### GET /stats/:code

Get click analytics for a short URL.

```bash
curl http://localhost:8080/stats/xK9a2b
```

```json
{
  "code": "xK9a2b",
  "original_url": "https://example.com/very/long/path",
  "total_clicks": 1200,
  "clicks_by_day": [
    { "date": "2024-03-01", "count": 340 },
    { "date": "2024-03-02", "count": 420 }
  ],
  "created_at": "2024-03-01T00:00:00Z"
}
```

### GET /health

```json
{ "status": "ok", "scylla": "ok" }
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `SCYLLA_HOSTS` | `localhost:9042` | ScyllaDB contact points |
| `SCYLLA_KEYSPACE` | `urlshortener` | Keyspace name |
| `SCYLLA_CONSISTENCY` | `LOCAL_QUORUM` | Consistency level |
| `SERVER_PORT` | `8080` | HTTP server port |
| `BASE_URL` | `http://localhost:8080` | Base URL for short links |

## Docker

### Build

```bash
docker build -t url-shortener .
```

### Run with Docker Compose

```bash
make docker-up
make migrate
docker run --rm --network host \
  -e SCYLLA_HOSTS=localhost:9042 \
  url-shortener
```

## Makefile

| Command | Description |
|---|---|
| `make dev` | Run API locally |
| `make docker-up` | Start ScyllaDB container |
| `make docker-down` | Stop ScyllaDB container |
| `make migrate` | Run database migrations |
| `make seed` | Insert fake data for benchmarking |
| `make unittest` | Run tests with coverage |
| `make bench` | Run benchmarks |
| `make lint` | Run golangci-lint |

## Database Schema

ScyllaDB tables designed for high write throughput and partition-aware queries:

- **urls** - Short code to original URL mapping (partition key: `code`)
- **clicks** - Individual click events, partitioned by `(code, bucket)` where bucket = `YYYY-MM-DD`, with 90-day TTL
- **click_counts** - Counter table for fast aggregation per code per day

## Design Decisions

- **Async analytics:** Click events are written via a buffered channel with a goroutine worker pool, ensuring redirects are never blocked by analytics writes
- **Time-bucketed partitions:** Click data is partitioned by date to prevent unbounded partition growth
- **Counter tables:** Separate counter table avoids scanning the clicks table for totals
- **Base62 codes:** 6-character codes give ~56 billion unique combinations with collision retry

## License

[MIT](LICENSE)
