package models

import (
	"time"

	matchmodels "clap/internal/modules/match/models"
	songmodels "clap/internal/modules/song/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchSongSchedule struct {
	ID            uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID       uuid.UUID         `gorm:"type:uuid;not null" json:"match_id"`
	SongID        uuid.UUID         `gorm:"type:uuid;not null" json:"song_id"`
	ScheduledTime time.Time         `gorm:"type:timestamp;not null" json:"scheduled_time"`
	EventType     string            `gorm:"type:varchar(50);not null" json:"event_type"` // e.g., "goal", "kickoff", "halftime", "fulltime"
	IsActive      bool              `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	DeletedAt     gorm.DeletedAt    `gorm:"index" json:"-"`
	CreatedBy     *uuid.UUID        `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy     *uuid.UUID        `gorm:"type:uuid" json:"updated_by,omitempty"`
	Match         matchmodels.Match `gorm:"foreignKey:MatchID" json:"match,omitempty"`
	Song          songmodels.Song   `gorm:"foreignKey:SongID" json:"song,omitempty"`
}

func (MatchSongSchedule) TableName() string {
	return "match_song_schedules"
}
