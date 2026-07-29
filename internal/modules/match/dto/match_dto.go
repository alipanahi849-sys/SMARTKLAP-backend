package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type CreateMatchRequest struct {
	LeagueID        uuid.UUID `json:"league_id" binding:"required"`
	SeasonID        uuid.UUID `json:"season_id" binding:"required"`
	HomeClubID      uuid.UUID `json:"home_club_id" binding:"required"`
	AwayClubID      uuid.UUID `json:"away_club_id" binding:"required"`
	Provider        string    `json:"provider"`
	ProviderMatchID string    `json:"provider_match_id"`
	MatchDateTime   string    `json:"match_datetime" binding:"required"`
	StadiumName     string    `json:"stadium_name"`
	Status          string    `json:"status" binding:"required,oneof=scheduled live halftime finished cancelled"`
}

type UpdateMatchRequest struct {
	Provider        string  `json:"provider"`
	ProviderMatchID string  `json:"provider_match_id"`
	MatchDateTime   string  `json:"match_datetime" binding:"required"`
	StadiumName     string  `json:"stadium_name"`
	Status          string  `json:"status" binding:"required,oneof=scheduled live halftime finished cancelled"`
	HomeScore       *int    `json:"home_score" binding:"omitempty,min=0"`
	AwayScore       *int    `json:"away_score" binding:"omitempty,min=0"`
	CurrentMinute   *string `json:"current_minute" binding:"omitempty,max=10"`
}

// MatchTeamInfo identifies one side of a match for the mobile detail screen.
type MatchTeamInfo struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	LogoURL string    `json:"logo_url"`
}

// MatchStatRow is one comparison row of the Statistics tab (contract §9.1).
type MatchStatRow struct {
	Label string `json:"label"`
	Home  int    `json:"home"`
	Away  int    `json:"away"`
}

// MatchTimelineItem is one entry of the Game tab timeline (contract §9.1).
type MatchTimelineItem struct {
	Kind        string `json:"kind"`
	Side        string `json:"side,omitempty"`
	EventType   string `json:"event_type,omitempty"`
	PlayerName  string `json:"player_name,omitempty"`
	Minute      string `json:"minute,omitempty"`
	Score       string `json:"score,omitempty"`
	Highlighted bool   `json:"highlighted"`
}

// SquadPlayer is one entry of the Squads tab (contract §9.1).
type SquadPlayer struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	JerseyNumber int       `json:"jersey_number"`
	Position     string    `json:"position"`
	PhotoURL     string    `json:"photo_url"`
}

// SquadGroup groups squad players by position section.
type SquadGroup struct {
	Title   string        `json:"title"`
	Players []SquadPlayer `json:"players"`
}

type MatchResponse struct {
	ID              uuid.UUID `json:"id"`
	LeagueID        uuid.UUID `json:"league_id"`
	SeasonID        uuid.UUID `json:"season_id"`
	HomeClubID      uuid.UUID `json:"home_club_id"`
	AwayClubID      uuid.UUID `json:"away_club_id"`
	Provider        string    `json:"provider"`
	ProviderMatchID string    `json:"provider_match_id"`
	MatchDateTime   string    `json:"match_datetime"`
	StadiumName     string    `json:"stadium_name"`
	Status          string    `json:"status"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`

	// Mobile detail extension (contract §9.1). Populated only on GetByID so
	// list endpoints keep their existing shape and query cost.
	HomeScore *int                `json:"home_score,omitempty"`
	AwayScore *int                `json:"away_score,omitempty"`
	Minute    string              `json:"minute,omitempty"`
	HomeTeam  *MatchTeamInfo      `json:"home_team,omitempty"`
	AwayTeam  *MatchTeamInfo      `json:"away_team,omitempty"`
	Stats     []MatchStatRow      `json:"stats,omitempty"`
	Timeline  []MatchTimelineItem `json:"timeline,omitempty"`
	Squads    []SquadGroup        `json:"squads,omitempty"`
}

type MatchListResponse struct {
	Data       []MatchResponse          `json:"data"`
	Pagination utils.PaginationResponse `json:"pagination"`
}
