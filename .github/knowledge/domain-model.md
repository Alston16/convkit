# ConvKit — Domain Model

## Core Types (`internal/common`)

```go
type RoomID    string
type UserID    string
type BotID     string
type MessageID string

type Message struct {
    ID        MessageID
    RoomID    RoomID
    SenderID  string        // UserID or BotID
    Body      string
    CreatedAt time.Time
    Metadata  map[string]any
}

type StreamToken struct {
    MessageID MessageID
    Delta     string  // only the new characters, not accumulated text
    Index     int
    Done      bool
}

type ToolCall struct {
    ID        string
    Name      string
    Arguments json.RawMessage
}

type ToolResult struct {
    CallID string
    Output json.RawMessage
    Error  string
}
```

## Multi-Agent Types

```go
type AgentRole string  // "supervisor" | "worker" | "observer"

type HandoffTask struct {
    To      BotID
    Task    string
    Context any
}

type HandoffResult struct {
    Output any
    Error  string
}
```

## Safety Types

```go
type FilterAction int
const (
    Allow  FilterAction = iota
    Block
    Redact
)

type SafetyVerdict struct {
    Action FilterAction
    Reason string
    Filter string
}
```

## Memory Tiers (Context Engine)

| Tier | Storage | Scope | TTL |
|---|---|---|---|
| Working memory | Redis | Current session | Session lifetime |
| Conversation memory | PostgreSQL | Conversation lifetime | Configurable (default: 30 days) |
| Long-term memory | pgvector / Qdrant | Cross-conversation semantic retrieval | Permanent |

## Database Tables

| Table | Layer | Description |
|---|---|---|
| `messages` | Bus (L2) | `(room_id, seq_num, sender_id, body, created_at)` |
| `tools` | Tool Runtime (L5) | `(workspace_id, name, description, schema, endpoint, auth_config)` |
| `agents` | Orchestration (L6) | `(room_id, bot_id, role, tool_scope)` |
| `traces` | Orchestration (L6) | Append-only agent decision/handoff/memory event log |
| `safety_events` | Safety Plane (L7) | `(message_id, filter, action, reason, timestamp)` |

All schema changes managed via **goose** migrations in `migrations/`.

## Redis Key Patterns

| Key | Purpose |
|---|---|
| `rooms:{roomID}:users:{userID}` | Presence (SETEX, TTL 30s, heartbeat every 10s) |
| `stream:{messageID}:tokens` | Sorted set of stream tokens for replay (TTL 5 min) |

## NATS JetStream Subjects

| Subject | Purpose |
|---|---|
| `rooms.{roomID}.messages` | Message fan-out to WebSocket subscribers |
| `rooms.{roomID}.presence` | Presence and typing indicator events |

## Terminology

- **Room**: The unit of conversation. Contains human participants and agent participants.
- **Bot / Agent**: An AI participant registered to a workspace. Bots own a tool set and respond via the SDK.
- **Workspace**: The top-level tenant. Auth, policy, and tool registrations are scoped to workspaces.
- **Handoff**: A structured delegation from a supervisor agent to a worker agent.
- **Delta**: A single streaming token increment (not the full accumulated text).
- **Safety plane**: Bidirectional middleware that runs on every message at every layer.
- **Context window**: The token-budget-capped slice of history + persona + RAG content sent to the LLM.
