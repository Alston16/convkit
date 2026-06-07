package context

import (
	"context"
	"fmt"

	"github.com/Alston16/convkit/internal/common"
	"github.com/Alston16/convkit/internal/safety"
)

// LLMMessage is a single message in the assembled context window sent to the LLM.
type LLMMessage struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// AssembleOpts controls context window assembly.
type AssembleOpts struct {
	RoomID        common.RoomID
	BotID         common.BotID
	UserMessage   common.Message
	ModelID       string // selects the Tokenizer implementation
	MaxTokens     int
	ReserveOutput int
	RAGQuery      string // empty = skip RAG
}

// ContextEngine is the Layer 4 interface for assembling LLM context windows.
type ContextEngine interface {
	Assemble(ctx context.Context, opts AssembleOpts) ([]LLMMessage, error)
}

// Config holds the external dependencies for the context layer.
type Config struct {
	Safety safety.SafetyPipeline
}

type service struct {
	cfg Config
}

// New constructs the ContextEngine implementation.
func New(cfg Config) ContextEngine {
	return &service{cfg: cfg}
}

func (s *service) Assemble(_ context.Context, _ AssembleOpts) ([]LLMMessage, error) {
	// Stub — implemented in Stage 4.
	return nil, fmt.Errorf("context.Assemble: not implemented")
}
