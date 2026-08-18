package dto

import "github.com/google/uuid"

type NewsItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	ImageURL  string `json:"image_url"`
}

type NewsListFilters struct {
	Cursor string
	Limit  int
}

type NewsListMeta struct {
	Limit      int     `json:"limit"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor,omitempty"`
}

type NewsListResponse struct {
	Items []NewsItem   `json:"items"`
	Meta  NewsListMeta `json:"meta"`
}

type NewsDetailResponse struct {
	ID          string     `json:"id"`
	ClubID      *uuid.UUID `json:"club_id,omitempty"`
	Title       string     `json:"title"`
	BodyHTML    string     `json:"body_html"`
	ImageURL    string     `json:"image_url"`
	PublishedAt string     `json:"published_at"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}

type NewsClubResponse struct {
	ClubID         *uuid.UUID `json:"club_id"`
	Name           string     `json:"name,omitempty"`
	LogoURL        string     `json:"logo_url,omitempty"`
	ProviderTeamID string     `json:"provider_team_id,omitempty"`
	Provider       string     `json:"provider,omitempty"`
}

type SetNewsClubRequest struct {
	ClubID         *uuid.UUID `json:"club_id"`
	ProviderTeamID string     `json:"provider_team_id"`
}
