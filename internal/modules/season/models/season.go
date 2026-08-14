package models

import (
	"time"

	"clap/internal/modules/league/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Season struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	LeagueID         uuid.UUID      `gorm:"type:uuid;not null" json:"league_id"`
	Name             string         `gorm:"type:varchar(255);not null" json:"name" binding:"required"`
	StartDate        time.Time      `gorm:"type:date;not null" json:"start_date"`
	EndDate          time.Time      `gorm:"type:date;not null" json:"end_date"`
	ProviderSeasonID string         `gorm:"type:varchar(100)" json:"provider_season_id"`
	IsActive         bool           `gorm:"default:true" json:"is_active"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy        *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy        *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
	League           models.League  `gorm:"foreignKey:LeagueID" json:"league,omitempty"`
}

func (Season) TableName() string {
	return "seasons"
}
