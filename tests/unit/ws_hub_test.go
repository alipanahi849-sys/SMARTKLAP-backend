package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"clap/internal/modules/realtime/dto"
	"clap/internal/modules/realtime/metrics"
	"clap/internal/modules/realtime/ws"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

func newTestHub(t *testing.T) (*ws.Hub, *ws.ConnectionManager, context.CancelFunc) {
	t.Helper()
	m := metrics.New()
	hub := ws.NewHub(m)
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	// Give the Run goroutine a moment to start.
	time.Sleep(10 * time.Millisecond)
	cm := ws.NewConnectionManager(hub)
	return hub, cm, cancel
}

// ─── Connection lifecycle ─────────────────────────────────────────────────────

func TestHub_ConnectionMetrics(t *testing.T) {
	_, cm, cancel := newTestHub(t)
	defer cancel()

	// Initially no connections.
	assert.Equal(t, int64(0), cm.ActiveConnectionCount())
	assert.Equal(t, int64(0), cm.ActiveSubscriptionCount())
}

// ─── Channel helpers ──────────────────────────────────────────────────────────

func TestMatchChannel_Format(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	want := "match:550e8400-e29b-41d4-a716-446655440000"
	assert.Equal(t, want, ws.MatchChannel(id))
}

// ─── Publish to channel ───────────────────────────────────────────────────────

func TestConnectionManager_PublishToChannel_Succeeds(t *testing.T) {
	_, cm, cancel := newTestHub(t)
	defer cancel()

	ctx := context.Background()
	channel := ws.MatchChannel(uuid.New())
	// Publish to a channel with no subscribers — should not error.
	err := cm.PublishToChannel(ctx, channel, []byte(`{"type":"test"}`))
	assert.NoError(t, err)
}

func TestConnectionManager_Broadcast_Succeeds(t *testing.T) {
	_, cm, cancel := newTestHub(t)
	defer cancel()

	ctx := context.Background()
	err := cm.Broadcast(ctx, []byte(`{"type":"test"}`))
	assert.NoError(t, err)
}

// ─── Hub graceful shutdown ────────────────────────────────────────────────────

func TestHub_GracefulShutdown(t *testing.T) {
	m := metrics.New()
	hub := ws.NewHub(m)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel() // signal shutdown

	select {
	case <-done:
		// Hub exited cleanly.
	case <-time.After(2 * time.Second):
		t.Fatal("Hub.Run did not exit within 2 seconds of context cancellation")
	}
}

// ─── EventEnvelope encoding ───────────────────────────────────────────────────

func TestEventEnvelope_JSON(t *testing.T) {
	matchID := uuid.New()
	env := dto.NewEnvelope(dto.EventTypeMatchRuntimeUpdated, &matchID, dto.MatchRuntimePayload{
		Status:    "running",
		ElapsedMs: 42000,
	})

	data, err := json.Marshal(env)
	require.NoError(t, err)

	var decoded dto.EventEnvelope
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, dto.EventTypeMatchRuntimeUpdated, decoded.Type)
	assert.Equal(t, matchID.String(), decoded.MatchID.String())
	assert.NotEmpty(t, decoded.ID)
	assert.Greater(t, decoded.Timestamp, int64(0))
}

func TestEventEnvelope_OmitsMatchIDWhenNil(t *testing.T) {
	env := dto.NewEnvelope(dto.EventTypeServerNotification, nil, map[string]any{"msg": "hello"})
	data, err := json.Marshal(env)
	require.NoError(t, err)

	// match_id should be absent from the JSON.
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	_, hasMatchID := raw["match_id"]
	assert.False(t, hasMatchID, "match_id should be omitted when nil")
}

// ─── Metrics snapshot ────────────────────────────────────────────────────────

func TestMetrics_Snapshot(t *testing.T) {
	m := metrics.New()
	m.ActiveConnections.Add(3)
	m.ActiveSubscriptions.Add(7)
	m.EventsPublished.Add(100)
	m.EventsDelivered.Add(99)
	m.EventsFailed.Add(1)

	snap := m.Snapshot()
	assert.Equal(t, int64(3), snap["active_connections"])
	assert.Equal(t, int64(7), snap["active_subscriptions"])
	assert.Equal(t, int64(100), snap["events_published"])
	assert.Equal(t, int64(99), snap["events_delivered"])
	assert.Equal(t, int64(1), snap["events_failed"])
}
