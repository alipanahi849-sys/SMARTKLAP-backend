package dto

import (
	"mime/multipart"

	"github.com/google/uuid"
)

type MediaUploadRequest struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

type MediaResponse struct {
	ID               uuid.UUID `json:"id"`
	StorageKey       string    `json:"storage_key"`
	OriginalFileName string    `json:"original_file_name"`
	MimeType         string    `json:"mime_type"`
	FileSize         int64     `json:"file_size"`
	Checksum         string    `json:"checksum"`
	UploadedBy       uuid.UUID `json:"uploaded_by"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
}

type PlaybackURLResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

type SongAudioUploadRequest struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

type SongAudioUploadResponse struct {
	MediaFileID uuid.UUID `json:"media_file_id"`
	StorageKey  string    `json:"storage_key"`
	MimeType    string    `json:"mime_type"`
	FileSize    int64     `json:"file_size"`
	DurationMs  int64     `json:"duration_ms"`
	Bitrate     int       `json:"bitrate"`
	SampleRate  int       `json:"sample_rate"`
}

type LyricsImportRequest struct {
	Content         string `json:"content" binding:"required"`
	ReplaceExisting bool   `json:"replace_existing"`
}

type LyricsImportResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}
