package models

import (
	"time"

	clubmodels "clap/internal/modules/club/models"
	leaguemodels "clap/internal/modules/league/models"
	seasonmodels "clap/internal/modules/season/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Match struct {
	ID                 uuid.UUID           `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	LeagueID           uuid.UUID           `gorm:"type:uuid;not null" json:"league_id"`
	SeasonID           uuid.UUID           `gorm:"type:uuid;not null" json:"season_id"`
	HomeClubID         uuid.UUID           `gorm:"type:uuid;not null" json:"home_club_id"`
	AwayClubID         uuid.UUID           `gorm:"type:uuid;not null" json:"away_club_id"`
	Provider           string              `gorm:"type:varchar(50)" json:"provider"`
	ProviderMatchID    string              `gorm:"type:varchar(100)" json:"provider_match_id"`
	MatchDateTime      time.Time           `gorm:"type:timestamp;not null" json:"match_datetime"`
	StadiumName        string              `gorm:"type:varchar(255)" json:"stadium_name"`
	CompetitionLogoURL string              `gorm:"type:varchar(500)" json:"competition_logo_url"`
	Status             string              `gorm:"type:varchar(20);not null;default:'scheduled'" json:"status"`
	HomeScore          *int                `gorm:"type:integer" json:"home_score,omitempty"`
	AwayScore          *int                `gorm:"type:integer" json:"away_score,omitempty"`
	CurrentMinute      string              `gorm:"type:varchar(10)" json:"current_minute,omitempty"`
	DetailsSyncedAt    *time.Time          `gorm:"type:timestamp" json:"details_synced_at,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	DeletedAt          gorm.DeletedAt      `gorm:"index" json:"-"`
	CreatedBy          *uuid.UUID          `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy          *uuid.UUID          `gorm:"type:uuid" json:"updated_by,omitempty"`
	League             leaguemodels.League `gorm:"foreignKey:LeagueID" json:"league,omitempty"`
	Season             seasonmodels.Season `gorm:"foreignKey:SeasonID" json:"season,omitempty"`
	HomeClub           clubmodels.Club     `gorm:"foreignKey:HomeClubID" json:"home_club,omitempty"`
	AwayClub           clubmodels.Club     `gorm:"foreignKey:AwayClubID" json:"away_club,omitempty"`
}

func (Match) TableName() string {
	return "matches"
}
