# ConvKit — System Architecture

## Overview

ConvKit is a **Go-based backend-as-a-service** for AI-powered chat applications. It handles conversation infrastructure — rooms, delivery, presence, streaming, context, tool calling, and multi-agent orchestration — so application developers write only AI logic.

## Deployment Model

**Single-binary design.** All six layers run in one process. Concurrency within the process (goroutines, channels, in-process pub/sub) handles room isolation and fan-out without a network hop.

- External dependencies: Redis, PostgreSQL, NATS JetStream
- `make dev` starts the full stack via Docker Compose (server + Redis + Postgres + NATS)
- Clean internal package boundaries make future horizontal splitting straightforward

## Six-Layer Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Safety Plane (all layers)               │
│         Moderation · PII redaction · Audit · Policy         │
├─────────────────────────────────────────────────────────────┤
│  Layer 6 — Agent Orchestration                              │
│  Multi-agent rooms · Supervisor/worker · Shared memory      │
├─────────────────────────────────────────────────────────────┤
│  Layer 5 — Tool Runtime                                     │
│  Schema registry · Timeout-isolated execution · Result injection │
├─────────────────────────────────────────────────────────────┤
│  Layer 4 — Context Engine                                   │
│  Window mgmt · Memory tiers · RAG injection · Token budget  │
├─────────────────────────────────────────────────────────────┤
│  Layer 3 — Streaming Engine                                 │
│  SSE delta protocol · Backpressure · Reconnect with replay  │
├─────────────────────────────────────────────────────────────┤
│  Layer 2 — Message Bus                                      │
│  Fan-out · Ordering · Delivery guarantees · Presence        │
├─────────────────────────────────────────────────────────────┤
│  Layer 1 — Transport                                        │
│  WebSocket · REST · Auth · Rate limiting                    │
└─────────────────────────────────────────────────────────────┘
```

## Package Structure

```
convkit/
├── cmd/
│   ├── server/             # Single server entrypoint
│   └── cli/                # Developer CLI
├── internal/
│   ├── transport/          # Layer 1
│   ├── bus/                # Layer 2
│   ├── stream/             # Layer 3
│   ├── context/            # Layer 4
│   ├── tools/              # Layer 5
│   ├── orchestration/      # Layer 6
│   └── safety/             # Safety plane (bidirectional middleware)
├── pkg/
│   ├── sdk/                # Public Go SDK
│   └── tokenizer/          # Pluggable tokenizer interface + implementations
├── migrations/             # goose SQL migration files
├── config/
│   └── server.yaml
├── deploy/
│   ├── docker-compose.yml
│   └── k8s/
├── benchmarks/
└── docs/
```

## Message Lifecycle

```
User message
      │
      ▼
Safety plane (inbound)
      │
      ▼
Context engine  ←── window + memory + persona + RAG + token budget
      │
      ▼
LLM inference (provider-agnostic)
      │
      ├── tool call ──► Tool runtime ──► re-enter with result ──┐
      │                                                         │
      └── text/stream ──► Streaming engine ──► Safety plane (outbound, per-token rolling check)
                                                    │
                                                    ▼
                                               delta → client
```

## Internal Communication

| Concern | Mechanism |
|---|---|
| Layer-to-layer calls | Direct Go function calls (no network boundary) |
| Message fan-out to WebSocket clients | NATS JetStream (`rooms.{roomID}.messages`) |
| Presence / session state | Redis (TTL-keyed) |
| Durable message store | PostgreSQL |
| Database migrations | `github.com/pressly/goose` — versioned, applied on startup |

## Safety Plane

Runs as **bidirectional middleware across every layer**. Introduced as a no-op passthrough in Stage 0 so every layer plugs into it from day one. Fully implemented in Stage 7.

- Inbound: called before context assembly
- Outbound: rolling window per-token check during streaming; full-response filters at stream close
- Configurable per workspace / room / bot with inheritance (bot > room > workspace)
- Default: **fail-closed** (block on filter error)

## Key Design Decisions

| Decision | V1 Choice | Rationale |
|---|---|---|
| Deployment topology | Single binary | Eliminates distributed complexity before needed |
| Message fan-out | NATS JetStream | Low overhead, at-least-once, built-in replay |
| Streaming protocol | SSE delta over WebSocket | Simpler reconnect semantics |
| Tokenizer | Pluggable interface, per-model | Model-agnostic; `ModelID` selects at runtime |
| Context window mgmt | Token-counting + recency | Deterministic, model-agnostic |
| Tool execution | Goroutine per call + timeout | Near-zero overhead; cancellation isolation |
| Memory graph conflicts | Last-write-wins + vector clock | Simple; vector clock makes conflicts visible |
| Multi-agent routing | Explicit handoff (pull) | Auditable, no surprise concurrency |
| Safety pipeline | Middleware chain, fail-closed | Composable, testable |
| LLM provider | Provider-agnostic interface | Apps bring their own LLM |
| Database migrations | goose | Versioned, auditable schema changes |
