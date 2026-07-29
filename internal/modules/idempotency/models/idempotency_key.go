package models

import (
	"time"

	"github.com/google/uuid"
)

// IdempotencyKey stores the result of a previously executed mutating request.
// When a client replays a request with the same X-Idempotency-Key header, the
// stored response is returned immediately without re-executing the handler.
//
// Expiry: records are considered stale once expires_at is past.
// Matching: both key + endpoint must match; if request_hash differs the
// client sent a different body with the same key — this is rejected.
type IdempotencyKey struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Key             string    `gorm:"type:varchar(255);not null;uniqueIndex:uidx_idempotency_key_endpoint,priority:1" json:"key"`
	Endpoint        string    `gorm:"type:varchar(255);not null;uniqueIndex:uidx_idempotency_key_endpoint,priority:2" json:"endpoint"`
	RequestHash     string    `gorm:"type:varchar(64);not null"                        json:"request_hash"`
	ResponsePayload string    `gorm:"type:text;not null"                               json:"response_payload"`
	StatusCode      int       `gorm:"not null;default:200"                             json:"status_code"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `gorm:"not null"                                         json:"expires_at"`
}

func (IdempotencyKey) TableName() string { return "idempotency_keys" }

func (k *IdempotencyKey) BeforeCreate(_ interface{}) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}

// IsExpired reports whether this record should no longer be used as a replay cache.
func (k *IdempotencyKey) IsExpired() bool {
	return time.Now().UTC().After(k.ExpiresAt)
}
