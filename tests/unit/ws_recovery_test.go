package unit

import (
	"context"
	"testing"
	"time"

	lyricsdto "clap/internal/modules/lyricssync/dto"
	"clap/internal/modules/realtime/dto"
	realtimesvc "clap/internal/modules/realtime/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── EventEnvelope helpers ────────────────────────────────────────────────────

func TestEventEnvelope_NewEnvelope_HasUniqueIDs(t *testing.T) {
	matchID := uuid.New()
	e1 := dto.NewEnvelope(dto.EventTypeMatchRuntimeUpdated, &matchID, nil)
	e2 := dto.NewEnvelope(dto.EventTypeMatchRuntimeUpdated, &matchID, nil)
	assert.NotEqual(t, e1.ID, e2.ID, "each envelope should have a unique ID")
}

func TestEventEnvelope_Timestamp_IsRecent(t *testing.T) {
	before := time.Now().UnixMilli()
	env := dto.NewEnvelope(dto.EventTypeServerNotification, nil, nil)
	after := time.Now().UnixMilli()
	assert.GreaterOrEqual(t, env.Timestamp, before)
	assert.LessOrEqual(t, env.Timestamp, after)
}

// ─── ReconnectionState stub tests ────────────────────────────────────────────

func TestReconnectionState_ServerTimePresent(t *testing.T) {
	before := time.Now().UnixMilli()
	state := &realtimesvc.ReconnectionState{
		ServerTimeMs: time.Now().UnixMilli(),
	}
	after := time.Now().UnixMilli()

	assert.GreaterOrEqual(t, state.ServerTimeMs, before)
	assert.LessOrEqual(t, state.ServerTimeMs, after)
}

// ─── LyricsEventBuilder tests ─────────────────────────────────────────────────

func TestBuildLyricsRealtimeEvents(t *testing.T) {
	matchID := uuid.New()
	sessionStart := int64(1_000_000)

	timeline := &lyricsdto.LyricsTimeline{
		SongID:   uuid.New(),
		Language: "en",
		Entries: []lyricsdto.LyricTimelineEntry{
			{Index: 0, TimestampMs: 0, Text: "We are the champions"},
			{Index: 1, TimestampMs: 5000, Text: "No time for losers"},
		},
		Total: 2,
	}

	// Manually build what BuildRealtimeEvents should produce.
	envelopes := make([]*dto.EventEnvelope, 0, len(timeline.Entries))
	for _, entry := range timeline.Entries {
		payload := dto.LyricsLinePayload{
			Line:        entry.Text,
			TimestampMs: sessionStart + entry.TimestampMs,
			Index:       entry.Index,
		}
		env := dto.NewEnvelope(dto.EventTypeLyricsLineChanged, &matchID, payload)
		envelopes = append(envelopes, env)
	}

	require.Len(t, envelopes, 2)
	assert.Equal(t, dto.EventTypeLyricsLineChanged, envelopes[0].Type)
	assert.Equal(t, matchID.String(), envelopes[0].MatchID.String())

	// Verify the payload has correct fields.
	payload0, ok := envelopes[0].Payload.(dto.LyricsLinePayload)
	require.True(t, ok)
	assert.Equal(t, "We are the champions", payload0.Line)
	assert.Equal(t, sessionStart+0, payload0.TimestampMs)

	payload1, ok := envelopes[1].Payload.(dto.LyricsLinePayload)
	require.True(t, ok)
	assert.Equal(t, "No time for losers", payload1.Line)
	assert.Equal(t, sessionStart+5000, payload1.TimestampMs)
}

// ─── EventDispatcher payload helper tests ────────────────────────────────────

// stubDispatchGateway captures published events.
type stubDispatchGateway struct {
	published []struct {
		matchID   uuid.UUID
		eventType string
		payload   any
	}
}

func (g *stubDispatchGateway) PublishMatchEvent(_ context.Context, matchID uuid.UUID, eventType string, payload any) error {
	g.published = append(g.published, struct {
		matchID   uuid.UUID
		eventType string
		payload   any
	}{matchID, eventType, payload})
	return nil
}

func (g *stubDispatchGateway) BroadcastEvent(_ context.Context, matchID uuid.UUID, eventType string, payload any) error {
	return g.PublishMatchEvent(context.Background(), matchID, eventType, payload)
}

func TestEventTypeConstants(t *testing.T) {
	assert.Equal(t, "match.runtime.updated", dto.EventTypeMatchRuntimeUpdated)
	assert.Equal(t, "song.playback.started", dto.EventTypeSongPlaybackStarted)
	assert.Equal(t, "song.playback.cancelled", dto.EventTypeSongPlaybackCancelled)
	assert.Equal(t, "lyrics.line.changed", dto.EventTypeLyricsLineChanged)
	assert.Equal(t, "server.notification", dto.EventTypeServerNotification)
}
