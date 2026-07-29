package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type CreateSongRequest struct {
	Title    string `json:"title" binding:"required"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration int    `json:"duration"`
	AudioURL string `json:"audio_url"`
	IsActive bool   `json:"is_active"`
}

type UpdateSongRequest struct {
	Title    string `json:"title" binding:"required"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration int    `json:"duration"`
	AudioURL string `json:"audio_url"`
	IsActive *bool  `json:"is_active"`
}

type SongResponse struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Artist      string     `json:"artist"`
	Album       string     `json:"album"`
	Duration    int        `json:"duration"`
	AudioURL    string     `json:"audio_url"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
	MediaFileID *uuid.UUID `json:"media_file_id,omitempty"`
	StorageKey  string     `json:"storage_key,omitempty"`
	MimeType    string     `json:"mime_type,omitempty"`
	FileSize    int64      `json:"file_size,omitempty"`
	DurationMs  int64      `json:"duration_ms,omitempty"`
	Bitrate     int        `json:"bitrate,omitempty"`
	SampleRate  int        `json:"sample_rate,omitempty"`
}

type SongListResponse struct {
	Data       []SongResponse           `json:"data"`
	Pagination utils.PaginationResponse `json:"pagination"`
}
