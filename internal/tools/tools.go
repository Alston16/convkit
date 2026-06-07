package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Alston16/convkit/internal/common"
	"github.com/Alston16/convkit/internal/safety"
)

// ToolDefinition is a registered tool with its JSON Schema and handler config.
type ToolDefinition struct {
	WorkspaceID string
	Name        string
	Description string
	Schema      json.RawMessage
}

// UserContext carries the end-user's credentials for tool auth delegation.
type UserContext struct {
	UserID common.UserID
	Token  string
}

// ToolProgress is an incremental update emitted by a streaming tool.
type ToolProgress struct {
	CallID  string
	Message string
	Done    bool
}

// ToolRegistry is the Layer 5 interface for registering and resolving tools.
type ToolRegistry interface {
	Register(ctx context.Context, tool ToolDefinition) error
	Resolve(ctx context.Context, workspaceID, name string) (ToolDefinition, error)
}

// ToolRuntime is the Layer 5 interface for executing tool calls.
type ToolRuntime interface {
	Execute(ctx context.Context, call common.ToolCall, userCtx UserContext) (common.ToolResult, error)
	ExecuteStream(ctx context.Context, call common.ToolCall, userCtx UserContext) (<-chan ToolProgress, error)
}

// Config holds the external dependencies for the tools layer.
type Config struct {
	Safety safety.SafetyPipeline
}

type service struct {
	cfg Config
}

// New constructs the ToolRegistry and ToolRuntime implementation.
func New(cfg Config) (ToolRegistry, ToolRuntime) {
	s := &service{cfg: cfg}
	return s, s
}

func (s *service) Register(_ context.Context, _ ToolDefinition) error {
	// Stub — implemented in Stage 5.
	return fmt.Errorf("tools.Register: not implemented")
}

func (s *service) Resolve(_ context.Context, _, _ string) (ToolDefinition, error) {
	// Stub — implemented in Stage 5.
	return ToolDefinition{}, fmt.Errorf("tools.Resolve: not implemented")
}

func (s *service) Execute(_ context.Context, _ common.ToolCall, _ UserContext) (common.ToolResult, error) {
	// Stub — implemented in Stage 5.
	return common.ToolResult{}, fmt.Errorf("tools.Execute: not implemented")
}

func (s *service) ExecuteStream(_ context.Context, _ common.ToolCall, _ UserContext) (<-chan ToolProgress, error) {
	// Stub — implemented in Stage 5.
	return nil, fmt.Errorf("tools.ExecuteStream: not implemented")
}
