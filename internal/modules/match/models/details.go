package models

import (
	"time"

	"github.com/google/uuid"
)

type MatchStat struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID   uuid.UUID `gorm:"type:uuid;not null" json:"match_id"`
	Label     string    `gorm:"type:varchar(100);not null" json:"label"`
	HomeValue int       `gorm:"type:integer;not null;default:0" json:"home_value"`
	AwayValue int       `gorm:"type:integer;not null;default:0" json:"away_value"`
	SortOrder int       `gorm:"type:integer;not null;default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (MatchStat) TableName() string {
	return "match_stats"
}

type MatchTimelineEvent struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID       uuid.UUID `gorm:"type:uuid;not null" json:"match_id"`
	Kind          string    `gorm:"type:varchar(10);not null" json:"kind"`
	Side          string    `gorm:"type:varchar(10);not null;default:''" json:"side"`
	EventType     string    `gorm:"type:varchar(30);not null;default:''" json:"event_type"`
	PlayerName    string    `gorm:"type:varchar(255);not null;default:''" json:"player_name"`
	SubPlayerName string    `gorm:"type:varchar(255);not null;default:''" json:"sub_player_name"`
	Minute        string    `gorm:"type:varchar(10);not null;default:''" json:"minute"`
	Score         string    `gorm:"type:varchar(20);not null;default:''" json:"score"`
	Highlighted   bool      `gorm:"not null;default:false" json:"highlighted"`
	SortOrder     int       `gorm:"type:integer;not null;default:0" json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (MatchTimelineEvent) TableName() string {
	return "match_timeline_events"
}

type MatchLineupPlayer struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID      uuid.UUID  `gorm:"type:uuid;not null" json:"match_id"`
	ClubID       uuid.UUID  `gorm:"type:uuid;not null" json:"club_id"`
	PlayerID     *uuid.UUID `gorm:"type:uuid" json:"player_id,omitempty"`
	Side         string     `gorm:"type:varchar(10);not null" json:"side"`
	Name         string     `gorm:"type:varchar(255);not null;default:''" json:"name"`
	Position     string     `gorm:"type:varchar(50);not null;default:''" json:"position"`
	JerseyNumber int        `gorm:"type:integer;not null;default:0" json:"jersey_number"`
	PhotoURL     string     `gorm:"type:varchar(500)" json:"photo_url"`
	IsStarter    bool       `gorm:"not null;default:true" json:"is_starter"`
	SortOrder    int        `gorm:"type:integer;not null;default:0" json:"sort_order"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (MatchLineupPlayer) TableName() string {
	return "match_lineup_players"
}
