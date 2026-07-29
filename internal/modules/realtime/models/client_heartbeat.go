package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClientHeartbeat records a single clock-sync round-trip from a client.
// DriftMs = ServerTimestamp - ClientTimestamp (positive = client is behind).
type ClientHeartbeat struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SessionID       uuid.UUID `gorm:"type:uuid;not null"                               json:"session_id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null"                               json:"user_id"`
	ClientTimestamp int64     `gorm:"not null"                                         json:"client_timestamp"`
	ServerTimestamp int64     `gorm:"not null"                                         json:"server_timestamp"`
	DriftMs         int64     `gorm:"not null"                                         json:"drift_ms"`
	CreatedAt       time.Time `json:"created_at"`
}

func (ClientHeartbeat) TableName() string { return "client_heartbeats" }

func (c *ClientHeartbeat) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
