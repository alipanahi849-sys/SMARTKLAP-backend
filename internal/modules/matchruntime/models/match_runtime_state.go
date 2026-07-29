package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RuntimeStatus string

const (
	RuntimeStatusPending RuntimeStatus = "pending"
	RuntimeStatusRunning RuntimeStatus = "running"
	RuntimeStatusPaused  RuntimeStatus = "paused"
	RuntimeStatusEnded   RuntimeStatus = "ended"
)

// MatchRuntimeState is the durable timer record for a match.
// Elapsed time is computed from first-principles on every read, so the
// timer is correct even after server restarts or long pauses.
//
// elapsed = (reference_point - started_at) - total_paused_ms
// where reference_point is:
//   - now()            when status = running
//   - paused_at        when status = paused
//   - ended_at         when status = ended
//
// Version is used for optimistic concurrency control — concurrent writers
// must provide the version they last read; a mismatch returns 409 Conflict.
type MatchRuntimeState struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID       uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex"                   json:"match_id"`
	Status        RuntimeStatus  `gorm:"type:varchar(20);not null;default:'pending'"      json:"status"`
	StartedAt     *time.Time     `gorm:"type:timestamp"                                   json:"started_at,omitempty"`
	PausedAt      *time.Time     `gorm:"type:timestamp"                                   json:"paused_at,omitempty"`
	EndedAt       *time.Time     `gorm:"type:timestamp"                                   json:"ended_at,omitempty"`
	TotalPausedMs int64          `gorm:"not null;default:0"                               json:"total_paused_ms"`
	Version       int64          `gorm:"not null;default:0"                               json:"version"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index"                                            json:"-"`
	CreatedBy     *uuid.UUID     `gorm:"type:uuid"                                        json:"created_by,omitempty"`
	UpdatedBy     *uuid.UUID     `gorm:"type:uuid"                                        json:"updated_by,omitempty"`
}

func (MatchRuntimeState) TableName() string { return "match_runtime_states" }

func (m *MatchRuntimeState) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
