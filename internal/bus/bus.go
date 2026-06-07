package bus

import (
	"context"
	"fmt"

	"github.com/Alston16/convkit/internal/common"
	"github.com/Alston16/convkit/internal/safety"
)

// Bus is the Layer 2 interface for message publishing and fan-out.
type Bus interface {
	Publish(ctx context.Context, roomID common.RoomID, msg common.Message) error
	Subscribe(ctx context.Context, roomID common.RoomID) (<-chan common.Message, error)
}

// PresenceService tracks which users are online in a room and typing state.
type PresenceService interface {
	Heartbeat(ctx context.Context, roomID common.RoomID, userID common.UserID) error
	Online(ctx context.Context, roomID common.RoomID) ([]common.UserID, error)
	SetTyping(ctx context.Context, roomID common.RoomID, userID common.UserID) error
}

// Config holds the external dependencies for the bus layer.
type Config struct {
	Safety safety.SafetyPipeline
}

type service struct {
	cfg Config
}

// New constructs the Bus implementation.
func New(cfg Config) Bus {
	return &service{cfg: cfg}
}

func (s *service) Publish(_ context.Context, _ common.RoomID, _ common.Message) error {
	// Stub — implemented in Stage 2.
	return fmt.Errorf("bus.Publish: not implemented")
}

func (s *service) Subscribe(_ context.Context, _ common.RoomID) (<-chan common.Message, error) {
	// Stub — implemented in Stage 2.
	return nil, fmt.Errorf("bus.Subscribe: not implemented")
}
