package models

import (
	"time"

	songmodels "clap/internal/modules/song/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SongLyric struct {
	ID        uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SongID    uuid.UUID       `gorm:"type:uuid;not null" json:"song_id"`
	Language  string          `gorm:"type:varchar(10);not null" json:"language"`
	Lyrics    string          `gorm:"type:text;not null" json:"lyrics"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt gorm.DeletedAt  `gorm:"index" json:"-"`
	CreatedBy *uuid.UUID      `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID      `gorm:"type:uuid" json:"updated_by,omitempty"`
	Song      songmodels.Song `gorm:"foreignKey:SongID" json:"song,omitempty"`
}

func (SongLyric) TableName() string {
	return "song_lyrics"
}
