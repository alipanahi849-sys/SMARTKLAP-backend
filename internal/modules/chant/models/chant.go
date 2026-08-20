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

// Chant sources. A chant either comes from the predefined song catalog on the
// Chants screen, or from a scheduled online chant an admin set for a match.
const (
	SourceCatalog = "catalog"
	SourceOnline  = "online"
)

// NormalizeSource maps anything unrecognised onto the online source, matching
// the client default.
func NormalizeSource(raw string) string {
	if raw == SourceCatalog {
		return SourceCatalog
	}
	return SourceOnline
}

// ChantCompletion records that a user played a song through to the end.
// Catalog completions are unique per user+song and carry no chant; online
// completions are unique per user+chant. Either way points land exactly once.
type ChantCompletion struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ChantID      *uuid.UUID `gorm:"type:uuid" json:"chant_id,omitempty"`
	SongID       *uuid.UUID `gorm:"type:uuid" json:"song_id,omitempty"`
	Source       string     `gorm:"type:varchar(16);not null;default:online" json:"source"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	PointsEarned int        `gorm:"not null;default:0" json:"points_earned"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (ChantCompletion) TableName() string {
	return "chant_completions"
}

// ChantListenSession is stamped when a user fetches lyrics. Completing a chant
// is only credited once at least the track's length has passed since then, so
// points cannot be claimed by skipping to the end.
type ChantListenSession struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	SongID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"song_id"`
	Source    string    `gorm:"type:varchar(16);primaryKey" json:"source"`
	StartedAt time.Time `json:"started_at"`
}

func (ChantListenSession) TableName() string {
	return "chant_listen_sessions"
}
