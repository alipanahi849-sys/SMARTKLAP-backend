package dto

import (
	"time"

	"github.com/google/uuid"
)

// RankInfo is the user's leaderboard position (Mobile API Contract §2.1).
type RankInfo struct {
	Position int   `json:"position"`
	Total    int64 `json:"total"`
}

// MobileProfileResponse is the shape of GET/PATCH /profile/me.
// ID is the user id (not the profiles-table row id).
type MobileProfileResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url"`
	Points    int       `json:"points"`
	Rank      RankInfo  `json:"rank"`
}

// UpdateMobileProfileRequest is the PATCH /profile/me body (§2.2).
type UpdateMobileProfileRequest struct {
	Name  *string `json:"name" binding:"omitempty,min=1,max=100"`
	Email *string `json:"email" binding:"omitempty,email"`
}

// LeaderboardItem is a single "Top ranks" card (§2.1).
type LeaderboardItem struct {
	Rank      int    `json:"rank"`
	Name      string `json:"name"`
	Points    int    `json:"points"`
	AvatarURL string `json:"avatar_url"`
}

// LeaderboardResponse wraps GET /profile/leaderboard.
type LeaderboardResponse struct {
	Items []LeaderboardItem `json:"items"`
}

// AvatarUploadResponse is returned by POST /profile/me/avatar (§2.2).
type AvatarUploadResponse struct {
	AvatarURL string `json:"avatar_url"`
}
