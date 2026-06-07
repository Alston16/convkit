# ConvKit — Staging Plan

Eight stages. Each stage ships a working, testable artifact. The safety plane is introduced as a **no-op passthrough in Stage 0** so every layer plugs into it from day one.

---

## Stage 0 — Project Bootstrap
**Goal:** Working repository, shared types, and a safety plane skeleton.

**Key deliverables:**
- Go module: `github.com/Alston16/convkit`
- Core domain types in `internal/common`: `RoomID`, `UserID`, `BotID`, `MessageID`, `Message`, `StreamToken`, `ToolCall`, `ToolResult`
- Safety plane skeleton in `internal/safety`: `SafetyPipeline` interface; default = no-op passthrough
- Pluggable tokenizer in `pkg/tokenizer`: `Tokenizer` interface; OpenAI (tiktoken-go) and character-estimate fallback
- `goose` configured; baseline empty schema migration
- Docker Compose environment; `make dev` / `make test`

**Done when:** `go test ./...` passes. Health endpoint returns `200 OK`. Safety no-op callable from all layer stubs.

---

## Stage 1 — Layer 1: Transport
**Goal:** Clients connect over WebSocket and REST. Auth and rate limiting enforced.

**Key deliverables:**
- WebSocket handler (`github.com/coder/websocket`) — one goroutine per connection
- REST API: room CRUD, message history, bot registration
- Auth middleware: JWT (HS256 dev / RS256 prod) + API key (hashed in Postgres)
- Per-connection rate limiter: token bucket (`golang.org/x/time/rate`)
- All inbound messages call `safety.RunInbound` (no-op)

**Done when:** Test client connects over WebSocket, sends a message, receives an echo. Unauthenticated connections rejected.

---

## Stage 2 — Layer 2: Message Bus
**Goal:** Messages durably stored and fanned out to all room subscribers, with presence and ordering.

**Key deliverables:**
- NATS JetStream subjects: `rooms.{roomID}.messages`, `rooms.{roomID}.presence`
- Postgres `messages` table: `(room_id, seq_num, sender_id, body, created_at)`
- Fan-out: server → NATS → subscriber goroutine per room → WebSocket clients
- Presence: Redis `SETEX rooms:{roomID}:users:{userID} 30`; heartbeat every 10s
- Typing indicators: ephemeral NATS, no persistence, TTL 3s
- Monotonic `seq_num` per room

**Done when:** Two WebSocket clients in the same room exchange messages. Presence correctly reflects online/offline.

---

## Stage 3 — Layer 3: Streaming Engine
**Goal:** AI responses stream token-by-token. Clients reconnect mid-stream without losing tokens.

**Key deliverables:**
- Delta protocol: `StreamToken{messageID, delta, index, done}` — only delta transmitted
- `io.Pipe`-based streaming with backpressure (slow client blocks writer, no unbounded buffering)
- Reconnect with replay: Redis sorted set `stream:{messageID}:tokens` (score = index, TTL 5 min); on reconnect client sends `last_index`
- All outbound tokens call `safety.RunOutbound` rolling window (no-op)

**Done when:** Bot sends streaming response. Client that disconnects mid-stream and reconnects receives the complete message without gaps.

---

## Stage 4 — Layer 4: Context Engine
**Goal:** Every LLM call receives the correct, token-budget-aware context slice.

**Window assembly algorithm:**
1. Fetch conversation history (Postgres), most-recent-first
2. Tokenise each message via pluggable `Tokenizer` (selected by `ModelID`)
3. Fill window backwards until `max_tokens - reserved_output_tokens`
4. Inject persona system prompt at position 0 (always included)
5. If long-term memory enabled: embed current message → retrieve top-K past exchanges → inject as `<memory>` block
6. If RAG enabled: retrieve top-K documents → inject as `<context>` block

**Memory tiers:** Working (Redis, session), Conversation (Postgres, configurable TTL), Long-term (pgvector/Qdrant, permanent)

**Done when:** Bot receives context that fits the model's token limit, contains persona, and includes relevant past exchanges when long-term memory is enabled.

---

## Stage 5 — Layer 5: Tool Runtime
**Goal:** Bots declare callable tools. Runtime validates, executes with timeout isolation, injects results back — automatically.

**Key deliverables:**
- Schema registry: `tools(workspace_id, name, description, schema, endpoint, auth_config)` in Postgres
- Timeout-isolated execution: dedicated goroutine per call, configurable timeout (default 10s), context cancellation
- LLM loop: `tool_call` → execute → inject `tool_result` → re-call LLM → repeat until final text
- Streaming tool results: `ToolProgress` events to client in real time
- Tool auth delegation: OAuth token forwarding to tool from end-user context

**Note:** Goroutine isolation provides timeout/cancellation only. Untrusted handlers should be remote HTTP endpoints.

**Done when:** Bot with registered tool automatically executes it when LLM requests it, receives result, produces coherent final response — no application-level loop management needed.

---

## Stage 6 — Layer 6: Agent Orchestration
**Goal:** Multiple agents collaborate in a single room with defined roles, shared memory, and observable traces.

**Topology:** Supervisor → Worker A, Worker B, Worker C ← Human participants

**Key deliverables:**
- Agent registry: `agents(room_id, bot_id, role, tool_scope)` in Postgres
- Handoff protocol: `Handoff{target_agent_id, task, context_slice}` — supervisor blocks pending worker result
- Shared memory graph: room-scoped key-value scratchpad, namespaced by agent ID, last-write-wins + vector clock
- Parallel dispatch: `agent.Dispatch([]Handoff{})` fans out, collects results concurrently
- Append-only `traces` table: every agent decision, tool call, handoff, memory write

**Done when:** 3-agent room (supervisor + 2 workers) handles multi-step task. All decisions visible in trace log.

---

## Stage 7 — Safety Plane
**Goal:** Every message passes through a composable safety pipeline. No message bypasses it.

**Filters:**
- **Inbound**: PII detection/redaction (regex + NER), prompt injection detection, content policy
- **Outbound**: PII leak detection in AI responses, policy re-check (rolling window + full-response at stream close)

**Key deliverables:**
- Audit log: `safety_events(message_id, filter, action, reason, timestamp)` in Postgres
- Policy config: per workspace / room / bot with inheritance (bot > room > workspace)
- Fail-open vs fail-closed: configurable, default = **fail-closed**

**Done when:** Messages blocked inbound never reach the context engine. Messages blocked outbound never reach the client. Every decision is in the audit log.

---

## Stage 8 — SDK & Developer Experience
**Goal:** Public SDK polished, documented, independently testable. Developer can be productive in under 10 minutes.

**Key deliverables:**
- `pkg/sdk` stable public API; all internal types unexported
- SDK connects via WebSocket, authenticated with API key
- `docs/sdk-guide.md` getting started guide
- `examples/echo-bot`, `examples/tool-bot`, `examples/multi-agent`

**Done when:** Developer clones, runs `make dev`, has working bot responding using only `docs/sdk-guide.md` in under 10 minutes.
