package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

// NewsItem matches the club_news preview shape of the mobile contract (§3.2).
type NewsItem struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Date     string    `json:"date"`
	ImageURL string    `json:"image_url"`
}

type NewsListResponse struct {
	Items []NewsItem     `json:"items"`
	Meta  utils.ListMeta `json:"meta"`
}
