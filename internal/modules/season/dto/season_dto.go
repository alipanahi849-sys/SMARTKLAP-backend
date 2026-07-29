package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type CreateSeasonRequest struct {
	LeagueID  uuid.UUID `json:"league_id" binding:"required"`
	Name      string    `json:"name" binding:"required"`
	StartDate string    `json:"start_date" binding:"required"`
	EndDate   string    `json:"end_date" binding:"required"`
	IsActive  bool      `json:"is_active"`
}

type UpdateSeasonRequest struct {
	Name      string `json:"name" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
	IsActive  *bool  `json:"is_active"`
}

type SeasonResponse struct {
	ID        uuid.UUID `json:"id"`
	LeagueID  uuid.UUID `json:"league_id"`
	Name      string    `json:"name"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	IsActive  bool      `json:"is_active"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type SeasonListResponse struct {
	Data       []SeasonResponse         `json:"data"`
	Pagination utils.PaginationResponse `json:"pagination"`
}
