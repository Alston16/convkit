<!--
Sync Impact Report
==================
Version change:      N/A → 1.0.0 (first authoring — all placeholder tokens resolved)
Modified principles: N/A (no prior version)
Added sections:
  - Core Principles I–VI
  - Streaming & Backpressure Standards
  - Development Workflow
  - Governance
Removed sections:    None
Templates requiring updates:
  - .specify/templates/plan-template.md  ✅ aligned (Constitution Check gate already present)
  - .specify/templates/spec-template.md  ✅ aligned (no constitution-specific sections required)
  - .specify/templates/tasks-template.md ✅ aligned (observability, testing, migration task categories present)
Follow-up TODOs:     None — all placeholder tokens resolved.
-->

# ConvKit Constitution

## Core Principles

### I. Safety-First (NON-NEGOTIABLE)

The safety plane MUST run as bidirectional middleware across every layer from Stage 0
onwards. Every inbound message MUST call `safety.RunInbound` before it reaches the
context engine. Every outbound token stream MUST call `safety.RunOutbound` on a rolling
window per token, and again at stream close for full-response filters. The no-op
passthrough satisfies this requirement during early stages — the call site MUST exist
even when the implementation does nothing.

Fail behaviour is **fail-closed** by default: when a safety filter errors, the message
is blocked, not passed through. A workspace may configure fail-open, but fail-closed is
the global default and MUST NOT be changed at the platform level.

**Rationale:** A safety violation that reaches a client or LLM is irreversible. The cost
of a blocked message is recoverable; the cost of leaked PII or a successful prompt
injection is not. Wiring every layer from Stage 0 prevents the "add safety later" trap.

### II. Layer Integrity

ConvKit is structured as six named layers — Transport, Message Bus, Streaming Engine,
Context Engine, Tool Runtime, Agent Orchestration — each owning exactly one `internal/`
package: `transport`, `bus`, `stream`, `context`, `tools`, `orchestration`.

Cross-layer communication MUST be direct Go function calls — no RPC, no channels across
package boundaries unless the interface contract explicitly requires it. Layers MUST NOT
skip adjacent layers in the pipeline; messages flow through the defined lifecycle in
order. Adding a new layer or splitting an existing one requires a constitution amendment.

**Rationale:** Clean layer ownership keeps cognitive load manageable, makes future
horizontal splitting straightforward (interfaces are already defined), and prevents
the spaghetti dependencies that accumulate when layers call each other arbitrarily.

### III. Observability is Non-Optional

Every layer MUST emit structured telemetry from its first commit:

- **Structured logs** via `rs/zerolog` — never `fmt.Println` or `log.Printf` in
  production paths.
- **Distributed traces** via `go.opentelemetry.io/otel` — one span per message from
  transport entry to delivery, including LLM call and tool execution spans.
- **Metrics** via `prometheus/client_golang` — at minimum: latency histograms, message
  throughput, streaming token rate, tool execution time, safety block rate.

No layer may ship without all three signals wired. Silent failures are not acceptable.

**Rationale:** A backend-as-a-service that AI applications depend on must be diagnosable
in production. Retro-fitting observability is expensive and error-prone; wiring it from
the first commit is the only reliable approach.

### IV. Schema Migration Discipline

All database schema changes MUST be expressed as goose-versioned `.sql` files in
`migrations/`. Committed migration files MUST NOT be modified — add a new migration
instead. Migrations are applied automatically on server startup.

**Rationale:** Versioned, auditable schema changes are the only safe path when the same
codebase must run across local dev, staging, and production. Editing committed migrations
creates irreproducible states that are difficult to diagnose and impossible to roll back
cleanly.

### V. Streaming Backpressure Guarantee

The streaming engine MUST use `io.Pipe`-based streaming. Slow clients MUST block the
writer — never drop tokens or buffer unboundedly. Token replay MUST be stored in a Redis
sorted set keyed `stream:{messageID}:tokens` (score = index, TTL 5 minutes) to support
reconnect-with-replay. Clients reconnecting MUST send `last_index`; the server MUST
replay from that index.

**Rationale:** Dropping tokens or allowing unbounded buffers both result in broken
user-visible chat experiences. Backpressure pushes flow control to the correct layer
(the transport), and the Redis replay buffer provides the reconnect guarantee without
retaining state indefinitely.

### VI. Simplicity Over Premature Scaling

V1 is a **single-binary server**. All six layers run in one process. No network boundary
is added between layers until load actually demands horizontal scaling. Complexity MUST
be justified: the `plan.md` Complexity Tracking table MUST document every deviation from
this principle with the concrete load signal that warrants it.

**Rationale:** Premature distribution adds operational overhead, introduces distributed
systems failure modes, and delays shipping. The clean internal package boundaries already
define the split points — adding a network boundary later is straightforward when load
requires it.

## Streaming & Backpressure Standards

All streaming behaviour in the Streaming Engine (Layer 3) and the Tool Runtime (Layer 5)
MUST comply with Principle V. In addition:

- Outbound safety filters run on a **rolling window of accumulated tokens** as the stream
  progresses — not as a post-hoc check after full delivery.
- Filters that require the full response (e.g. holistic coherence checks) MUST be applied
  at stream close, not mid-stream.
- Tool handlers MAY emit `ToolProgress` events during execution; these MUST be delivered
  to the client in real time before the final `ToolResult`.

## Development Workflow

- `make dev` is the canonical command to start the full local stack (server + Redis +
  Postgres + NATS via Docker Compose).
- `make test` runs `go test ./...`. This MUST pass on every commit — a failing test suite
  is a blocking issue, not a deferred task.
- All dependencies are added via `go get <module>@<version>` — never by editing `go.mod`
  directly.
- Authentication in development uses JWT HS256; production deployments MUST use RS256.
- API keys are stored only as hashes in PostgreSQL; the raw key is never persisted.
- Each layer MUST have unit tests against its interface (not its internal implementation).
  Integration tests use the embedded Docker Compose stack (`make dev`) — no mocks for
  the database, NATS, or Redis in integration tests.

## Governance

This constitution supersedes all other practices and guidelines for the ConvKit project.
Any conflict between this document and a README, comment, or individual decision record
is resolved in favour of this constitution.

**Amendment procedure:**
1. Propose the amendment in a pull request that modifies this file.
2. The amendment MUST include a rationale and, where applicable, a migration plan for
   existing code that would be affected.
3. Version the amendment according to the semantic versioning rules below.
4. All affected templates (`plan-template.md`, `spec-template.md`, `tasks-template.md`)
   MUST be updated in the same pull request.

**Versioning policy:**
- MAJOR: backward-incompatible governance change, principle removal, or redefinition.
- MINOR: new principle or section added, or materially expanded guidance.
- PATCH: clarifications, wording improvements, typo fixes, non-semantic refinements.

**Compliance review:**
All feature PRs MUST satisfy the Constitution Check gate in `plan.md` before merging.
Any deviation from a principle MUST be documented in the Complexity Tracking table of
`plan.md` with explicit justification.

**Version**: 1.0.0 | **Ratified**: 2026-05-02 | **Last Amended**: 2026-06-05
