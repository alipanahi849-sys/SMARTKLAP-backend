package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Song struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title     string         `gorm:"type:varchar(255);not null" json:"title" binding:"required"`
	Artist    string         `gorm:"type:varchar(255)" json:"artist"`
	Album     string         `gorm:"type:varchar(255)" json:"album"`
	Duration  int            `gorm:"type:integer" json:"duration"` // in seconds
	AudioURL  string         `gorm:"type:varchar(500)" json:"audio_url"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
	// Media fields
	MediaFileID *uuid.UUID `gorm:"type:uuid" json:"media_file_id,omitempty"`
	StorageKey  string     `gorm:"type:varchar(500)" json:"storage_key,omitempty"`
	MimeType    string     `gorm:"type:varchar(100)" json:"mime_type,omitempty"`
	FileSize    int64      `gorm:"type:bigint" json:"file_size,omitempty"`
	DurationMs  int64      `gorm:"type:bigint" json:"duration_ms,omitempty"`
	Bitrate     int        `gorm:"type:integer" json:"bitrate,omitempty"`
	SampleRate  int        `gorm:"type:integer" json:"sample_rate,omitempty"`
}

func (Song) TableName() string {
	return "songs"
}
