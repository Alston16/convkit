# ConvKit — AI-Native Conversation Infrastructure

> **Go-based backend-as-a-service for building AI-powered chat applications.**  
> ConvKit handles all conversation infrastructure — rooms, delivery, presence, streaming, context, tool calling, and multi-agent orchestration — so your application focuses entirely on AI logic, not plumbing.

---

## The Problem

Most chat backends treat bots as afterthoughts. They handle messaging well but leave AI teams re-inventing the same infrastructure:

| What generic chat providers give you | What AI applications actually need |
|---|---|
| Message delivery | **Streaming token delivery** (delta protocol) |
| Static message history | **Context window management** (memory tiers, RAG injection) |
| Webhook on message | **Tool / function calling runtime** (schema registry, timeout-isolated execution) |
| One bot per integration | **Multi-agent orchestration** (supervisor/worker topologies, shared memory) |
| Content moderation (maybe) | **Safety plane across every layer** (PII, injection attacks, policy enforcement) |

ConvKit is built around the inversion: **the AI is a first-class citizen of the infrastructure**, not a plugin bolted on top.

---

## Architecture

ConvKit is structured as six layers, each owning a distinct responsibility. Layers 1–2 handle reliable message transport. Layers 3–6 are the AI-native primitives that differentiate ConvKit from commodity chat backends. A **safety plane** runs as bidirectional middleware across every layer.

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

### Single-Binary Design

ConvKit runs as a **single server process**. All six layers live in one binary; concurrency within the process (goroutines, channels, in-process pub/sub) handles room isolation and fan-out without a network hop. NATS JetStream is still used for durable message persistence and replay — it just isn't bridging two separate services.

This keeps local development simple (`make dev` starts one server process plus Redis, Postgres, and NATS), eliminates an entire class of distributed systems complexity from V1, and defers horizontal scaling concerns until load actually demands it. When it does, the clean internal package boundaries make it straightforward to split layers across nodes — the interfaces are already defined, you're just adding a network boundary between them.

### Message Lifecycle

Every message — human or agent — flows through the same in-process pipeline:

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

> **Note on outbound safety and streaming:** Outbound safety filters run on a rolling window of accumulated tokens as the stream progresses — not as a post-hoc check after full delivery. This avoids buffering the entire response before the client sees anything, while still catching policy violations before they reach the client. Filters that require the full response (e.g. holistic coherence checks) should be applied at stream close.

### Internal Communication

- **In-process layer communication**: direct Go function calls across packages — no network boundary
- **Message fan-out to WebSocket clients**: NATS JetStream (`rooms.{roomID}.messages`)
- **Presence / session state**: Redis (TTL-keyed)
- **Durable message store**: PostgreSQL
- **Database migrations**: `github.com/pressly/goose` — all schema changes are versioned and applied on startup

---

## Repository Structure

```
convkit/
├── cmd/
│   ├── server/             # Single server entrypoint
│   └── cli/                # Developer CLI (room management, bot registration)
├── internal/
│   ├── transport/          # Layer 1: WebSocket, REST handlers
│   ├── bus/                # Layer 2: Message bus, fan-out, presence
│   ├── stream/             # Layer 3: SSE delta engine, backpressure
│   ├── context/            # Layer 4: Context engine, memory tiers, RAG
│   ├── tools/              # Layer 5: Tool runtime, schema registry
│   ├── orchestration/      # Layer 6: Multi-agent rooms, supervisor, memory graph
│   └── safety/             # Safety plane middleware (bidirectional, no-op in Stage 0)
├── pkg/
│   ├── sdk/                # Public Go SDK (what application developers import)
│   └── tokenizer/          # Pluggable tokenizer interface + per-model implementations
├── migrations/             # goose SQL migration files
├── config/
│   └── server.yaml
├── deploy/
│   ├── docker-compose.yml  # Local dev: server + Redis + Postgres + NATS
│   └── k8s/                # Kubernetes manifests
├── benchmarks/             # Latency, throughput, and recall benchmarks
└── docs/
    ├── architecture.md
    ├── sdk-guide.md
    └── deployment.md
```

---

## Go Module Layout

