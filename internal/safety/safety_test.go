package safety_test

import (
	"context"
	"testing"
	"time"

	"github.com/Alston16/convkit/internal/common"
	"github.com/Alston16/convkit/internal/safety"
)

func TestNoopRunInbound_AllowsMessage(t *testing.T) {
	sp := safety.NewNoop()

	original := common.Message{
		ID:        "msg-1",
		RoomID:    "room-1",
		SenderID:  "user-1",
		Body:      "hello",
		CreatedAt: time.Now(),
	}

	result, verdict, err := sp.RunInbound(context.Background(), original)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Action != common.Allow {
		t.Fatalf("expected Allow, got %v", verdict.Action)
	}
	if result.ID != original.ID || result.Body != original.Body {
		t.Fatalf("message was modified unexpectedly: got %+v", result)
	}
}

func TestNoopRunOutbound_AllowsToken(t *testing.T) {
	sp := safety.NewNoop()

	original := common.StreamToken{
		MessageID: "msg-1",
		Delta:     "hello",
		Index:     0,
		Done:      false,
	}

	result, verdict, err := sp.RunOutbound(context.Background(), original)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Action != common.Allow {
		t.Fatalf("expected Allow, got %v", verdict.Action)
	}
	if result.Delta != original.Delta || result.Index != original.Index {
		t.Fatalf("token was modified unexpectedly: got %+v", result)
	}
}
