package dto

import (
	"time"

	"github.com/google/uuid"
)

// ProfileRank is the user's leaderboard position (contract §2.1).
type ProfileRank struct {
	Position int `json:"position"`
	Total    int `json:"total"`
}

// MobileProfileResponse is the shape of GET/PATCH /profile/me.
// ID is the user id (not the profiles-table row id).
type MobileProfileResponse struct {
	ID        uuid.UUID   `json:"id"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	Name      string      `json:"name"`
	Email     string      `json:"email"`
	AvatarURL string      `json:"avatar_url"`
	Points    int         `json:"points"`
	Rank      ProfileRank `json:"rank"`
}

// UpdateMobileProfileRequest is the PATCH /profile/me body (§2.2).
// Email cannot be changed here — use POST /api/v1/auth/change-email.
type UpdateMobileProfileRequest struct {
	Name *string `json:"name" binding:"omitempty,min=1,max=100"`
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
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	AvatarURL string    `json:"avatar_url"`
}
