package models

import (
	"time"

	songmodels "clap/internal/modules/song/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Chant is a scheduled crowd song for a match. Completing a chant awards
// points to the user (Mobile API Contract §4).
type Chant struct {
	ID                  uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID             uuid.UUID       `gorm:"type:uuid;not null" json:"match_id"`
	SongID              uuid.UUID       `gorm:"type:uuid;not null" json:"song_id"`
	Title               string          `gorm:"type:varchar(255);not null" json:"title"`
	Points              int             `gorm:"not null;default:0" json:"points"`
	DurationSeconds     int             `gorm:"not null;default:0" json:"duration_seconds"`
	ScheduledAt         time.Time       `gorm:"type:timestamp;not null" json:"scheduled_at"`
	FlashDurationMs     int             `gorm:"not null;default:500" json:"flash_duration_ms"`
	VibrationDurationMs int             `gorm:"not null;default:500" json:"vibration_duration_ms"`
	IsPreview           bool            `gorm:"not null;default:false" json:"is_preview"`
	IsActive            bool            `gorm:"not null;default:true" json:"is_active"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	DeletedAt           gorm.DeletedAt  `gorm:"index" json:"-"`
	CreatedBy           *uuid.UUID      `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy           *uuid.UUID      `gorm:"type:uuid" json:"updated_by,omitempty"`
	Song                songmodels.Song `gorm:"foreignKey:SongID" json:"song,omitempty"`
}

func (Chant) TableName() string {
	return "chants"
}

// ChantCompletion records that a user finished a chant; unique per user+chant
// so points are awarded exactly once.
type ChantCompletion struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ChantID      uuid.UUID `gorm:"type:uuid;not null" json:"chant_id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	PointsEarned int       `gorm:"not null;default:0" json:"points_earned"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ChantCompletion) TableName() string {
	return "chant_completions"
}