```
module github.com/yourorg/convkit

go 1.23

require (
    github.com/coder/websocket                v1.8.12
    github.com/nats-io/nats.go               v1.36.0
    github.com/redis/go-redis/v9             v9.5.0
    github.com/jackc/pgx/v5                  v5.6.0
    github.com/pressly/goose/v3              v3.20.0
    go.opentelemetry.io/otel                 v1.27.0
    github.com/prometheus/client_golang      v1.19.0
    github.com/rs/zerolog                    v1.33.0
)
```

> `github.com/coder/websocket` is used in place of `nhooyr.io/websocket`. It is the actively maintained fork by the same original author and is API-compatible. `nhooyr.io/websocket` is no longer actively maintained.

> `google.golang.org/grpc` is not required in V1. gRPC remains an option for the external SDK transport, but there is no inter-service RPC in the single-binary design.

---

## SDK — What Application Developers See

The public-facing SDK hides every layer below. Application developers write only AI logic.

```go
import "github.com/yourorg/convkit/pkg/sdk"

// Register a bot
bot := sdk.NewBot(sdk.BotConfig{
    Name:      "support-agent",
    Workspace: "acme-corp",
    APIKey:    os.Getenv("CONVKIT_API_KEY"),
})

// Handle messages — streaming response
bot.OnMessage(func(ctx context.Context, msg *sdk.Message) {
    stream := bot.NewStream(ctx, msg)
    defer stream.Close()

    // Call your LLM — ConvKit handles the rest
    for token := range myLLM.Stream(ctx, msg.ContextWindow()) {
        stream.Write(token)
    }
})

// Register a callable tool
bot.RegisterTool(sdk.Tool{
    Name:        "search_orders",
    Description: "Look up orders by customer ID",
    Schema:      orderSearchSchema,
    Handler:     handleOrderSearch,
})

bot.Start()
```

---

## Staging Plan

The build is structured in eight stages. Each stage ships a working, testable artifact. Stages 0–2 cover the foundation. Stages 3–7 are the AI-native layers that constitute ConvKit's core value.

The safety plane is introduced as a **no-op passthrough in Stage 0** so every subsequent layer plugs into it from day one. It is fully implemented in Stage 7.

---

### Stage 0 — Project Bootstrap

**Goal:** Working repository, shared types, and a safety plane skeleton that every later layer can plug into from the start.

**Deliverables:**
- [x] Go module initialised: `github.com/yourorg/convkit`
- [x] Core domain types defined in `internal/common`:
  - `RoomID`, `UserID`, `BotID`, `MessageID`
  - `Message{ID, RoomID, SenderID, Body, CreatedAt, Metadata}`
  - `StreamToken{MessageID, Delta, Index, Done}`
  - `ToolCall{ID, Name, Arguments json.RawMessage}`
  - `ToolResult{CallID, Output json.RawMessage, Error string}`
- [x] **Safety plane skeleton** in `internal/safety`: `SafetyPipeline` interface defined; default implementation is a no-op passthrough. All layers will call `RunInbound` / `RunOutbound` from their first commit — the no-op means no behaviour change until Stage 7.
- [x] **Pluggable tokenizer interface** in `pkg/tokenizer`: `Tokenizer` interface with `CountTokens(model, text string) int`. Implementations for OpenAI (tiktoken-go) and a character-estimate fallback. Model ID selects the implementation at runtime.
- [x] **Migration baseline**: `goose` configured; initial empty schema migration in `migrations/`.
- [x] Docker Compose environment: single server + Redis + Postgres + NATS
- [x] `make dev` brings the full environment up; `make test` runs all unit tests

**Definition of done:** `go test ./...` passes. Server process starts and the health endpoint returns `200 OK`. Safety plane no-op passthrough is callable from all layer stubs.

---

### Stage 1 — Layer 1: Transport

**Goal:** Clients can connect over WebSocket and REST. All connections are authenticated and rate-limited.

**Scope:**
- [ ] WebSocket handler (`github.com/coder/websocket`) — one goroutine per connection, context-cancelled on disconnect
- [ ] REST API for room CRUD, message history fetch, bot registration
- [ ] Auth middleware: JWT (HS256 for development, RS256 for production) and API key (hashed, stored in Postgres)
- [ ] Per-connection rate limiter using a token bucket (`golang.org/x/time/rate`)
- [ ] All inbound messages pass through `safety.RunInbound` (no-op at this stage)

