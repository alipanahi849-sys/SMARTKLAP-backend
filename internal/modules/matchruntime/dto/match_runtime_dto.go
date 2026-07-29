package dto

import (
	"time"

	"clap/internal/modules/matchruntime/models"

	"github.com/google/uuid"
)

type MatchRuntimeResponse struct {
	ID            uuid.UUID            `json:"id"`
	MatchID       uuid.UUID            `json:"match_id"`
	Status        models.RuntimeStatus `json:"status"`
	StartedAt     *time.Time           `json:"started_at,omitempty"`
	PausedAt      *time.Time           `json:"paused_at,omitempty"`
	EndedAt       *time.Time           `json:"ended_at,omitempty"`
	TotalPausedMs int64                `json:"total_paused_ms"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// MatchTimeResponse is returned by CurrentMatchTime — it includes a computed
// elapsed value so clients do not have to repeat the arithmetic.
type MatchTimeResponse struct {
	MatchID       uuid.UUID  `json:"match_id"`
	Status        string     `json:"status"`
	ElapsedMs     int64      `json:"elapsed_ms"`
	TotalPausedMs int64      `json:"total_paused_ms"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	ServerTimeMs  int64      `json:"server_time_ms"`
}

func ToMatchRuntimeResponse(s *models.MatchRuntimeState) *MatchRuntimeResponse {
	return &MatchRuntimeResponse{
		ID:            s.ID,
		MatchID:       s.MatchID,
		Status:        s.Status,
		StartedAt:     s.StartedAt,
		PausedAt:      s.PausedAt,
		EndedAt:       s.EndedAt,
		TotalPausedMs: s.TotalPausedMs,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}
