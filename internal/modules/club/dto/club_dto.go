package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type CreateClubRequest struct {
	Name        string `json:"name" binding:"required"`
	ShortName   string `json:"short_name"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	Country     string `json:"country"`
	IsActive    bool   `json:"is_active"`
}

type UpdateClubRequest struct {
	Name        string `json:"name" binding:"required"`
	ShortName   string `json:"short_name"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	Country     string `json:"country"`
	IsActive    *bool  `json:"is_active"`
}

type ClubResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ShortName   string    `json:"short_name"`
	Description string    `json:"description"`
	LogoURL     string    `json:"logo_url"`
	Country     string    `json:"country"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

type ClubListResponse struct {
	Data       []ClubResponse           `json:"data"`
	Pagination utils.PaginationResponse `json:"pagination"`
}
