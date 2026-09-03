package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/vsayfb/gig-platform-notification-lambda/internal/event"
	"github.com/vsayfb/gig-platform-notification-lambda/internal/service"
)

type pubSubPushEnvelope struct {
	Message struct {
		Attributes map[string]string `json:"attributes"`
		Data       []byte            `json:"data"`
		ID         string            `json:"messageId"`
	} `json:"message"`
}

func (h *Handler) HandlePubSubPush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var push pubSubPushEnvelope
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&push); err != nil || len(push.Message.Data) == 0 {
		slog.WarnContext(r.Context(), "discard malformed Pub/Sub push", "message_id", push.Message.ID)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var envelope event.Envelope
	if err := json.Unmarshal(push.Message.Data, &envelope); err != nil {
		slog.WarnContext(r.Context(), "discard malformed notification envelope", "message_id", push.Message.ID)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := h.service.Handle(r.Context(), envelope); err != nil {
		if code, ok := service.NonRetryableCode(err); ok {
			slog.WarnContext(r.Context(), "discard non-retryable notification event", "diagnostic_code", code, "message_id", push.Message.ID)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		slog.ErrorContext(r.Context(), "handle Pub/Sub notification event", "error", err, "message_id", push.Message.ID)
		http.Error(w, "retry", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
