package event

import "time"

// NewMessageNotificationEvent is published by the Chat service when a new
// message is sent. It carries facts only; wording is owned by the renderer.
type NewMessageNotificationEvent struct {
	RecipientID    string    `json:"recipient_id"`
	SenderID       string    `json:"sender_id"`
	SenderName     string    `json:"sender_name"`
	ThreadID       string    `json:"thread_id"`
	GigID          string    `json:"gig_id"`
	MessageID      string    `json:"message_id"`
	MessagePreview string    `json:"message_preview"`
	OccurredAt     time.Time `json:"occurred_at"`
}
