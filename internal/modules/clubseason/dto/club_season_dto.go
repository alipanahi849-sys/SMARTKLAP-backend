package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type CreateClubSeasonRequest struct {
	ClubID   uuid.UUID `json:"club_id" binding:"required"`
	SeasonID uuid.UUID `json:"season_id" binding:"required"`
	JoinedAt string    `json:"joined_at" binding:"required"`
	Status   string    `json:"status" binding:"required,oneof=active suspended withdrawn"`
}

type UpdateClubSeasonRequest struct {
	Status string `json:"status" binding:"required,oneof=active suspended withdrawn"`
}

type ClubSeasonResponse struct {
	ID        uuid.UUID `json:"id"`
	ClubID    uuid.UUID `json:"club_id"`
	SeasonID  uuid.UUID `json:"season_id"`
	JoinedAt  string    `json:"joined_at"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type ClubSeasonListResponse struct {
	Data       []ClubSeasonResponse     `json:"data"`
	Pagination utils.PaginationResponse `json:"pagination"`
}
