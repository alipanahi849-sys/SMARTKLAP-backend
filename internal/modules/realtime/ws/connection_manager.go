package ws

import (
	"context"

	"github.com/google/uuid"
)

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
// Non-blocking: the message is placed on the Hub's publish queue; callers
// receive a context-cancellation error if the queue is full when ctx expires.
func (cm *ConnectionManager) PublishToChannel(ctx context.Context, channel string, data []byte) error {
	select {
	case cm.hub.publish <- publishMsg{channel: &channel, data: data}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PublishToUser sends data to all active connections of a single user.
func (cm *ConnectionManager) PublishToUser(ctx context.Context, userID uuid.UUID, data []byte) error {
	select {
	case cm.hub.publish <- publishMsg{userID: &userID, data: data}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Broadcast sends data to every connected client regardless of channel.
func (cm *ConnectionManager) Broadcast(ctx context.Context, data []byte) error {
	select {
	case cm.hub.publish <- publishMsg{data: data}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// DisconnectUser forcibly terminates every WebSocket connection owned by the
// given user and removes them from all channels (CR-9).
func (cm *ConnectionManager) DisconnectUser(ctx context.Context, userID uuid.UUID) error {
	select {
	case cm.hub.disconnectUser <- disconnectMsg{userID: userID}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Healthy reports whether the underlying Hub event loop is operating normally.
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
