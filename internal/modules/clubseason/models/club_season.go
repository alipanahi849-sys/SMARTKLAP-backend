package models

import (
	"time"

	clubmodels "clap/internal/modules/club/models"
	seasonmodels "clap/internal/modules/season/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClubSeason struct {
	ID        uuid.UUID           `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClubID    uuid.UUID           `gorm:"type:uuid;not null" json:"club_id"`
	SeasonID  uuid.UUID           `gorm:"type:uuid;not null" json:"season_id"`
	JoinedAt  time.Time           `gorm:"type:date;not null" json:"joined_at"`
	Status    string              `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	DeletedAt gorm.DeletedAt      `gorm:"index" json:"-"`
	CreatedBy *uuid.UUID          `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID          `gorm:"type:uuid" json:"updated_by,omitempty"`
	Club      clubmodels.Club     `gorm:"foreignKey:ClubID" json:"club,omitempty"`
	Season    seasonmodels.Season `gorm:"foreignKey:SeasonID" json:"season,omitempty"`
}

func (ClubSeason) TableName() string {
	return "club_seasons"
}
