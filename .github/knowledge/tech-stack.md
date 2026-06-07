# ConvKit — Technology Stack

## Language & Runtime

- **Go** (module: `github.com/Alston16/convkit`, go 1.26.2)
- Single-binary server — all layers in one process

## Core Dependencies

| Package | Version | Purpose |
|---|---|---|
| `github.com/coder/websocket` | v1.8.12 | WebSocket transport (actively maintained fork of `nhooyr.io/websocket`) |
| `github.com/nats-io/nats.go` | v1.36.0 | NATS JetStream for message fan-out and durable replay |
| `github.com/redis/go-redis/v9` | v9.5.0 | Presence state, working memory, stream token replay buffer |
| `github.com/jackc/pgx/v5` | v5.6.0 | PostgreSQL driver for durable message storage |
| `github.com/pressly/goose/v3` | v3.20.0 | SQL schema migrations (versioned, applied on startup) |
| `go.opentelemetry.io/otel` | v1.27.0 | Distributed tracing |
| `github.com/prometheus/client_golang` | v1.19.0 | Metrics (latency histograms, throughput, safety block rate, etc.) |
| `github.com/rs/zerolog` | v1.33.0 | Structured logging |

> **Note:** `github.com/coder/websocket` is used in place of `nhooyr.io/websocket`. It is the actively maintained fork by the same original author and is API-compatible.
>
> **Note:** `google.golang.org/grpc` is **not** required in V1. gRPC remains an option for the external SDK transport, but there is no inter-service RPC in the single-binary design.

## Infrastructure

| Service | Role |
|---|---|
| **PostgreSQL** | Durable message store, tool registry, agent registry, safety audit log, conversation memory |
| **Redis** | Presence (TTL-keyed), working memory (session state), stream token replay buffer (sorted set, TTL 5 min) |
| **NATS JetStream** | Message fan-out (`rooms.{roomID}.messages`), presence events |

## Observability Stack

| Signal | Library | What's captured |
|---|---|---|
| Structured logs | `rs/zerolog` | Every request, routing decision, safety event, error |
| Metrics | `prometheus/client_golang` | Latency histograms, throughput, streaming token rate, tool exec time, safety block rate |
| Distributed traces | `go.opentelemetry.io/otel` | Full trace per message: transport → delivery, LLM call, tool execution spans |
| Admin dashboard | Built-in HTTP handler | Live room list, agent trace viewer, safety event log, token budget visualiser |

Compatible with LGTM stack (Grafana + Loki + Tempo) and Datadog.

## Auth

- **JWT**: HS256 for development, RS256 for production
- **API keys**: hashed, stored in PostgreSQL
- Rate limiting: token bucket per connection (`golang.org/x/time/rate`)

## Local Development

```bash
make dev    # starts server + Redis + Postgres + NATS via Docker Compose
make test   # runs all unit tests (go test ./...)
```
