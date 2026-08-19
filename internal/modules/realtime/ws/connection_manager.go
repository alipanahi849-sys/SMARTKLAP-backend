package ws

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrHubStopped is returned when a query reaches a Hub that has shut down.
var ErrHubStopped = errors.New("websocket hub stopped")

// ConnectionManager is the high-level API that the WebSocket gateway and handlers
// use to interact with the Hub.  It hides Hub internals from the rest of the
// application so the Hub can be replaced or mocked independently.
type ConnectionManager struct {
	hub *Hub
}

// NewConnectionManager wraps a Hub.
func NewConnectionManager(hub *Hub) *ConnectionManager {
	return &ConnectionManager{hub: hub}
}

// PublishToChannel sends data to every client subscribed to the named channel.
// Non-blocking: the message is placed on each shard's publish queue; callers
// receive a context-cancellation error if a queue is full when ctx expires.
func (cm *ConnectionManager) PublishToChannel(ctx context.Context, channel string, data []byte) error {
	return cm.hub.enqueuePublish(ctx, publishMsg{channel: &channel, data: data})
}

// PublishToUser sends data to all active connections of a single user.
func (cm *ConnectionManager) PublishToUser(ctx context.Context, userID uuid.UUID, data []byte) error {
	return cm.hub.enqueuePublish(ctx, publishMsg{userID: &userID, data: data})
}

// Broadcast sends data to every connected client regardless of channel.
func (cm *ConnectionManager) Broadcast(ctx context.Context, data []byte) error {
	return cm.hub.enqueuePublish(ctx, publishMsg{data: data})
}

// SetWelcomeMessage stores a snapshot delivered to newly registered connections
// (late join during a chant countdown). Pass nil to clear.
func (cm *ConnectionManager) SetWelcomeMessage(data []byte) {
	cm.hub.SetWelcomeMessage(data)
}

// DisconnectUser forcibly terminates every WebSocket connection owned by the
// given user and removes them from all channels (CR-9).
func (cm *ConnectionManager) DisconnectUser(ctx context.Context, userID uuid.UUID) error {
	return cm.hub.enqueueDisconnectUser(ctx, userID)
}

// ConnectedUserIDs returns the distinct user IDs of every connected client.
// The snapshot is taken inside the shard event loops, so it is consistent but
// may be stale by the time it is used.
func (cm *ConnectionManager) ConnectedUserIDs(ctx context.Context) ([]uuid.UUID, error) {
	return cm.hub.queryUsers(ctx, usersQuery{all: true})
}

// ChannelUserIDs returns the distinct user IDs currently subscribed to the
// named channel.
func (cm *ConnectionManager) ChannelUserIDs(ctx context.Context, channel string) ([]uuid.UUID, error) {
	return cm.hub.queryUsers(ctx, usersQuery{channel: channel})
}

// Healthy reports whether the underlying Hub event loops are operating normally.
func (cm *ConnectionManager) Healthy() bool {
	return cm.hub.Healthy()
}

// Hub returns the underlying Hub so the WS handler can register new clients.
func (cm *ConnectionManager) Hub() *Hub {
	return cm.hub
}

// ActiveConnectionCount is a best-effort read of the current connection gauge.
func (cm *ConnectionManager) ActiveConnectionCount() int64 {
	return cm.hub.metrics.ActiveConnections.Load()
}

// ActiveSubscriptionCount is a best-effort read of the current subscription gauge.
func (cm *ConnectionManager) ActiveSubscriptionCount() int64 {
	return cm.hub.metrics.ActiveSubscriptions.Load()
}
