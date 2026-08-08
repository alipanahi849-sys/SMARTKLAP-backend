package models

import (
	"time"

	"github.com/google/uuid"
)

type StripePaymentEvent struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EventID   string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"event_id"`
	EventType string     `gorm:"type:varchar(100);not null" json:"event_type"`
	OrderID   *uuid.UUID `gorm:"type:uuid" json:"order_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (StripePaymentEvent) TableName() string { return "stripe_payment_events" }
