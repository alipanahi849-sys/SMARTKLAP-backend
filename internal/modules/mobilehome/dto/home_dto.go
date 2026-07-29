package dto

import (
	newsdto "clap/internal/modules/news/dto"
	shopdto "clap/internal/modules/shop/dto"

	"github.com/google/uuid"
)

// ─── Stadium Mode (contract §3.1) ─────────────────────────────────────────────

type UserSummary struct {
	Points    int `json:"points"`
	CartCount int `json:"cart_count"`
}

type LiveMatchTeam struct {
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
	Score   int    `json:"score"`
}

type LiveMatch struct {
	ID       uuid.UUID     `json:"id"`
	Status   string        `json:"status"`
	Minute   string        `json:"minute"`
	HomeTeam LiveMatchTeam `json:"home_team"`
	AwayTeam LiveMatchTeam `json:"away_team"`
}

type ChantProgramItem struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Subtitle   string    `json:"subtitle"`
	MinutesAgo int       `json:"minutes_ago"`
	Status     string    `json:"status"`
}

type ChantProgram struct {
	TodayPoints int                `json:"today_points"`
	TodayTarget int                `json:"today_target"`
	RecentItems []ChantProgramItem `json:"recent_items"`
}

type StadiumHomeResponse struct {
	UserSummary  UserSummary           `json:"user_summary"`
	LiveMatch    *LiveMatch            `json:"live_match"`
	ChantProgram ChantProgram          `json:"chant_program"`
	Foods        []shopdto.CatalogItem `json:"foods"`
}

// ─── Club Mode (contract §3.2) ────────────────────────────────────────────────

type UpcomingMatch struct {
	ID               uuid.UUID `json:"id"`
	HomeName         string    `json:"home_name"`
	HomeLogoURL      string    `json:"home_logo_url"`
	AwayName         string    `json:"away_name"`
	AwayLogoURL      string    `json:"away_logo_url"`
	Date             string    `json:"date"`
	Time             string    `json:"time"`
	Status           string    `json:"status"`
	CountdownSeconds int64     `json:"countdown_seconds"`
	Score            *string   `json:"score"`
}

type ClubHomeResponse struct {
	UpcomingMatches []UpcomingMatch       `json:"upcoming_matches"`
	ClubStore       []shopdto.CatalogItem `json:"club_store"`
	ClubNews        []newsdto.NewsItem    `json:"club_news"`
}
