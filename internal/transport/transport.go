package transport

import (
	"context"
	"fmt"

	"github.com/Alston16/convkit/internal/common"
	"github.com/Alston16/convkit/internal/safety"
)

// RawMessage is the raw bytes received from a connected client.
type RawMessage []byte

// Connection represents an active client connection.
type Connection interface {
	Send(ctx context.Context, data []byte) error
	Close() error
}

// MessageHandler is the callback invoked for each inbound message.
type MessageHandler func(ctx context.Context, conn Connection, msg RawMessage) error

// Transport is the Layer 1 interface for accepting client connections.
type Transport interface {
	RegisterHandler(path string, h MessageHandler)
	ListenAndServe(ctx context.Context, addr string) error
}

// Config holds the external dependencies for the transport layer.
type Config struct {
	Safety safety.SafetyPipeline
}

type service struct {
	cfg      Config
	handlers map[string]MessageHandler
}

// New constructs the Transport implementation.
func New(cfg Config) Transport {
	return &service{
		cfg:      cfg,
		handlers: make(map[string]MessageHandler),
	}
}

func (s *service) RegisterHandler(path string, h MessageHandler) {
	s.handlers[path] = h
}

func (s *service) ListenAndServe(_ context.Context, _ string) error {
	// Stub — implemented in Stage 1.
	return fmt.Errorf("transport.ListenAndServe: not implemented")
}

// processInbound demonstrates the safety plane call pattern for inbound messages.
// Real WebSocket/REST handlers will call this before any processing.
func (s *service) processInbound(ctx context.Context, msg common.Message) (common.Message, error) {
	safe, verdict, err := s.cfg.Safety.RunInbound(ctx, msg)
	if err != nil {
		return common.Message{}, fmt.Errorf("transport.processInbound: %w", err)
	}
	if verdict.Action == common.Block {
		return common.Message{}, ErrBlocked
	}
	return safe, nil
}

// ErrBlocked is returned when the safety plane blocks a message.
var ErrBlocked = fmt.Errorf("message blocked by safety plane")
