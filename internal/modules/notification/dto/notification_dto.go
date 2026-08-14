package dto

import (
	"time"

	"github.com/google/uuid"
)

// RegisterDeviceRequest is the body for POST /api/v1/notifications/devices.
type RegisterDeviceRequest struct {
	FCMToken string `json:"fcm_token" binding:"required"`
	Platform string `json:"platform" binding:"required"`
}

// UnregisterDeviceRequest is the body for DELETE /api/v1/notifications/devices.
type UnregisterDeviceRequest struct {
	FCMToken string `json:"fcm_token" binding:"required"`
}

// DeviceResponse is returned after registering an FCM token.
type DeviceResponse struct {
	ID        uuid.UUID `json:"id"`
	FCMToken  string    `json:"fcm_token"`
	Platform  string    `json:"platform"`
	UpdatedAt time.Time `json:"updated_at"`
}
