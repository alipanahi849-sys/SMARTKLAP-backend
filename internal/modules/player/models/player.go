package models

import (
	"time"

	clubmodels "clap/internal/modules/club/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RadarStat struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type Player struct {
	ID                 uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClubID             uuid.UUID       `gorm:"type:uuid;not null" json:"club_id"`
	Name               string          `gorm:"type:varchar(255);not null" json:"name"`
	JerseyNumber       int             `gorm:"type:integer;not null;default:0" json:"jersey_number"`
	Position           string          `gorm:"type:varchar(50);not null;default:''" json:"position"`
	Age                int             `gorm:"type:integer;not null;default:0" json:"age"`
	PreferredFoot      string          `gorm:"type:varchar(10);not null;default:''" json:"preferred_foot"`
	Nationality        string          `gorm:"type:varchar(100);not null;default:''" json:"nationality"`
	HeightCM           int             `gorm:"type:integer;not null;default:0" json:"height_cm"`
	WeightKG           int             `gorm:"type:integer;not null;default:0" json:"weight_kg"`
	WeakFootPercentage int             `gorm:"type:integer;not null;default:0" json:"weak_foot_percentage"`
	PhotoURL           string          `gorm:"type:varchar(500)" json:"photo_url"`
	RadarStats         []RadarStat     `gorm:"type:jsonb;serializer:json" json:"radar_stats"`
	Formation          string          `gorm:"type:varchar(20);not null;default:''" json:"formation"`
	Provider           string          `gorm:"type:varchar(50)" json:"provider"`
	ProviderPlayerID   string          `gorm:"type:varchar(100)" json:"provider_player_id"`
	IsActive           bool            `gorm:"not null;default:true" json:"is_active"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	DeletedAt          gorm.DeletedAt  `gorm:"index" json:"-"`
	CreatedBy          *uuid.UUID      `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy          *uuid.UUID      `gorm:"type:uuid" json:"updated_by,omitempty"`
	Club               clubmodels.Club `gorm:"foreignKey:ClubID" json:"club,omitempty"`
}

func (Player) TableName() string {
	return "players"
}
