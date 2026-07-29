// Package gateway defines the transport-agnostic realtime messaging contract.
// Swap InMemoryRealtimeGateway for a Centrifugo, NATS, or Kafka implementation
// without touching any domain or service code.
package gateway

import (
	"context"
	"encoding/json"
)

// GatewayEvent is the envelope sent over the realtime transport.
type GatewayEvent struct {
	Type    string          `json:"type"`
	Channel string          `json:"channel"`
	Payload json.RawMessage `json:"payload"`
}

// RealtimeGateway abstracts publish/subscribe for all realtime transports.
type RealtimeGateway interface {
	// Publish sends an event to all subscribers of a specific channel.
	Publish(ctx context.Context, channel string, event *GatewayEvent) error

	// Broadcast sends an event to every active subscriber regardless of channel.
	Broadcast(ctx context.Context, event *GatewayEvent) error

	// Subscribe returns a receive-only channel of events for the given channel key.
	// The returned channel is closed when ctx is cancelled — callers must handle this.
	Subscribe(ctx context.Context, channel string) (<-chan *GatewayEvent, error)

	// DisconnectUser removes all subscriptions and in-flight state for a user.
	DisconnectUser(ctx context.Context, userID string) error
}
