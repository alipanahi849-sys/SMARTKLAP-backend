package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

// VideoAuthor is the poster's public identity (contract §8.1).
type VideoAuthor struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// VideoItem is one feed entry.
type VideoItem struct {
	ID           uuid.UUID   `json:"id"`
	VideoURL     string      `json:"video_url"`
	ThumbnailURL string      `json:"thumbnail_url"`
	Author       VideoAuthor `json:"author"`
	PostedAt     string      `json:"posted_at"`
	Tags         []string    `json:"tags"`
	LikesCount   int         `json:"likes_count"`
	ViewsCount   int         `json:"views_count"`
	IsLiked      bool        `json:"is_liked"`
}

// VideoFeedResponse is GET /videos/feed and GET /videos/mine.
type VideoFeedResponse struct {
	Items []VideoItem    `json:"items"`
	Meta  utils.ListMeta `json:"meta"`
}

// VideoUploadResponse is POST /videos (contract §8.3).
type VideoUploadResponse struct {
	ID       uuid.UUID `json:"id"`
	Status   string    `json:"status"`
	VideoURL *string   `json:"video_url"`
}
