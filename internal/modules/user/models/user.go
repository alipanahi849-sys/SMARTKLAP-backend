package models

import (
	"clap/internal/modules/auth/models"
	"clap/internal/shared/database"

	"github.com/google/uuid"
)

type Profile struct {
	database.BaseModel
	UserID      uuid.UUID   `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	User        models.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Bio         string      `gorm:"type:text" json:"bio"`
	AvatarURL   string      `gorm:"type:varchar(500)" json:"avatar_url"`
	DateOfBirth *string     `gorm:"type:date" json:"date_of_birth"`
	Country     string      `gorm:"type:varchar(100)" json:"country"`
	City        string      `gorm:"type:varchar(100)" json:"city"`
}

func (Profile) TableName() string {
	return "profiles"
}