**Key interfaces:**
```go
type Transport interface {
    RegisterHandler(path string, h MessageHandler)
    ListenAndServe(ctx context.Context, addr string) error
}

type MessageHandler func(ctx context.Context, conn Connection, msg RawMessage) error
```

**Tests:**
- [ ] WebSocket connect/disconnect lifecycle
- [ ] Auth rejection on missing or invalid JWT
- [ ] Rate limit enforcement (expect 429 after N requests/second)

**Definition of done:** A test client connects over WebSocket, sends a message, receives an echo. All connections without valid auth are rejected.

---

### Stage 2 — Layer 2: Message Bus

**Goal:** Messages are durably stored and fan out to all room subscribers, with presence and ordering guarantees.

**Scope:**
- [ ] NATS JetStream subject layout: `rooms.{roomID}.messages`, `rooms.{roomID}.presence`
- [ ] Message storage: Postgres `messages` table with `(room_id, seq_num, sender_id, body, created_at)` — added via goose migration
- [ ] Fan-out: server publishes to NATS; a subscriber goroutine per room forwards to connected WebSocket clients
- [ ] Presence service: Redis `SETEX rooms:{roomID}:users:{userID} 30` — heartbeat every 10s
- [ ] Typing indicators: ephemeral NATS publish, no persistence, TTL 3s
- [ ] Message ordering: monotonic `seq_num` per room, assigned at write time

**Key interfaces:**
```go
type Bus interface {
    Publish(ctx context.Context, roomID RoomID, msg Message) error
    Subscribe(ctx context.Context, roomID RoomID) (<-chan Message, error)
}

type PresenceService interface {
    Heartbeat(ctx context.Context, roomID RoomID, userID UserID) error
    Online(ctx context.Context, roomID RoomID) ([]UserID, error)
    SetTyping(ctx context.Context, roomID RoomID, userID UserID) error
}
```

**Tests:**
- [ ] Insert 1,000 messages into a room; verify `seq_num` monotonicity
- [ ] 3 subscribers on the same room all receive the same message within 50ms
- [ ] Presence expires correctly after heartbeat stops

**Definition of done:** Two WebSocket clients in the same room exchange messages. Presence correctly reflects online/offline transitions.

---

### Stage 3 — Layer 3: Streaming Engine

**Goal:** AI responses stream token-by-token to clients. Clients can reconnect mid-stream without losing tokens.

**Scope:**
- [ ] **Delta protocol**: each `StreamToken` carries `{messageID, delta, index, done}`. Only the changed delta is transmitted, not the full accumulated text.
- [ ] `io.Pipe`-based streaming: LLM goroutine writes tokens; streaming engine goroutine reads and forwards. If the client is slow, the write blocks (backpressure) rather than buffering unboundedly.
- [ ] **Reconnect with replay**: token log stored in Redis as a sorted set `stream:{messageID}:tokens` with score = index, TTL = 5 minutes. On reconnect, client sends `last_index`; server replays from that index.
- [ ] Typing indicator automation: streaming engine emits a `typing=true` presence event when a stream opens, clears it when done.
- [ ] All outbound tokens pass through `safety.RunOutbound` on a rolling window (no-op at this stage).

**Key interfaces:**
```go
type Streamer interface {
    // Called by bot SDK to open a response stream
    OpenStream(ctx context.Context, replyTo MessageID) (StreamWriter, error)
}

type StreamWriter interface {
    Write(delta string) error    // blocks on backpressure
    Close() error
}
```

**Tests:**
- [ ] Stream 500 tokens; verify client receives all 500 in order
- [ ] Simulate client disconnect at token 250; reconnect with `last_index=249`; verify tokens 250–500 replayed
- [ ] Slow client (artificial delay) must not cause writer to drop tokens — verify blocking, not dropping

**Definition of done:** A bot sends a streaming response. A client that disconnects mid-stream and reconnects receives the complete message without gaps.

---

### Stage 4 — Layer 4: Context Engine

**Goal:** Every LLM call receives the correct, token-budget-aware slice of conversation history, persona, and retrieved knowledge.

