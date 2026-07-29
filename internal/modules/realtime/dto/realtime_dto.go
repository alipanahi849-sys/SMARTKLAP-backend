package dto

import (
	"time"

	"clap/internal/modules/realtime/models"

	"github.com/google/uuid"
)

// ─── Time-Sync DTOs ──────────────────────────────────────────────────────────

type TimeSyncRequest struct {
	ClientTimestampMs int64 `json:"client_timestamp_ms" binding:"required"`
}

type ServerTimeResponse struct {
	ServerTimeMs  int64  `json:"server_time_ms"`
	ServerTimeISO string `json:"server_time_iso"`
}

// DriftResponse exposes the raw drift calculation for diagnostics.
type DriftResponse struct {
	ClientTimestampMs int64 `json:"client_timestamp_ms"`
	ServerTimestampMs int64 `json:"server_timestamp_ms"`
	DriftMs           int64 `json:"drift_ms"` // positive = client behind server
	AbsDriftMs        int64 `json:"abs_drift_ms"`
	IsSignificant     bool  `json:"is_significant"` // true when |drift| > 500 ms
}

// TimeSyncResponse is the full payload returned to clients on sync requests.
type TimeSyncResponse struct {
	ServerTimeMs      int64  `json:"server_time_ms"`
	ServerTimeISO     string `json:"server_time_iso"`
	ClientTimestampMs int64  `json:"client_timestamp_ms"`
	DriftMs           int64  `json:"drift_ms"`
	CorrectedClientMs int64  `json:"corrected_client_ms"`
	IsSignificant     bool   `json:"is_significant"`
}

// ─── Session DTOs ─────────────────────────────────────────────────────────────

type RealtimeSessionResponse struct {
	ID        uuid.UUID            `json:"id"`
	MatchID   uuid.UUID            `json:"match_id"`
	StartedAt *time.Time           `json:"started_at,omitempty"`
	Status    models.SessionStatus `json:"status"`
	CreatedAt time.Time            `json:"created_at"`
}

// ─── Event DTOs ───────────────────────────────────────────────────────────────

type RealtimeEventResponse struct {
	ID          uuid.UUID                `json:"id"`
	SessionID   uuid.UUID                `json:"session_id"`
	EventType   models.RealtimeEventType `json:"event_type"`
	ExecuteAtMs int64                    `json:"execute_at_ms"`
	PayloadJSON string                   `json:"payload_json"`
	CreatedAt   time.Time                `json:"created_at"`
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func ToRealtimeSessionResponse(s *models.RealtimeSession) *RealtimeSessionResponse {
	return &RealtimeSessionResponse{
		ID:        s.ID,
		MatchID:   s.MatchID,
		StartedAt: s.StartedAt,
		Status:    s.Status,
		CreatedAt: s.CreatedAt,
	}
}

func ToRealtimeEventResponse(e *models.RealtimeEvent) *RealtimeEventResponse {
	return &RealtimeEventResponse{
		ID:          e.ID,
		SessionID:   e.SessionID,
		EventType:   e.EventType,
		ExecuteAtMs: e.ExecuteAtMs,
		PayloadJSON: e.PayloadJSON,
		CreatedAt:   e.CreatedAt,
	}
}
