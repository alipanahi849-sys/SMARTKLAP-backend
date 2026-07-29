package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RealtimeEventType string

const (
	EventTypeSongStart RealtimeEventType = "song_start"
	EventTypeSongStop  RealtimeEventType = "song_stop"
	EventTypeLyricSync RealtimeEventType = "lyric_sync"
	EventTypeVibrate   RealtimeEventType = "vibrate"
	EventTypeFlash     RealtimeEventType = "flash"
	EventTypeTimerSync RealtimeEventType = "timer_sync"
)

// RealtimeEvent is a single scheduled realtime action within a session.
// ExecuteAtMs is a Unix epoch in milliseconds so clients can compute offsets
// without timezone conversion.
type RealtimeEvent struct {
	ID          uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SessionID   uuid.UUID         `gorm:"type:uuid;not null"                               json:"session_id"`
	EventType   RealtimeEventType `gorm:"type:varchar(50);not null"                        json:"event_type"`
	ExecuteAtMs int64             `gorm:"not null"                                         json:"execute_at_ms"`
	PayloadJSON string            `gorm:"type:jsonb;not null;default:'{}'"                 json:"payload_json"`
	CreatedAt   time.Time         `json:"created_at"`
}

func (RealtimeEvent) TableName() string { return "realtime_events" }

func (r *RealtimeEvent) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
