package common

import (
	"encoding/json"
	"time"
)

// --- ID types ---

type RoomID string
type UserID string
type BotID string
type MessageID string

// --- Core message types ---

type Message struct {
	ID        MessageID
	RoomID    RoomID
	SenderID  string // UserID or BotID
	Body      string
	CreatedAt time.Time
	Metadata  map[string]any
}

type StreamToken struct {
	MessageID MessageID
	Delta     string // only the new characters, not accumulated text
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

// --- Multi-agent types ---

type AgentRole string

const (
	AgentRoleSupervisor AgentRole = "supervisor"
	AgentRoleWorker     AgentRole = "worker"
	AgentRoleObserver   AgentRole = "observer"
)

type HandoffTask struct {
	To      BotID
	Task    string
	Context any
}

type HandoffResult struct {
	Output any
	Error  string
}

// --- Safety types ---

type FilterAction int

const (
	Allow FilterAction = iota
	Block
	Redact
)

type SafetyVerdict struct {
	Action FilterAction
	Reason string
	Filter string
}
