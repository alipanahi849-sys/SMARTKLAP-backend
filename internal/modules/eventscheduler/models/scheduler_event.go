package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchedulerEventStatus string

const (
	// SchedulerEventPending is the initial state — event is waiting to fire.
	SchedulerEventPending SchedulerEventStatus = "pending"
	// SchedulerEventProcessing is set atomically when a worker claims the event.
	// Using FOR UPDATE SKIP LOCKED ensures only one worker processes it.
	SchedulerEventProcessing SchedulerEventStatus = "processing"
	// SchedulerEventExecuted marks successful completion.
	SchedulerEventExecuted SchedulerEventStatus = "executed"
	// SchedulerEventCancelled is set when the event is explicitly withdrawn.
	SchedulerEventCancelled SchedulerEventStatus = "cancelled"
	// SchedulerEventFailed marks a terminal error after processing attempts.
	SchedulerEventFailed SchedulerEventStatus = "failed"
)

// SchedulerEvent is the durable, database-backed record of a scheduled action.
// The in-memory priority queue is populated from this table on startup so that
// events survive process restarts.
//
// Execution safety:
//
//	pending → processing : claimed atomically via FOR UPDATE SKIP LOCKED
//	processing → executed: marked after successful dispatch
//	processing → failed  : marked after an unrecoverable error
//
// This prevents double-execution, lost execution, and concurrent execution.
type SchedulerEvent struct {
	ID          uuid.UUID            `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SessionID   uuid.UUID            `gorm:"type:uuid;not null"                               json:"session_id"`
	EventType   string               `gorm:"type:varchar(50);not null"                        json:"event_type"`
	ExecuteAt   time.Time            `gorm:"type:timestamp;not null"                          json:"execute_at"`
	PayloadJSON string               `gorm:"type:jsonb;not null;default:'{}'"                 json:"payload_json"`
	Status      SchedulerEventStatus `gorm:"type:varchar(20);not null;default:'pending'"      json:"status"`
	FailReason  string               `gorm:"type:text"                                        json:"fail_reason,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	DeletedAt   gorm.DeletedAt       `gorm:"index"                                            json:"-"`
}

func (SchedulerEvent) TableName() string { return "scheduler_events" }

func (s *SchedulerEvent) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
