package orchestration

import (
	"context"
	"fmt"

	"github.com/Alston16/convkit/internal/common"
	"github.com/Alston16/convkit/internal/safety"
)

// AgentRoom is the Layer 6 interface for multi-agent room management.
type AgentRoom interface {
	RegisterAgent(ctx context.Context, bot common.BotID, role common.AgentRole, tools []string) error
	Handoff(ctx context.Context, from, to common.BotID, task common.HandoffTask) (common.HandoffResult, error)
	Dispatch(ctx context.Context, from common.BotID, tasks []common.HandoffTask) ([]common.HandoffResult, error)
}

// SharedMemory is the Layer 6 interface for the agent scratchpad.
type SharedMemory interface {
	Set(ctx context.Context, roomID common.RoomID, agentID common.BotID, key string, value any) error
	Get(ctx context.Context, roomID common.RoomID, key string) (any, error)
	List(ctx context.Context, roomID common.RoomID, prefix string) (map[string]any, error)
}

// Config holds the external dependencies for the orchestration layer.
type Config struct {
	Safety safety.SafetyPipeline
}

type service struct {
	cfg Config
}

// New constructs the AgentRoom and SharedMemory implementation.
func New(cfg Config) (AgentRoom, SharedMemory) {
	s := &service{cfg: cfg}
	return s, s
}

func (s *service) RegisterAgent(_ context.Context, _ common.BotID, _ common.AgentRole, _ []string) error {
	// Stub — implemented in Stage 6.
	return fmt.Errorf("orchestration.RegisterAgent: not implemented")
}

func (s *service) Handoff(_ context.Context, _, _ common.BotID, _ common.HandoffTask) (common.HandoffResult, error) {
	// Stub — implemented in Stage 6.
	return common.HandoffResult{}, fmt.Errorf("orchestration.Handoff: not implemented")
}

func (s *service) Dispatch(_ context.Context, _ common.BotID, _ []common.HandoffTask) ([]common.HandoffResult, error) {
	// Stub — implemented in Stage 6.
	return nil, fmt.Errorf("orchestration.Dispatch: not implemented")
}

func (s *service) Set(_ context.Context, _ common.RoomID, _ common.BotID, _ string, _ any) error {
	// Stub — implemented in Stage 6.
	return fmt.Errorf("orchestration.Set: not implemented")
}

func (s *service) Get(_ context.Context, _ common.RoomID, _ string) (any, error) {
	// Stub — implemented in Stage 6.
	return nil, fmt.Errorf("orchestration.Get: not implemented")
}

func (s *service) List(_ context.Context, _ common.RoomID, _ string) (map[string]any, error) {
	// Stub — implemented in Stage 6.
	return nil, fmt.Errorf("orchestration.List: not implemented")
}
