package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type News struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ClubID      *uuid.UUID     `gorm:"type:uuid" json:"club_id,omitempty"`
	Title       string         `gorm:"type:varchar(500);not null" json:"title"`
	BodyHTML    string         `gorm:"column:body_html;type:text" json:"body_html"`
	ImageURL    string         `gorm:"type:varchar(500)" json:"image_url"`
	PublishedAt time.Time      `gorm:"type:timestamp;not null" json:"published_at"`
	IsActive    bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy   *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy   *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (News) TableName() string {
	return "news"
}
