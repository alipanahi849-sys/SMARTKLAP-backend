package unit_test

import (
	"encoding/json"
	"testing"

	"clap/internal/modules/lyricssync/dto"
	realtimemodels "clap/internal/modules/realtime/models"

	"github.com/google/uuid"
)

// buildTestTimeline constructs a timeline for unit testing without a DB.
func buildTestTimeline(entries []dto.LyricTimelineEntry) *dto.LyricsTimeline {
	return &dto.LyricsTimeline{
		SongID:   uuid.New(),
		Language: "en",
		Entries:  entries,
		Total:    len(entries),
	}
}

// generateEventsFromTimeline is the pure core of LyricsSyncService.GenerateRealtimeEvents,
// extracted here so we can test it without a real database connection.
func generateEventsFromTimeline(
	timeline *dto.LyricsTimeline,
	sessionID uuid.UUID,
	sessionStartMs int64,
) ([]*realtimemodels.RealtimeEvent, error) {
	events := make([]*realtimemodels.RealtimeEvent, 0, len(timeline.Entries))
	for _, entry := range timeline.Entries {
		payload := map[string]interface{}{
			"index":        entry.Index,
			"text":         entry.Text,
			"timestamp_ms": entry.TimestampMs,
		}
		pb, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		events = append(events, &realtimemodels.RealtimeEvent{
			SessionID:   sessionID,
			EventType:   realtimemodels.EventTypeLyricSync,
			ExecuteAtMs: sessionStartMs + entry.TimestampMs,
			PayloadJSON: string(pb),
		})
	}
	return events, nil
}

func TestLyricsSync_EventCount(t *testing.T) {
	entries := []dto.LyricTimelineEntry{
		{Index: 0, TimestampMs: 0, Text: "First line"},
		{Index: 1, TimestampMs: 5000, Text: "Second line"},
		{Index: 2, TimestampMs: 10000, Text: "Third line"},
	}

	events, err := generateEventsFromTimeline(buildTestTimeline(entries), uuid.New(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestLyricsSync_ExecuteAtMsAnchored(t *testing.T) {
	const sessionStartMs = int64(1_700_000_000_000)
	entries := []dto.LyricTimelineEntry{
		{Index: 0, TimestampMs: 1000, Text: "A"},
		{Index: 1, TimestampMs: 3500, Text: "B"},
	}

	events, _ := generateEventsFromTimeline(buildTestTimeline(entries), uuid.New(), sessionStartMs)

	if events[0].ExecuteAtMs != sessionStartMs+1000 {
		t.Errorf("events[0].ExecuteAtMs: got %d, want %d", events[0].ExecuteAtMs, sessionStartMs+1000)
	}
	if events[1].ExecuteAtMs != sessionStartMs+3500 {
		t.Errorf("events[1].ExecuteAtMs: got %d, want %d", events[1].ExecuteAtMs, sessionStartMs+3500)
	}
}

func TestLyricsSync_EventTypeLyricSync(t *testing.T) {
	entries := []dto.LyricTimelineEntry{{Index: 0, TimestampMs: 0, Text: "Hello"}}
	events, _ := generateEventsFromTimeline(buildTestTimeline(entries), uuid.New(), 0)

	if events[0].EventType != realtimemodels.EventTypeLyricSync {
		t.Errorf("expected event_type=%s, got %s", realtimemodels.EventTypeLyricSync, events[0].EventType)
	}
}

func TestLyricsSync_EmptyTimeline(t *testing.T) {
	events, err := generateEventsFromTimeline(buildTestTimeline(nil), uuid.New(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestLyricsSync_SessionIDPropagated(t *testing.T) {
	sessionID := uuid.New()
	entries := []dto.LyricTimelineEntry{{Index: 0, TimestampMs: 500, Text: "Test"}}
	events, _ := generateEventsFromTimeline(buildTestTimeline(entries), sessionID, 0)

	if events[0].SessionID != sessionID {
		t.Errorf("session_id mismatch: got %s, want %s", events[0].SessionID, sessionID)
	}
}

func TestLyricsSync_PayloadContainsText(t *testing.T) {
	entries := []dto.LyricTimelineEntry{{Index: 0, TimestampMs: 0, Text: "Go is awesome"}}
	events, _ := generateEventsFromTimeline(buildTestTimeline(entries), uuid.New(), 0)

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(events[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if payload["text"] != "Go is awesome" {
		t.Errorf("payload text: got %v, want 'Go is awesome'", payload["text"])
	}
}
