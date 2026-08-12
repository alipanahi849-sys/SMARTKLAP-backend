package dto

import (
	"time"

	"github.com/google/uuid"
)

// ─── Event type constants ─────────────────────────────────────────────────────

const (
	EventTypeMatchRuntimeUpdated   = "match.runtime.updated"
	EventTypeSongPlaybackStarted   = "song.playback.started"
	EventTypeSongPlaybackCancelled = "song.playback.cancelled"
	EventTypeLyricsLineChanged     = "lyrics.line.changed"
	EventTypeServerNotification    = "server.notification"
	// EventTypeChantUpcoming is emitted (per user) when an active chant is
	// about to start (~2 minutes before its scheduled time).
	EventTypeChantUpcoming = "chant.upcoming"
	// EventTypeChantStarted is broadcast to every connected client at the
	// chant's scheduled start time — the authoritative "go" signal.
	EventTypeChantStarted = "chant.started"

	// Control events (client ↔ server)
	EventTypePing = "ping"
	EventTypePong = "pong"

	// EventTypeError is a structured server→client error notification, used for
	// rejected subscriptions and other client-actionable failures.
	EventTypeError = "error"
)

// EventEnvelope is the standard wrapper for every realtime message sent to clients.
// All business events MUST use this structure — raw payloads are forbidden.
//
//	{
//	  "id":        "uuid",
//	  "type":      "match.runtime.updated",
//	  "match_id":  "uuid",
//	  "timestamp": 1718100000000,
//	  "server_time_ms": 1718100000000,
//	  "payload":   { ... }
//	}
type EventEnvelope struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	MatchID      *uuid.UUID `json:"match_id,omitempty"`
	Timestamp    int64      `json:"timestamp"`
	ServerTimeMs int64      `json:"server_time_ms"`
	Payload      any        `json:"payload"`
}

// NewEnvelope constructs a ready-to-send EventEnvelope with a fresh UUID and
// a millisecond UTC timestamp.
func NewEnvelope(eventType string, matchID *uuid.UUID, payload any) *EventEnvelope {
	now := time.Now().UnixMilli()
	return &EventEnvelope{
		ID:           uuid.New().String(),
		Type:         eventType,
		MatchID:      matchID,
		Timestamp:    now,
		ServerTimeMs: now,
		Payload:      payload,
	}
}

// ─── Standard payloads ────────────────────────────────────────────────────────

// MatchRuntimePayload is the body of a match.runtime.updated event.
type MatchRuntimePayload struct {
	Status    string `json:"status"`
	ElapsedMs int64  `json:"elapsed_ms"`
}

// PlaybackStartedPayload is the body of a song.playback.started event.
type PlaybackStartedPayload struct {
	ScheduleID string `json:"schedule_id"`
	SongID     string `json:"song_id"`
	MatchID    string `json:"match_id"`
}

// LyricsLinePayload is the body of a lyrics.line.changed event.
type LyricsLinePayload struct {
	Line        string `json:"line"`
	TimestampMs int64  `json:"timestamp_ms"`
	Index       int    `json:"index,omitempty"`
	ChantID     string `json:"chant_id,omitempty"`
}

// ChantStartedPayload is the body of a chant.started event.
type ChantStartedPayload struct {
	ChantID  string    `json:"chant_id"`
	MatchID  string    `json:"match_id"`
	SongID   string    `json:"song_id,omitempty"`
	StartsAt time.Time `json:"starts_at"`
}

// ClientMessage is sent from a connected client to the server.
type ClientMessage struct {
	Type    string `json:"type"`
	Channel string `json:"channel,omitempty"` // for subscribe/unsubscribe
}

// ErrorPayload is the body of a server→client error event.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Channel string `json:"channel,omitempty"`
}