**Memory tiers:**

| Tier | Storage | Scope | TTL |
|---|---|---|---|
| Working memory | Redis | Current session | Session lifetime |
| Conversation memory | Postgres | Conversation lifetime | Configurable (default: 30 days) |
| Long-term memory | pgvector / Qdrant | Cross-conversation semantic retrieval | Permanent |

**Scope:**
- [ ] **Window management algorithm:**
  1. Fetch raw conversation history (Postgres), most-recent-first
  2. Tokenise each message using the pluggable `Tokenizer` interface, selecting the implementation by `ModelID`
  3. Fill window from most-recent backwards until `max_tokens - reserved_output_tokens` budget is reached
  4. Inject persona system prompt at position 0 (always included, counts against budget)
  5. If long-term memory enabled: embed current user message → retrieve top-K similar past exchanges → inject as `<memory>` block after system prompt
  6. If RAG enabled: retrieve top-K documents → inject as `<context>` block
- [ ] Redis working memory: fast key-value store for in-flight session state (e.g. pending tool calls, draft responses)
- [ ] Postgres conversation memory: durable message log with configurable retention (goose migration)
- [ ] Vector store integration (pgvector default, Qdrant optional): embed and retrieve past exchanges
- [ ] Persona config: per-bot system prompt stored in Postgres, loaded on context assembly

**Key interfaces:**
```go
type ContextEngine interface {
    Assemble(ctx context.Context, opts AssembleOpts) ([]LLMMessage, error)
}

type AssembleOpts struct {
    RoomID        RoomID
    BotID         BotID
    UserMessage   Message
    ModelID       string     // selects the Tokenizer implementation
    MaxTokens     int
    ReserveOutput int
    RAGQuery      string     // empty = skip RAG
}
```

**Tests:**
- [ ] Insert 10,000-token conversation history; assert assembled context never exceeds `max_tokens`
- [ ] Persona is always present at position 0 regardless of window truncation
- [ ] Long-term memory retrieval returns semantically relevant messages, not just recent ones
- [ ] Token count accuracy within ±5% of actual model tokeniser output (tested per model implementation)

**Definition of done:** A bot receives a context window that fits its model's token limit, contains the persona, and includes semantically relevant past exchanges when long-term memory is enabled.

---

### Stage 5 — Layer 5: Tool Runtime

**Goal:** Bots can declare callable tools. The runtime validates arguments, executes with timeout isolation, and injects results back into the LLM loop — automatically.

**Scope:**
- [ ] **Schema registry**: tools registered per workspace as JSON Schema. Stored in Postgres `tools(workspace_id, name, description, schema, endpoint, auth_config)` (goose migration).
- [ ] **Timeout-isolated execution**: each tool call runs in a dedicated goroutine with a configurable timeout (default 10s) and context cancellation. Tool handlers may be local Go functions (SDK) or remote HTTP endpoints. This provides timeout and cancellation isolation; it is not a security sandbox — untrusted tool handlers should be remote HTTP endpoints, not local Go functions.
- [ ] **LLM loop**: when the LLM returns a `tool_call`, the runtime executes it, injects the `tool_result` back into the context, and re-calls the LLM — looping until the model returns a final text response.
- [ ] **Streaming tool results**: tools may stream progress updates during execution. These appear as `ToolProgress` events on the client in real-time.
- [ ] **Tool auth delegation**: tools can be called with the end-user's credentials (OAuth token forwarding), not the bot's credentials.

**Key interfaces:**
```go
type ToolRegistry interface {
    Register(ctx context.Context, tool ToolDefinition) error
    Resolve(ctx context.Context, workspaceID, name string) (ToolDefinition, error)
}

type ToolRuntime interface {
    Execute(ctx context.Context, call ToolCall, userCtx UserContext) (ToolResult, error)
    ExecuteStream(ctx context.Context, call ToolCall, userCtx UserContext) (<-chan ToolProgress, error)
}
```

