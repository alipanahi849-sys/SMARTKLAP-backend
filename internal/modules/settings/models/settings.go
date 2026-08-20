package models

import (
	"time"

	clubmodels "clap/internal/modules/club/models"

	"github.com/google/uuid"
)

type AppSettings struct {
	ID             int16      `gorm:"primaryKey;default:1" json:"id"`
	FeaturedClubID *uuid.UUID `gorm:"type:uuid" json:"featured_club_id,omitempty"`
	NewsClubID     *uuid.UUID `gorm:"type:uuid" json:"news_club_id,omitempty"`
	// ChantSongPoints is awarded the first time a user plays a catalog song
	// through to the end from the Chants screen.
	ChantSongPoints int `gorm:"not null;default:100" json:"chant_song_points"`
	// ChantOnlinePoints is awarded for finishing a scheduled online chant. It is
	// deliberately separate from ChantSongPoints.
	ChantOnlinePoints int              `gorm:"not null;default:200" json:"chant_online_points"`
	ChantDailyTarget  int              `gorm:"not null;default:500" json:"chant_daily_target"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	FeaturedClub      *clubmodels.Club `gorm:"foreignKey:FeaturedClubID" json:"featured_club,omitempty"`
	NewsClub          *clubmodels.Club `gorm:"foreignKey:NewsClubID" json:"news_club,omitempty"`
}

func (AppSettings) TableName() string {
	return "app_settings"
}
