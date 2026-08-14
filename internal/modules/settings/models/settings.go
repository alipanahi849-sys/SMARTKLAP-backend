package models

import (
	"time"

	clubmodels "clap/internal/modules/club/models"

	"github.com/google/uuid"
)

type AppSettings struct {
	ID             int16            `gorm:"primaryKey;default:1" json:"id"`
	FeaturedClubID *uuid.UUID       `gorm:"type:uuid" json:"featured_club_id,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	FeaturedClub   *clubmodels.Club `gorm:"foreignKey:FeaturedClubID" json:"featured_club,omitempty"`
}

func (AppSettings) TableName() string {
	return "app_settings"
}