**Bot SDK surface:**
```go
bot.RegisterTool(sdk.Tool{
    Name:        "fetch_invoice",
    Description: "Fetch invoice details by invoice ID",
    Schema: jsonschema.Object{
        Properties: map[string]jsonschema.Schema{
            "invoice_id": {Type: "string"},
        },
        Required: []string{"invoice_id"},
    },
    Handler: func(ctx context.Context, args json.RawMessage, user sdk.UserContext) (any, error) {
        var input struct{ InvoiceID string `json:"invoice_id"` }
        json.Unmarshal(args, &input)
        return fetchInvoice(ctx, input.InvoiceID, user.Token)
    },
})
```

**Tests:**
- [ ] LLM requests a tool call → runtime executes → result injected → LLM produces final text; verify full loop completes
- [ ] Tool timeout (set to 100ms, handler sleeps 500ms) → `context.DeadlineExceeded` returned, bot receives error result
- [ ] Invalid tool arguments (schema mismatch) → execution rejected before handler is called
- [ ] Streaming tool result delivers progress events before final result

**Definition of done:** A bot with a registered tool automatically executes it when the LLM requests it, receives the result, and produces a coherent final response — without any application-level loop management.

---

### Stage 6 — Layer 6: Agent Orchestration

**Goal:** Multiple agents can collaborate in a single room, with defined roles, a shared memory graph, and observable execution traces.

**Multi-agent room topology:**
```
Room
├── Supervisor agent     (orchestrates sub-tasks, owns final response)
│   ├── Worker agent A   (e.g. research)
│   ├── Worker agent B   (e.g. code generation)
│   └── Worker agent C   (e.g. review)
└── Human participants   (observe or interject)
```

**Scope:**
- [ ] **Agent registry**: agents registered per room with a role (`supervisor`, `worker`, `observer`) and a tool set. Stored in Postgres `agents(room_id, bot_id, role, tool_scope)` (goose migration).
- [ ] **Handoff protocol**: a supervisor agent produces a structured `Handoff{target_agent_id, task, context_slice}` message type. The runtime delivers it to the target agent's `OnHandoff` handler and blocks the supervisor pending the result.
- [ ] **Shared memory graph**: a key-value scratchpad scoped to the room, readable and writable by all agents. Keys are namespaced by agent ID to prevent accidental collisions. Writes are last-write-wins; a vector clock is recorded per key for observability. **Agents should not write to the same key from concurrent dispatch branches** — the vector clock makes conflicts visible in traces but does not merge them.
- [ ] **Parallel execution**: supervisor can dispatch multiple handoffs simultaneously using `agent.Dispatch([]Handoff{})`, which fans out and collects results concurrently.
- [ ] **Agent traces**: every agent decision, tool call, handoff, and memory write is recorded in an append-only `traces` table (goose migration). Traces are queryable in real time by the developer dashboard.

**Key interfaces:**
```go
type AgentRoom interface {
    RegisterAgent(ctx context.Context, bot BotID, role AgentRole, tools []ToolName) error
    Handoff(ctx context.Context, from, to BotID, task HandoffTask) (HandoffResult, error)
    Dispatch(ctx context.Context, from BotID, tasks []HandoffTask) ([]HandoffResult, error)
}

type SharedMemory interface {
    Set(ctx context.Context, roomID RoomID, agentID BotID, key string, value any) error
    Get(ctx context.Context, roomID RoomID, key string) (any, error)
    List(ctx context.Context, roomID RoomID, prefix string) (map[string]any, error)
}
```

**Bot SDK surface:**
```go
// Supervisor agent
supervisor.OnMessage(func(ctx context.Context, msg *sdk.Message) {
    results, _ := supervisor.Dispatch(ctx, []sdk.HandoffTask{
        {To: "researcher", Task: "find recent case law on " + msg.Text},
        {To: "summariser", Task: "summarise the user's prior context"},
    })
    finalResponse := supervisor.Synthesise(ctx, results)
    msg.Reply(ctx, finalResponse)
})

// Worker agent
researcher.OnHandoff(func(ctx context.Context, task *sdk.HandoffTask) sdk.HandoffResult {
    docs := researcher.Tool("legal_search").Call(ctx, task.Task)
    return sdk.HandoffResult{Output: docs}
})
```

