package event

import (
	"encoding/json"
	"testing"
)

func TestNewMessageClaimIDIsStableAndMessageSpecific(t *testing.T) {
	t.Parallel()

	first, err := NewMessageClaimID(json.RawMessage(`{"message_id":"64b7f2c1a3e5d7890abc5678"}`))
	if err != nil {
		t.Fatalf("NewMessageClaimID() error = %v", err)
	}
	repeated, err := NewMessageClaimID(json.RawMessage(`{"message_id":"64b7f2c1a3e5d7890abc5678"}`))
	if err != nil {
		t.Fatalf("NewMessageClaimID() repeated error = %v", err)
	}
	second, err := NewMessageClaimID(json.RawMessage(`{"message_id":"64b7f2c1a3e5d7890abc5679"}`))
	if err != nil {
		t.Fatalf("NewMessageClaimID() second error = %v", err)
	}
	if first != repeated {
		t.Fatalf("claim IDs differ for the same message: %s != %s", first, repeated)
	}
	if first == second {
		t.Fatalf("claim IDs match for distinct messages: %s", first)
	}
}

func TestNewMessageClaimIDRejectsMalformedMessageID(t *testing.T) {
	t.Parallel()

	if _, err := NewMessageClaimID(json.RawMessage(`{"message_id":"not-an-object-id"}`)); err == nil {
		t.Fatal("NewMessageClaimID() error = nil, want malformed ID error")
	}
}
