# URL Shortener

[![CI](https://github.com/quangdangfit/url-shortener/actions/workflows/ci.yml/badge.svg)](https://github.com/quangdangfit/url-shortener/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/quangdangfit/url-shortener/branch/main/graph/badge.svg)](https://codecov.io/gh/quangdangfit/url-shortener)

A high-performance URL shortening service with click analytics, built with Go, ScyllaDB, and Redis.

## Tech Stack

- **Language:** Go 1.24+
- **HTTP Framework:** [Fiber](https://gofiber.io/)
- **Database:** [ScyllaDB](https://www.scylladb.com/) (Cassandra-compatible)
- **Cache:** [Redis](https://redis.io/) (per-field TTL via `HEXPIRE`)
- **Driver:** [scylladb/gocql](https://github.com/scylladb/gocql)

## Architecture

The project follows **Clean Architecture** with strict dependency inversion — all dependencies point inward toward the domain layer.

```
cmd/
├── api/main.go                  # Entrypoint, dependency wiring
└── migrate/main.go              # Database migration runner

internal/
├── domain/                      # Entities (zero external dependencies)
│   ├── url.go                   #   URL entity
│   └── click.go                 #   Click, ClickCount entities
│
├── port/                        # Port interfaces (Dependency Inversion)
│   ├── service.go               #   Shortener, Analytics
│   └── repository.go            #   URLRepository, ClickRepository
│
├── usecase/                     # Use cases (business logic)
│   ├── shortener.go             #   URL shortening & resolution
│   └── analytics.go             #   Async click tracking & stats
│
├── handler/                     # Primary adapter (HTTP)
│   ├── shorten.go               #   POST /shorten
│   ├── redirect.go              #   GET /:code
│   └── stats.go                 #   GET /stats/:code
│
├── repository/                  # Secondary adapter (data access)
│   ├── scylla_url.go            #   ScyllaDB URL storage
│   ├── scylla_click.go          #   ScyllaDB click storage
│   └── cached_url.go            #   Redis-cached URL (decorator)
│
├── config/                      # Environment configuration
└── db/                          # ScyllaDB session & migrations

benchmark/                       # Performance benchmarks
scripts/                         # Seed data generator
```

**Dependency flow:**

```
handler ──→ port ←── usecase
              ↑
          repository
```

Handlers and repositories depend on `port` interfaces — never on each other or on concrete implementations.

## Getting Started

### Prerequisites

- Go 1.24+
- Docker & Docker Compose

### Run Infrastructure

```bash
make docker-up
```

Wait for ScyllaDB and Redis to be healthy, then run migrations:

```bash
make migrate
```

### Run the API

```bash
cp .env.example .env
make dev
```

The server starts at `http://localhost:8080`.

## API

### POST /shorten

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

- 6-character Base62 code with collision retry
- `ttl_days` is optional; omit for no expiry

### GET /:code

```bash
curl -L http://localhost:8080/xK9a2b
```

- `301` redirect to original URL
- `404` if not found, `410` if expired
- Click analytics recorded asynchronously

### GET /stats/:code

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
{ "status": "ok", "scylla": "ok", "redis": "ok" }
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `SCYLLA_HOSTS` | `localhost:9042` | ScyllaDB contact points |
| `SCYLLA_KEYSPACE` | `urlshortener` | Keyspace name |
| `SCYLLA_CONSISTENCY` | `LOCAL_QUORUM` | Consistency level |
| `REDIS_URI` | `localhost:6379` | Redis address |
| `SERVER_PORT` | `8080` | HTTP server port |
| `BASE_URL` | `http://localhost:8080` | Base URL for short links |

## Docker

```bash
# Build
docker build -t url-shortener .

# Run (after docker-compose infra is up)
make docker-up && make migrate
docker run --rm --network host url-shortener
```

## Makefile

| Command | Description |
|---|---|
| `make dev` | Run API locally |
| `make docker-up` | Start ScyllaDB + Redis |
| `make docker-down` | Stop containers |
| `make migrate` | Run database migrations |
| `make seed` | Insert fake data for benchmarking |
| `make unittest` | Run tests with coverage |
| `make bench` | Run benchmarks |
| `make lint` | Run golangci-lint |

## Database Schema

ScyllaDB tables designed for high write throughput and partition-aware queries:

- **urls** — short code → original URL mapping (partition key: `code`)
- **clicks** — click events partitioned by `(code, bucket)` where bucket = `YYYY-MM-DD`, 90-day TTL
- **click_counts** — counter table for fast per-day aggregation

## Design Decisions

- **Clean Architecture** — domain entities have zero external dependencies; all cross-layer communication goes through port interfaces, making every layer independently testable and replaceable
- **Redis cache** — single hash key `urls` with per-field `HEXPIRE` (5 min TTL); decorator pattern over `URLRepository` so caching is transparent to the use case layer
- **Async analytics** — buffered channel + goroutine worker pool ensures redirects are never blocked by click writes
- **Time-bucketed partitions** — click data partitioned by date to prevent unbounded partition growth
- **Counter tables** — separate counter table avoids full-scanning the clicks table for totals

## License

[MIT](LICENSE)
