package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type CreateSongLyricRequest struct {
	SongID   uuid.UUID `json:"song_id" binding:"required"`
	Language string    `json:"language" binding:"required"`
	Lyrics   string    `json:"lyrics" binding:"required"`
}

type UpdateSongLyricRequest struct {
	Language string `json:"language" binding:"required"`
	Lyrics   string `json:"lyrics" binding:"required"`
}

type SongLyricResponse struct {
	ID        uuid.UUID `json:"id"`
	SongID    uuid.UUID `json:"song_id"`
	Language  string    `json:"language"`
	Lyrics    string    `json:"lyrics"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type SongLyricListResponse struct {
	Data       []SongLyricResponse      `json:"data"`
	Pagination utils.PaginationResponse `json:"pagination"`
}

type ImportLyricsRequest struct {
	Content         string `json:"content" binding:"required"`
	ReplaceExisting bool   `json:"replace_existing"`
}

type LyricsImportResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}
