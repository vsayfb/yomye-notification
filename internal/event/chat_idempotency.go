package event

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// chatEventNamespace provides a stable UUIDv5 representation for Chat's
// MongoDB message_id so it can use the UUID-keyed processed-event claim table.
// The raw message_id remains the authoritative notification source key.
var chatEventNamespace = uuid.MustParse("71fa1ee5-96e8-5bf8-96bc-d5c4f71ed98a")

func NewMessageClaimID(payload json.RawMessage) (uuid.UUID, error) {
	var identity struct {
		MessageID string `json:"message_id"`
	}
	if err := json.Unmarshal(payload, &identity); err != nil {
		return uuid.Nil, fmt.Errorf("decode message identity: %w", err)
	}
	identity.MessageID = strings.TrimSpace(identity.MessageID)
	if !IsMongoObjectID(identity.MessageID) {
		return uuid.Nil, fmt.Errorf("message_id must be a MongoDB ObjectID")
	}
	return uuid.NewSHA1(chatEventNamespace, []byte("new_message:"+identity.MessageID)), nil
}

func IsMongoObjectID(value string) bool {
	if len(value) != 24 {
		return false
	}
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}
