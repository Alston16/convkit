package stream

import (
	"context"
	"fmt"

	"github.com/Alston16/convkit/internal/common"
	"github.com/Alston16/convkit/internal/safety"
)

// StreamWriter is returned by OpenStream. The bot SDK writes tokens to it.
// Write blocks on backpressure — slow clients block the writer rather than
// allowing unbounded buffering.
type StreamWriter interface {
	Write(delta string) error
	Close() error
}

// Streamer is the Layer 3 interface for opening response streams.
type Streamer interface {
	OpenStream(ctx context.Context, replyTo common.MessageID) (StreamWriter, error)
}

// Config holds the external dependencies for the streaming layer.
type Config struct {
	Safety safety.SafetyPipeline
}

type service struct {
	cfg Config
}

// New constructs the Streamer implementation.
func New(cfg Config) Streamer {
	return &service{cfg: cfg}
}

func (s *service) OpenStream(_ context.Context, _ common.MessageID) (StreamWriter, error) {
	// Stub — implemented in Stage 3.
	return nil, fmt.Errorf("stream.OpenStream: not implemented")
}

// processOutbound demonstrates the safety plane call pattern for outbound tokens.
// The real streaming engine calls this per-token on a rolling window.
func (s *service) processOutbound(ctx context.Context, token common.StreamToken) (common.StreamToken, error) {
	safe, verdict, err := s.cfg.Safety.RunOutbound(ctx, token)
	if err != nil {
		return common.StreamToken{}, fmt.Errorf("stream.processOutbound: %w", err)
	}
	if verdict.Action == common.Block {
		return common.StreamToken{}, ErrStreamBlocked
	}
	return safe, nil
}

// ErrStreamBlocked is returned when the safety plane blocks an outbound token.
var ErrStreamBlocked = fmt.Errorf("token blocked by safety plane")
