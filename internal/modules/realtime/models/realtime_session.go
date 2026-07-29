package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionStatus string

const (
	SessionStatusPending   SessionStatus = "pending"
	SessionStatusRunning   SessionStatus = "running"
	SessionStatusPaused    SessionStatus = "paused"
	SessionStatusCompleted SessionStatus = "completed"
)

// RealtimeSession ties a match to an active realtime broadcasting session.
// One match may have at most one non-completed session at a time.
//
// Version provides optimistic concurrency control: concurrent updates must
// supply the version they last read; a mismatch results in a 409 Conflict.
type RealtimeSession struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID   uuid.UUID      `gorm:"type:uuid;not null"                               json:"match_id"`
	StartedAt *time.Time     `gorm:"type:timestamp"                                   json:"started_at,omitempty"`
	Status    SessionStatus  `gorm:"type:varchar(20);not null;default:'pending'"      json:"status"`
	Version   int64          `gorm:"not null;default:0"                               json:"version"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                            json:"-"`
	CreatedBy *uuid.UUID     `gorm:"type:uuid"                                        json:"created_by,omitempty"`
}

func (RealtimeSession) TableName() string { return "realtime_sessions" }

func (r *RealtimeSession) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