**Tests:**
- [ ] Supervisor dispatches 3 workers in parallel; verify all 3 execute concurrently (elapsed time < sum of individual times)
- [ ] Shared memory write conflict (two agents, same key, simultaneous): verify last-write-wins and vector clock records both writes
- [ ] Handoff chain (A → B → C): verify full trace captured and result propagates back to A correctly
- [ ] Agent with no role for a given tool scope cannot invoke that tool; verify rejection

**Definition of done:** A 3-agent room (supervisor + 2 workers) handles a multi-step task, writes intermediate results to shared memory, and the supervisor synthesises a coherent final response. All agent decisions are visible in the trace log.

---

### Stage 7 — Safety Plane

**Goal:** Every message — human or agent, inbound or outbound — passes through a composable safety pipeline. No message bypasses it. This stage replaces the no-op passthrough installed in Stage 0 with real filter implementations.

**Scope:**
- [ ] **Pipeline model**: safety runs as a Go middleware chain, called at the transport layer entry point (inbound) and inside the streaming engine (outbound, rolling window). Each filter returns `(modified_message, allow/block/redact, reason)`.
- [ ] **Inbound filters** (applied before context assembly):
  - PII detection and redaction (names, emails, phone numbers, credit card patterns) — regex + NER model
  - Prompt injection detection (adversarial instructions in user input attempting to hijack the bot)
  - Content policy (configurable per workspace: profanity, hate speech, NSFW)
- [ ] **Outbound filters** (applied per rolling token window during streaming; full-response filters applied at stream close):
  - PII leak detection in AI responses
  - Policy re-check on generated content
- [ ] **Audit log**: every safety decision (allow / block / redact) recorded in Postgres `safety_events(message_id, filter, action, reason, timestamp)` (goose migration). Queryable via REST API and developer dashboard.
- [ ] **Policy configuration**: per workspace, per room, per bot, with inheritance (room overrides workspace, bot overrides room).
- [ ] **Fail-open vs fail-closed**: configurable per workspace. Default: fail-closed (block on filter error).

**Key interfaces:**
```go
type SafetyPipeline interface {
    RunInbound(ctx context.Context, msg Message) (Message, SafetyVerdict, error)
    RunOutbound(ctx context.Context, msg Message) (Message, SafetyVerdict, error)
}

type SafetyFilter interface {
    Name() string
    Apply(ctx context.Context, msg Message, policy Policy) (Message, FilterAction, string)
}

type FilterAction int
const (
    Allow  FilterAction = iota
    Block
    Redact
)
```

**Tests:**
- [ ] Message containing a credit card number: verify PII redacted before delivery to LLM and before delivery to client
- [ ] Prompt injection attempt (`"Ignore previous instructions and..."`) detected and blocked inbound
- [ ] Outbound content policy violation blocked before reaching client
- [ ] Safety filter panic: verify fail-closed behaviour — message blocked, error logged, no panic propagation
- [ ] Policy override: room-level policy stricter than workspace-level; verify room policy takes precedence

**Definition of done:** All messages pass through the safety pipeline with real filter implementations. A message blocked inbound never reaches the context engine. A message blocked outbound never reaches the client. Every decision is in the audit log.

---

### Stage 8 — SDK & Developer Experience

**Goal:** The public SDK is polished, documented, and independently testable. Developers can integrate ConvKit without reading internal source.

**Scope:**
- [ ] `pkg/sdk` finalised with stable public API surface; all internal types unexported
- [ ] SDK connects to ConvKit server over WebSocket — same transport as human clients, authenticated with API key
- [ ] `sdk.NewBot`, `bot.OnMessage`, `bot.RegisterTool`, `bot.OnHandoff`, `bot.Start` fully implemented and tested in isolation against a test server
- [ ] `docs/sdk-guide.md`: getting started guide, tool registration, streaming, multi-agent patterns
- [ ] `examples/echo-bot`: minimal working bot, runnable with `go run . --api-key dev-key-1234`
- [ ] `examples/tool-bot`: bot with a registered tool that calls a live HTTP endpoint
- [ ] `examples/multi-agent`: supervisor + 2 workers completing a multi-step task

**Tests:**
- [ ] SDK integration test: echo bot connects, sends message, receives streamed reply — tested against embedded test server, no external dependencies
- [ ] Tool registration and invocation round-trip via SDK
- [ ] SDK reconnect: server restarts mid-stream; SDK reconnects and receives replay

