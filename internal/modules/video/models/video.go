package models

import (
	"time"

	authmodels "clap/internal/modules/auth/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Media type discriminators (contract §8.3 — the New Post screen uploads
// either an image or a video).
const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
)

// Video statuses.
const (
	StatusProcessing = "processing"
	StatusPublished  = "published"
	StatusRejected   = "rejected"
)

// Video is a user-generated feed post (Mobile API Contract §8).
type Video struct {
	ID           uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID       `gorm:"type:uuid;not null" json:"user_id"`
	MediaType    string          `gorm:"type:varchar(10);not null" json:"media_type"`
	Caption      string          `gorm:"type:text" json:"caption"`
	Tags         string          `gorm:"type:jsonb;not null;default:'[]'" json:"tags"`
	StorageKey   string          `gorm:"type:varchar(500);not null" json:"storage_key"`
	ThumbnailKey string          `gorm:"type:varchar(500);not null;default:''" json:"thumbnail_key"`
	MimeType     string          `gorm:"type:varchar(100);not null" json:"mime_type"`
	FileSize     int64           `gorm:"not null;default:0" json:"file_size"`
	Status       string          `gorm:"type:varchar(20);not null;default:'published'" json:"status"`
	LikesCount   int             `gorm:"not null;default:0" json:"likes_count"`
	ViewsCount   int             `gorm:"not null;default:0" json:"views_count"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	DeletedAt    gorm.DeletedAt  `gorm:"index" json:"-"`
	User         authmodels.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Video) TableName() string {
	return "videos"
}

// VideoLike is a user's like on a video (composite PK, one like per user).
type VideoLike struct {
	VideoID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"video_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (VideoLike) TableName() string {
	return "video_likes"
}

// VideoView records that a user has seen a video (composite PK, one view per user).
type VideoView struct {
	VideoID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"video_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (VideoView) TableName() string {
	return "video_views"
}
