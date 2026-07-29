package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlaybackStatus string

const (
	PlaybackStatusPending   PlaybackStatus = "pending"
	PlaybackStatusActive    PlaybackStatus = "active"
	PlaybackStatusCompleted PlaybackStatus = "completed"
	PlaybackStatusCancelled PlaybackStatus = "cancelled"
)

// PlaybackSchedule represents a song scheduled to play at a specific wall-clock
// time within a match. No audio streaming — this is an event scheduling record only.
//
// DurationMs defines the expected playback window length. Together with
// ScheduledAt it allows overlap detection: a new schedule must not intersect
// [ScheduledAt, ScheduledAt+DurationMs) of any existing active schedule for
// the same match.
//
// Version provides optimistic concurrency control.
type PlaybackSchedule struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID     uuid.UUID      `gorm:"type:uuid;not null"                               json:"match_id"`
	SongID      uuid.UUID      `gorm:"type:uuid;not null"                               json:"song_id"`
	ScheduledAt time.Time      `gorm:"type:timestamp;not null"                          json:"scheduled_at"`
	DurationMs  int64          `gorm:"not null;default:0"                               json:"duration_ms"`
	Status      PlaybackStatus `gorm:"type:varchar(20);not null;default:'pending'"      json:"status"`
	Version     int64          `gorm:"not null;default:0"                               json:"version"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index"                                            json:"-"`
	CreatedBy   *uuid.UUID     `gorm:"type:uuid"                                        json:"created_by,omitempty"`
}

func (PlaybackSchedule) TableName() string { return "playback_schedules" }

func (p *PlaybackSchedule) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
