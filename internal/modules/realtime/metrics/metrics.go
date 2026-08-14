// Package metrics holds in-process atomic counters for the realtime WebSocket layer.
// No external dependency required: counters are read by a /metrics endpoint and
// included in structured log lines.
package metrics

import "sync/atomic"

// Metrics tracks the key observability signals for the WebSocket delivery layer.
// All fields are safe for concurrent access via sync/atomic.
type Metrics struct {
	// Connection-level gauges
	ActiveConnections   atomic.Int64
	ActiveSubscriptions atomic.Int64

	// Event throughput counters (monotonically increasing)
	EventsPublished atomic.Int64
	EventsDelivered atomic.Int64
	EventsFailed    atomic.Int64

	// Production reliability/observability counters.
	// EventsPublishFailed counts gateway publish failures (serialisation or
	// enqueue), distinct from EventsFailed which also covers slow-consumer drops.
	EventsPublishFailed atomic.Int64
	// ClientsDroppedBufferFull counts connections evicted because their outbound
	// buffer was full (slow consumers).
	ClientsDroppedBufferFull atomic.Int64
	// AuthFailures counts rejected WebSocket upgrade authentications.
	AuthFailures atomic.Int64
	// SubscriptionsRejected counts subscribe requests blocked by the rate limiter.
	SubscriptionsRejected atomic.Int64
	// UsersDisconnected counts forced server-side user disconnects.
	UsersDisconnected atomic.Int64
	// SessionsExpired counts connections closed because their JWT expired.
	SessionsExpired atomic.Int64
	// StaleProcessingRecovered counts scheduler events reset from processing→pending
	// by the watchdog.
	StaleProcessingRecovered atomic.Int64
	// HubPanics counts recovered panics inside the Hub event loop.
	HubPanics atomic.Int64
}

// New returns a zero-value Metrics instance.
func New() *Metrics {
	return &Metrics{}
}

// Snapshot returns a consistent read of all counters at this instant.
func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"active_connections":          m.ActiveConnections.Load(),
		"active_subscriptions":        m.ActiveSubscriptions.Load(),
		"events_published":            m.EventsPublished.Load(),
		"events_delivered":            m.EventsDelivered.Load(),
		"events_failed":               m.EventsFailed.Load(),
		"events_publish_failed":       m.EventsPublishFailed.Load(),
		"clients_dropped_buffer_full": m.ClientsDroppedBufferFull.Load(),
		"auth_failures":               m.AuthFailures.Load(),
		"subscriptions_rejected":      m.SubscriptionsRejected.Load(),
		"users_disconnected":          m.UsersDisconnected.Load(),
		"sessions_expired":            m.SessionsExpired.Load(),
		"stale_processing_recovered":  m.StaleProcessingRecovered.Load(),
		"hub_panics":                  m.HubPanics.Load(),
	}
}
