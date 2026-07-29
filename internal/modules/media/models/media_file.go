package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MediaFile struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	StorageKey       string         `gorm:"type:varchar(500);not null;unique" json:"storage_key"`
	OriginalFileName string         `gorm:"type:varchar(255);not null" json:"original_file_name"`
	MimeType         string         `gorm:"type:varchar(100);not null" json:"mime_type"`
	FileSize         int64          `gorm:"type:bigint;not null" json:"file_size"`
	Checksum         string         `gorm:"type:varchar(64);not null;unique" json:"checksum"`
	UploadedBy       uuid.UUID      `gorm:"type:uuid;not null" json:"uploaded_by"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MediaFile) TableName() string {
	return "media_files"
}
