package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type CreateMatchSongScheduleRequest struct {
	MatchID       uuid.UUID `json:"match_id" binding:"required"`
	SongID        uuid.UUID `json:"song_id" binding:"required"`
	ScheduledTime string    `json:"scheduled_time" binding:"required"`
	EventType     string    `json:"event_type" binding:"required"`
	IsActive      bool      `json:"is_active"`
}

type UpdateMatchSongScheduleRequest struct {
	ScheduledTime string `json:"scheduled_time" binding:"required"`
	EventType     string `json:"event_type" binding:"required"`
	IsActive      *bool  `json:"is_active"`
}

type MatchSongScheduleResponse struct {
	ID            uuid.UUID `json:"id"`
	MatchID       uuid.UUID `json:"match_id"`
	SongID        uuid.UUID `json:"song_id"`
	ScheduledTime string    `json:"scheduled_time"`
	EventType     string    `json:"event_type"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
}

type MatchSongScheduleListResponse struct {
	Data       []MatchSongScheduleResponse `json:"data"`
	Pagination utils.PaginationResponse    `json:"pagination"`
}
