package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Club struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name           string         `gorm:"type:varchar(255);not null" json:"name" binding:"required"`
	ShortName      string         `gorm:"type:varchar(50)" json:"short_name"`
	Description    string         `gorm:"type:text" json:"description"`
	LogoURL        string         `gorm:"type:varchar(500)" json:"logo_url"`
	Country        string         `gorm:"type:varchar(100)" json:"country"`
	VenueName      string         `gorm:"type:varchar(255)" json:"venue_name"`
	Provider       string         `gorm:"type:varchar(50)" json:"provider"`
	ProviderTeamID string         `gorm:"type:varchar(100)" json:"provider_team_id"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy      *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy      *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (Club) TableName() string {
	return "clubs"
}