**Definition of done:** A developer can clone the repo, run `make dev`, and have a working bot connected and responding in under 10 minutes using only `docs/sdk-guide.md`.

---

## Observability

ConvKit emits structured telemetry from every layer. No silent failures.

| Signal | Library | What's captured |
|---|---|---|
| Structured logs | `github.com/rs/zerolog` | Every request, routing decision, safety event, error |
| Metrics | `prometheus/client_golang` | Latency histograms, message throughput, streaming token rate, tool execution time, safety block rate |
| Distributed traces | `go.opentelemetry.io/otel` | Full trace per message from transport to delivery, including LLM call and tool execution spans |
| Admin dashboard | Built-in HTTP handler | Live room list, agent trace viewer, safety event log, token budget visualiser |

All signals are compatible with the standard LGTM stack (Grafana + Loki + Tempo) and Datadog.

---

## Key Design Decisions

| Decision | V1 Choice | Rationale | Future Option |
|---|---|---|---|
| Deployment topology | Single binary | Eliminates distributed systems complexity before it's needed; clean package boundaries make future splitting straightforward | Split coordinator + shard nodes when load demands horizontal room scaling |
| Message fan-out | NATS JetStream | Low operational overhead, at-least-once, built-in replay | Kafka for higher throughput at scale |
| Streaming protocol | SSE delta over WebSocket | Simpler reconnect semantics; HTTP/2 framing handles flow control | WebTransport when browser support matures |
| Tokenizer | Pluggable interface, per-model implementations | Model-agnostic; tiktoken-go for OpenAI models, character estimate as fallback; `ModelID` selects at runtime | First-party tokenizer APIs as they become available |
| Context window mgmt | Token-counting + recency | Deterministic, model-agnostic | Learned context compression |
| Tool execution | Goroutine per call + timeout | Go's goroutine model is near-zero overhead; provides cancellation isolation for trusted handlers | Remote HTTP endpoints for untrusted tools; WASM sandbox as a future option |
| Memory graph conflicts | Last-write-wins + vector clock | Simple to implement correctly; vector clock makes conflicts visible in traces | CRDTs for true concurrent merge |
| Multi-agent routing | Explicit handoff (pull) | Auditable, no surprise concurrency | Event-driven (push) for reactive agents |
| Safety pipeline | Middleware chain, fail-closed, no-op from Stage 0 | Composable, testable; every layer plugs in from day one | ML-based policy scoring |
| LLM provider | Provider-agnostic interface | Apps bring their own LLM | Optional hosted inference via partner API |
| Database migrations | goose | Versioned, auditable schema changes applied on startup | — |

---

## Running Locally

```bash
# Start the full stack (server + Redis + Postgres + NATS)
make dev

# Run all tests
make test

# Run benchmarks
make bench

# Connect the example echo bot
cd examples/echo-bot && go run . --api-key dev-key-1234

# Open the developer dashboard
open http://localhost:8080/dashboard
```

---

## Non-Goals (V1)

- **Horizontal scaling** — V1 is a single-process server. Vertical scaling handles growth until a multi-node architecture is actually warranted.
- **Multi-tenancy isolation at the infrastructure level** — V1 uses workspace IDs as logical isolation; hard infrastructure isolation (separate clusters per tenant) is a V2 concern.
- **On-premise LLM hosting** — ConvKit routes to any provider-agnostic LLM interface; it does not bundle model weights.
- **Voice / audio transport** — text and structured data only in V1.
- **Mobile SDKs (iOS / Android)** — V1 ships a Go SDK and a REST/WebSocket protocol spec. Community-maintained mobile SDKs can be built against the protocol.
- **Billing / token metering** — usage tracking per workspace is distinct from the token budget feature and is a V2 concern.

---

## References

- [NATS JetStream docs](https://docs.nats.io/nats-concepts/jetstream)
- [coder/websocket — maintained Go WebSocket library](https://github.com/coder/websocket)
- [pgvector — vector similarity in Postgres](https://github.com/pgvector/pgvector)
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)
- [goose — database migration tool](https://github.com/pressly/goose)
- [JSON Schema — tool argument validation](https://json-schema.org/)