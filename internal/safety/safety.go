package safety

import (
	"context"

	"github.com/Alston16/convkit/internal/common"
)

// SafetyPipeline is the bidirectional middleware contract for the safety plane.
// All layers call RunInbound before processing a message and RunOutbound before
// delivering tokens to the client. In Stage 0 the only implementation is the
// no-op passthrough; real filters are added in Stage 7.
type SafetyPipeline interface {
	// RunInbound is called at transport entry before the context engine.
	// Returns the (possibly modified) message and a verdict. If verdict.Action
	// is Block, the caller must not process the message further.
	RunInbound(ctx context.Context, msg common.Message) (common.Message, common.SafetyVerdict, error)

	// RunOutbound is called by the streaming engine on a rolling token window.
	// Returns the (possibly modified) token and a verdict. If verdict.Action is
	// Block, the stream must be terminated.
	RunOutbound(ctx context.Context, token common.StreamToken) (common.StreamToken, common.SafetyVerdict, error)
}

// noop is the no-op passthrough implementation. It approves every message and
// token without modification. This is the default for Stages 0–6.
type noop struct{}

// NewNoop returns a SafetyPipeline that passes everything through unchanged.
func NewNoop() SafetyPipeline {
	return &noop{}
}

func (n *noop) RunInbound(_ context.Context, msg common.Message) (common.Message, common.SafetyVerdict, error) {
	return msg, common.SafetyVerdict{Action: common.Allow}, nil
}

func (n *noop) RunOutbound(_ context.Context, token common.StreamToken) (common.StreamToken, common.SafetyVerdict, error) {
	return token, common.SafetyVerdict{Action: common.Allow}, nil
}
