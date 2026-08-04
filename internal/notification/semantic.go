package notification

import (
	"encoding/json"
	"fmt"
)

// AddLocalizationMetadata persists the semantic localization contract in the
// existing metadata JSON object returned by Core's notifications endpoint.
func AddLocalizationMetadata(
	metadata map[string]any,
	localizationKey string,
	localizationArgs map[string]string,
) map[string]any {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	if localizationArgs == nil {
		localizationArgs = map[string]string{}
	}

	metadata["localization_key"] = localizationKey
	metadata["localization_args"] = localizationArgs
	return metadata
}

// NewSemanticPush creates an FCM data payload whose authoritative content is
// locale-neutral. Title and body remain only as a temporary display fallback
// for clients that do not yet understand the semantic fields.
func NewSemanticPush(
	recipientID string,
	n *Notification,
	localizationKey string,
	localizationArgs map[string]string,
	legacyTitle string,
	legacyBody string,
	additionalRouting map[string]string,
) (*PushNotification, error) {
	if localizationKey == "" {
		return nil, fmt.Errorf("notification: localization_key is required")
	}
	if localizationArgs == nil {
		localizationArgs = map[string]string{}
	}

	encodedArgs, err := json.Marshal(localizationArgs)
	if err != nil {
		return nil, fmt.Errorf("notification: marshal localization_args: %w", err)
	}

	data := make(map[string]string, 5+len(additionalRouting))
	data["type"] = n.Type
	data["entity_type"] = n.EntityType
	data["entity_id"] = n.EntityID
	data["localization_key"] = localizationKey
	data["localization_args"] = string(encodedArgs)

	for key, value := range additionalRouting {
		if _, reserved := data[key]; reserved {
			return nil, fmt.Errorf("notification: additional routing overrides semantic field %q", key)
		}
		data[key] = value
	}

	return &PushNotification{
		RecipientID: recipientID,
		Title:       legacyTitle,
		Body:        legacyBody,
		Data:        data,
	}, nil
}
