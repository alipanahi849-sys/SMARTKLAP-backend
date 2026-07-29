package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type CreateLeagueRequest struct {
	Name             string `json:"name" binding:"required"`
	Country          string `json:"country"`
	Provider         string `json:"provider"`
	ProviderLeagueID string `json:"provider_league_id"`
	IsActive         bool   `json:"is_active"`
}

type UpdateLeagueRequest struct {
	Name             string `json:"name" binding:"required"`
	Country          string `json:"country"`
	Provider         string `json:"provider"`
	ProviderLeagueID string `json:"provider_league_id"`
	IsActive         *bool  `json:"is_active"`
}

type LeagueResponse struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Country          string    `json:"country"`
	Provider         string    `json:"provider"`
	ProviderLeagueID string    `json:"provider_league_id"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
}

type LeagueListResponse struct {
	Data       []LeagueResponse         `json:"data"`
	Pagination utils.PaginationResponse `json:"pagination"`
}
