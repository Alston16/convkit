# Pattern: Layer Interface

## Intent

Each of ConvKit's six layers exposes a single Go interface that defines its contract. All cross-layer calls go through the interface, never directly to a concrete type. This keeps layers independently testable and allows the implementation to be swapped (e.g. adding a network boundary for horizontal scaling) without touching callers.

## When to Use

- Implementing any of the six layers (Transport, Bus, Stream, Context, Tools, Orchestration)
- Writing a new component that is called from another layer

## Template

```go
// internal/<layer>/<layer>.go

package <layer>

import (
    "context"
    "github.com/Alston16/convkit/internal/common"
)

// <Layer> defines the contract for Layer N.
// All callers depend on this interface, never on the concrete type.
type <Layer> interface {
    // Primary operation
    <Method>(ctx context.Context, <args>) (<return>, error)
}

// Config holds all external dependencies injected at construction.
type Config struct {
    // e.g. DB *pgxpool.Pool, Redis *redis.Client, Safety safety.SafetyPipeline
}

type service struct {
    cfg Config
    // unexported fields
}

// New constructs the layer implementation with its dependencies.
func New(cfg Config) <Layer> {
    return &service{cfg: cfg}
}

func (s *service) <Method>(ctx context.Context, <args>) (<return>, error) {
    // 1. Call safety.RunInbound / RunOutbound if this is a message-handling layer
    // 2. Core logic
    // 3. Wrap errors: fmt.Errorf("<layer>.<Method>: %w", err)
}
```

## Established Layer Interfaces

### Layer 1 — Transport

```go
type Transport interface {
    RegisterHandler(path string, h MessageHandler)
    ListenAndServe(ctx context.Context, addr string) error
}
type MessageHandler func(ctx context.Context, conn Connection, msg RawMessage) error
```

### Layer 2 — Bus

```go
type Bus interface {
    Publish(ctx context.Context, roomID common.RoomID, msg common.Message) error
    Subscribe(ctx context.Context, roomID common.RoomID) (<-chan common.Message, error)
}

type PresenceService interface {
    Heartbeat(ctx context.Context, roomID common.RoomID, userID common.UserID) error
    Online(ctx context.Context, roomID common.RoomID) ([]common.UserID, error)
    SetTyping(ctx context.Context, roomID common.RoomID, userID common.UserID) error
}
```

### Layer 3 — Streaming Engine

```go
type Streamer interface {
    OpenStream(ctx context.Context, replyTo common.MessageID) (StreamWriter, error)
}

type StreamWriter interface {
    Write(delta string) error  // blocks on backpressure
    Close() error
}
```

### Layer 4 — Context Engine

```go
type ContextEngine interface {
    Assemble(ctx context.Context, opts AssembleOpts) ([]LLMMessage, error)
}

type AssembleOpts struct {
    RoomID        common.RoomID
    BotID         common.BotID
    UserMessage   common.Message
    ModelID       string
    MaxTokens     int
    ReserveOutput int
    RAGQuery      string  // empty = skip RAG
}
```

### Layer 5 — Tool Runtime

```go
type ToolRegistry interface {
    Register(ctx context.Context, tool ToolDefinition) error
    Resolve(ctx context.Context, workspaceID, name string) (ToolDefinition, error)
}

type ToolRuntime interface {
    Execute(ctx context.Context, call common.ToolCall, userCtx UserContext) (common.ToolResult, error)
    ExecuteStream(ctx context.Context, call common.ToolCall, userCtx UserContext) (<-chan ToolProgress, error)
}
```

### Layer 6 — Agent Orchestration

```go
type AgentRoom interface {
    RegisterAgent(ctx context.Context, bot common.BotID, role AgentRole, tools []ToolName) error
    Handoff(ctx context.Context, from, to common.BotID, task HandoffTask) (HandoffResult, error)
    Dispatch(ctx context.Context, from common.BotID, tasks []HandoffTask) ([]HandoffResult, error)
}

type SharedMemory interface {
    Set(ctx context.Context, roomID common.RoomID, agentID common.BotID, key string, value any) error
    Get(ctx context.Context, roomID common.RoomID, key string) (any, error)
    List(ctx context.Context, roomID common.RoomID, prefix string) (map[string]any, error)
}
```

### Safety Plane

```go
type SafetyPipeline interface {
    RunInbound(ctx context.Context, msg common.Message) (common.Message, SafetyVerdict, error)
    RunOutbound(ctx context.Context, msg common.Message) (common.Message, SafetyVerdict, error)
}
```

## Rules

1. Concrete structs are always unexported. Callers only hold the interface type.
2. The `New(cfg Config) Interface` constructor pattern is used for all layers.
3. Error wrapping format: `"<package>.<Method>: %w"`.
4. The safety plane is always a field on the `Config` struct for message-handling layers — never obtained via a global.
