package models

import (
	"clap/internal/shared/database"
	"time"

	"github.com/google/uuid"
)

// AdminUser is a staff/operator account for the admin dashboard.
// It is intentionally separate from app (fan) users in the users table.
type AdminUser struct {
	database.BaseModel
	Email     string              `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	FirstName string              `gorm:"type:varchar(100)" json:"first_name"`
	LastName  string              `gorm:"type:varchar(100)" json:"last_name"`
	IsActive  bool                `gorm:"default:true" json:"is_active"`
	RefreshTokens []AdminRefreshToken `gorm:"foreignKey:AdminUserID" json:"refresh_tokens,omitempty"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}

func (a *AdminUser) DisplayName() string {
	name := a.FirstName
	if a.LastName != "" {
		if name != "" {
			name += " "
		}
		name += a.LastName
	}
	if name == "" {
		return a.Email
	}
	return name
}

type AdminRefreshToken struct {
	database.BaseModel
	AdminUserID uuid.UUID  `gorm:"type:uuid;not null;index" json:"admin_user_id"`
	AdminUser   AdminUser  `gorm:"foreignKey:AdminUserID" json:"admin_user,omitempty"`
	Token       string     `gorm:"type:varchar(500);uniqueIndex;not null" json:"-"`
	ExpiresAt   time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt   *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	IPAddress   string     `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent   string     `gorm:"type:varchar(500)" json:"user_agent,omitempty"`
}

func (AdminRefreshToken) TableName() string {
	return "admin_refresh_tokens"
}

func (rt *AdminRefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}

func (rt *AdminRefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

func (rt *AdminRefreshToken) IsValid() bool {
	return !rt.IsRevoked() && !rt.IsExpired()
}
