package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
)

// PushDevice is one FCM registration for a user's phone or tablet.
type PushDevice struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	FCMToken  string    `gorm:"column:fcm_token;type:text;not null;uniqueIndex" json:"fcm_token"`
	Platform  string    `gorm:"type:varchar(20);not null" json:"platform"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PushDevice) TableName() string {
	return "push_devices"
}
