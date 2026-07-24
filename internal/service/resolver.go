package service

import (
	"context"
	"encoding/json"
)

// Resolver expands a single inbound event payload into one payload per
// notification recipient. Most event types don't need one — the payload
// already names its own recipient — so registering a Resolver is opt-in,
// only for event types where recipients must be resolved live against the
// database at consume time (e.g. "everyone subscribed to category X in
// city Y right now"). Renderer stays untouched either way: it always
// receives exactly one already-addressed payload.
type Resolver interface {
	Resolve(ctx context.Context, payload json.RawMessage) ([]json.RawMessage, error)
}
