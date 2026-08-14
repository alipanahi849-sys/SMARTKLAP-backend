package dto

import "github.com/google/uuid"

type NewsItem struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
	ImageURL  string    `json:"image_url"`
}

type NewsListFilters struct {
	Cursor *uuid.UUID
	Limit  int
}

type NewsListMeta struct {
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
	NextCursor *uuid.UUID `json:"next_cursor,omitempty"`
}

type NewsListResponse struct {
	Items []NewsItem   `json:"items"`
	Meta  NewsListMeta `json:"meta"`
}

type NewsDetailResponse struct {
	ID          uuid.UUID  `json:"id"`
	ClubID      *uuid.UUID `json:"club_id,omitempty"`
	Title       string     `json:"title"`
	BodyHTML    string     `json:"body_html"`
	ImageURL    string     `json:"image_url"`
	PublishedAt string     `json:"published_at"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}
