package event

import "time"

// GigApprovedNotificationEvent is published by the Gig service when a gig
// listing is approved. Field set is inferred by analogy with
// NewMessageNotificationEvent since CLAUDE.md does not enumerate it;
// adjust to match the producer's actual payload.
type GigApprovedNotificationEvent struct {
	RecipientID string    `json:"recipient_id"`
	ApproverID  string    `json:"approver_id"`
	GigID       string    `json:"gig_id"`
	GigTitle    string    `json:"gig_title"`
	OccurredAt  time.Time `json:"occurred_at"`
}
