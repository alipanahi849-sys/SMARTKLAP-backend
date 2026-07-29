package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type League struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name             string         `gorm:"type:varchar(255);not null" json:"name" binding:"required"`
	Country          string         `gorm:"type:varchar(100)" json:"country"`
	Provider         string         `gorm:"type:varchar(50)" json:"provider"`
	ProviderLeagueID string         `gorm:"type:varchar(100)" json:"provider_league_id"`
	IsActive         bool           `gorm:"default:true" json:"is_active"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy        *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy        *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (League) TableName() string {
	return "leagues"
}
