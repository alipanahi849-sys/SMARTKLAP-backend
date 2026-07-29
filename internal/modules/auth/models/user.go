package models

import (
	"clap/internal/shared/database"
	"time"

	"github.com/google/uuid"
)

type User struct {
	database.BaseModel
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	FirstName    string         `gorm:"type:varchar(100)" json:"first_name"`
	LastName     string         `gorm:"type:varchar(100)" json:"last_name"`
	Phone        string         `gorm:"type:varchar(20)" json:"phone"`
	Points       int            `gorm:"not null;default:0" json:"points"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	IsVerified   bool           `gorm:"default:false" json:"is_verified"`
	Roles        []Role         `gorm:"many2many:user_roles;" json:"roles,omitempty"`
	RefreshToken []RefreshToken `gorm:"foreignKey:UserID" json:"refresh_tokens,omitempty"`
}

// DisplayName returns the mobile-facing single "name" field, combining the
// stored first/last name parts.
func (u *User) DisplayName() string {
	name := u.FirstName
	if u.LastName != "" {
		if name != "" {
			name += " "
		}
		name += u.LastName
	}
	return name
}

func (User) TableName() string {
	return "users"
}

type Role struct {
	database.BaseModel
	Name        string `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Users       []User `gorm:"many2many:user_roles;" json:"users,omitempty"`
}

func (Role) TableName() string {
	return "roles"
}

type RefreshToken struct {
	database.BaseModel
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	User      User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Token     string     `gorm:"type:varchar(500);uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	IPAddress string     `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent string     `gorm:"type:varchar(500)" json:"user_agent,omitempty"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}

func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

func (rt *RefreshToken) IsValid() bool {
	return !rt.IsRevoked() && !rt.IsExpired()
}
