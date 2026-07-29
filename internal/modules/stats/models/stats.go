package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Player is a club squad member with stats used by the mobile Statistics
// screens (Mobile API Contract §9). RadarStats holds a JSON array of
// {"label","value"} entries.
type Player struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClubID             uuid.UUID      `gorm:"type:uuid;not null" json:"club_id"`
	Name               string         `gorm:"type:varchar(255);not null" json:"name"`
	JerseyNumber       int            `gorm:"not null;default:0" json:"jersey_number"`
	Position           string         `gorm:"type:varchar(50);not null;default:''" json:"position"`
	Age                int            `gorm:"not null;default:0" json:"age"`
	PreferredFoot      string         `gorm:"type:varchar(10);not null;default:''" json:"preferred_foot"`
	Nationality        string         `gorm:"type:varchar(100);not null;default:''" json:"nationality"`
	HeightCm           int            `gorm:"not null;default:0" json:"height_cm"`
	WeightKg           int            `gorm:"not null;default:0" json:"weight_kg"`
	WeakFootPercentage int            `gorm:"not null;default:0" json:"weak_foot_percentage"`
	PhotoURL           string         `gorm:"type:varchar(500)" json:"photo_url"`
	RadarStats         string         `gorm:"type:jsonb;not null;default:'[]'" json:"radar_stats"`
	Formation          string         `gorm:"type:varchar(20);not null;default:''" json:"formation"`
	IsActive           bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy          *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy          *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (Player) TableName() string {
	return "players"
}

// MatchStat is one comparison row (e.g. "Total shots" 95 vs 85).
type MatchStat struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID   uuid.UUID `gorm:"type:uuid;not null" json:"match_id"`
	Label     string    `gorm:"type:varchar(100);not null" json:"label"`
	HomeValue int       `gorm:"not null;default:0" json:"home_value"`
	AwayValue int       `gorm:"not null;default:0" json:"away_value"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (MatchStat) TableName() string {
	return "match_stats"
}

// MatchTimelineEvent is one entry in the match timeline (goal, card, marker…).
type MatchTimelineEvent struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID     uuid.UUID `gorm:"type:uuid;not null" json:"match_id"`
	Kind        string    `gorm:"type:varchar(10);not null" json:"kind"`
	Side        string    `gorm:"type:varchar(10);not null;default:''" json:"side"`
	EventType   string    `gorm:"type:varchar(30);not null;default:''" json:"event_type"`
	PlayerName  string    `gorm:"type:varchar(255);not null;default:''" json:"player_name"`
	Minute      string    `gorm:"type:varchar(10);not null;default:''" json:"minute"`
	Score       string    `gorm:"type:varchar(20);not null;default:''" json:"score"`
	Highlighted bool      `gorm:"not null;default:false" json:"highlighted"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (MatchTimelineEvent) TableName() string {
	return "match_timeline_events"
}
