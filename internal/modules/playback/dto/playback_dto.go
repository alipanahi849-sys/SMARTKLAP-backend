package dto

import (
	"time"

	"clap/internal/modules/playback/models"

	"github.com/google/uuid"
)

type ScheduleSongRequest struct {
	MatchID     uuid.UUID `json:"match_id"     binding:"required"`
	SongID      uuid.UUID `json:"song_id"      binding:"required"`
	ScheduledAt time.Time `json:"scheduled_at" binding:"required"`
	// DurationMs is the expected playback window in milliseconds.
	// Providing it enables overlap detection against other schedules.
	// Zero means "unknown duration" and overlap check is skipped.
	DurationMs int64 `json:"duration_ms"`
}

type PlaybackScheduleResponse struct {
	ID          uuid.UUID             `json:"id"`
	MatchID     uuid.UUID             `json:"match_id"`
	SongID      uuid.UUID             `json:"song_id"`
	ScheduledAt time.Time             `json:"scheduled_at"`
	DurationMs  int64                 `json:"duration_ms"`
	Status      models.PlaybackStatus `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
}

type UpcomingPlaybackResponse struct {
	Schedules []*PlaybackScheduleResponse `json:"schedules"`
	Total     int                         `json:"total"`
}

func ToPlaybackScheduleResponse(s *models.PlaybackSchedule) *PlaybackScheduleResponse {
	return &PlaybackScheduleResponse{
		ID:          s.ID,
		MatchID:     s.MatchID,
		SongID:      s.SongID,
		ScheduledAt: s.ScheduledAt,
		DurationMs:  s.DurationMs,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt,
	}
}
