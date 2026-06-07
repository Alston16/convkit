---
applyTo: "internal/**/*.go,pkg/**/*.go,cmd/**/*.go"
---

# ConvKit — Backend Development Guidelines

## Module & Package Conventions

- To add a dependency, always use `go get <module>@<version>` — never edit `go.mod` directly.
- Module: `github.com/Alston16/convkit`
- Internal packages live in `internal/`. Public API lives in `pkg/sdk` and `pkg/tokenizer`.
- Each layer owns exactly one `internal/` package: `transport`, `bus`, `stream`, `context`, `tools`, `orchestration`, `safety`.
- Cross-layer calls are **direct Go function calls** — no RPC, no channels across package boundaries unless explicitly required by the interface contract.

## Safety Plane — Non-Negotiable

Every inbound message **must** call `safety.RunInbound` before reaching the context engine.  
Every outbound token stream **must** call `safety.RunOutbound` on a rolling window.  
This applies from Stage 0 onwards, even when the implementation is a no-op passthrough.

```go
// Always required — never skip safety plane calls
msg, verdict, err := s.safety.RunInbound(ctx, msg)
if verdict.Action == safety.Block {
    return ErrBlocked
}
```

## Database Migrations

- All schema changes go in `migrations/` as goose `.sql` files.
- Never modify a committed migration — add a new one.
- Migrations are applied automatically on startup.

## Error Handling

- Use sentinel errors for domain conditions (`ErrBlocked`, `ErrToolTimeout`, etc.).
- Wrap errors with context: `fmt.Errorf("bus.Publish: %w", err)`
- Never swallow errors from safety plane, tool runtime, or database writes.

## Streaming

- Use `io.Pipe` for streaming — LLM goroutine writes, streaming engine goroutine reads.
- Never buffer unboundedly. Slow clients must **block the writer** (backpressure), not drop tokens.
- Store tokens in Redis sorted set `stream:{messageID}:tokens` (score = index, TTL 5 min) to enable reconnect replay.

## Tool Execution

- Each tool call runs in a **dedicated goroutine** with a configurable timeout (default 10s) and context cancellation.
- Always validate tool arguments against the registered JSON Schema **before** calling the handler.
- Untrusted tool handlers should be remote HTTP endpoints, not local Go functions.

## Context Window Assembly

- Always select the `Tokenizer` implementation using `ModelID` — never hardcode a tokeniser.
- Persona system prompt is **always at position 0**, regardless of window truncation.
- Never exceed `max_tokens - reserved_output_tokens`. Assert this in tests.

## Authentication

- JWT: HS256 for development, RS256 for production.
- API keys: store only the hash in Postgres, never the raw key.
- Apply rate limiting per-connection using a token bucket (`golang.org/x/time/rate`).

## Observability

- Use `rs/zerolog` for all structured logging. Never use `fmt.Println` or `log.Printf` in production paths.
- Emit an OpenTelemetry span for every message from transport entry to delivery.
- Expose Prometheus metrics for: latency histograms, message throughput, streaming token rate, tool execution time, safety block rate.

## Testing

- `go test ./...` must always pass.
- Each layer must have unit tests against its interface (not internal implementation).
- Integration tests use the embedded Docker Compose stack (`make dev`).
- Token count accuracy tests: assert within ±5% of actual model tokeniser output per implementation.
