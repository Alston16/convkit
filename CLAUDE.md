# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make dev      # start full local stack (server + Redis + Postgres + NATS via Docker Compose)
make test     # go test ./...
make build    # go build ./...
make migrate  # apply goose migrations against $CONVKIT_DSN

# Run a single test
go test ./internal/safety/...
go test ./pkg/tokenizer/... -run TestCountTokens

# Run the server directly (requires the Docker Compose stack to be up)
CONVKIT_CONFIG=config/server.yaml go run ./cmd/server
```

`go test ./...` must pass on every commit. The health endpoint is `GET /healthz`.

## Architecture

ConvKit is a **single-binary Go backend-as-a-service** for AI-powered chat. All six layers run in one process; concurrency (goroutines, channels, in-process pub/sub) handles isolation. External dependencies: PostgreSQL, Redis, NATS JetStream.

### Six Layers (`internal/`)

| Package | Layer | Responsibility |
|---|---|---|
| `transport` | 1 | WebSocket + REST, auth (JWT/API key), rate limiting |
| `bus` | 2 | NATS fan-out, Postgres durable store, presence (Redis), ordering |
| `stream` | 3 | SSE delta protocol, backpressure via `io.Pipe`, Redis replay buffer |
| `context` | 4 | Token-budget context assembly, memory tiers, RAG injection |
| `tools` | 5 | JSON Schema registry, timeout-isolated goroutine execution, LLM tool loop |
| `orchestration` | 6 | Multi-agent rooms, handoff protocol, shared memory graph, trace log |
| `safety` | — | Bidirectional middleware across every layer (no-op in Stage 0) |

`internal/common` holds all shared domain types (`Message`, `StreamToken`, `ToolCall`, etc.).

The public SDK lives in `pkg/sdk` (not yet implemented); `pkg/tokenizer` provides a pluggable tokenizer interface with a tiktoken-go implementation for OpenAI models.

### Layer Interface Pattern

Every layer exposes one Go interface; concrete types are always unexported. The standard constructor is `New(cfg Config) Interface`. Dependencies (including the safety plane) are injected via `Config`, never via globals.

```go
// internal/<layer>/<layer>.go
type <Layer> interface { ... }

type Config struct {
    Safety safety.SafetyPipeline
    // DB, Redis, NATS clients as needed
}

type service struct{ cfg Config }

func New(cfg Config) <Layer> { return &service{cfg: cfg} }
```

### Safety Plane — Non-Negotiable

Every inbound message **must** call `safety.RunInbound` before reaching the context engine. Every outbound token stream **must** call `safety.RunOutbound` on a rolling window. This applies from Stage 0 even though the current implementation is a no-op passthrough (`safety.NewNoop()`). The call site must exist.

```go
msg, verdict, err := s.cfg.Safety.RunInbound(ctx, msg)
if verdict.Action == common.Block {
    return ErrBlocked
}
```

### Message Lifecycle

```
User message → safety.RunInbound → Context engine → LLM inference
  → tool_call branch: Tool runtime → re-enter with result
  → text/stream branch: Streaming engine → safety.RunOutbound (rolling) → delta → client
```

### Database Migrations

All schema changes go in `migrations/` as goose-versioned `.sql` files. Committed migrations must never be modified — add a new one. Migrations run automatically on server startup.

### Error Wrapping Convention

```go
return fmt.Errorf("<package>.<Method>: %w", err)
// e.g. fmt.Errorf("bus.Publish: %w", err)
```

### Config

Loaded from `config/server.yaml` by default; override with `CONVKIT_CONFIG` env var. Contains `server.port`, `postgres.dsn`, `redis.addr`, `nats.url`.

## Current State

The project is at **Stage 0** (bootstrap). All six layers exist as interface stubs wired into `cmd/server/main.go`. The safety plane no-op is in place. Postgres migrations run on startup. The tokenizer (`pkg/tokenizer`) has a working tiktoken-go implementation and tests. Implementation of individual layers begins at Stage 1 (Transport).

## Project Knowledge Base

The `.github/` folder is the structured knowledge base for this repository:
- `.github/knowledge/` — architecture, domain model, tech stack, staging plan
- `.github/instructions/backend-api.md` — coding rules that apply to all `internal/**`, `pkg/**`, `cmd/**`
- `.github/patterns/layer-interface.md` — canonical interface + constructor pattern with all six layer contracts
- `.github/knowledge/staging-plan.md` — the eight-stage build plan with per-stage definitions of done

Reference these before implementing a new layer or making architectural changes.
