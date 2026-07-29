package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Quiz type discriminators (contract §5.2 — result quizzes take a side/draw
// choice, player quizzes take a player_id).
const (
	QuizTypeResult = "result"
	QuizTypePlayer = "player"
	QuizTypeCustom = "custom"
)

// Quiz is a per-match prediction question (Mobile API Contract §5).
type Quiz struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	MatchID   uuid.UUID      `gorm:"type:uuid;not null" json:"match_id"`
	Title     string         `gorm:"type:varchar(255);not null" json:"title"`
	QuizType  string         `gorm:"type:varchar(30);not null;default:'result'" json:"quiz_type"`
	Points    int            `gorm:"not null;default:0" json:"points"`
	IsActive  bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
	Options   []QuizOption   `gorm:"foreignKey:QuizID" json:"options,omitempty"`
}

func (Quiz) TableName() string {
	return "quizzes"
}

// QuizOption is a selectable answer for a quiz.
type QuizOption struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	QuizID    uuid.UUID `gorm:"type:uuid;not null" json:"quiz_id"`
	Label     string    `gorm:"type:varchar(255);not null" json:"label"`
	Value     string    `gorm:"type:varchar(255);not null" json:"value"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

func (QuizOption) TableName() string {
	return "quiz_options"
}

// QuizAnswer is a user's submission; unique per user+quiz.
type QuizAnswer struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	QuizID       uuid.UUID `gorm:"type:uuid;not null" json:"quiz_id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Choice       string    `gorm:"type:varchar(255);not null" json:"choice"`
	PointsEarned int       `gorm:"not null;default:0" json:"points_earned"`
	CreatedAt    time.Time `json:"created_at"`
}

func (QuizAnswer) TableName() string {
	return "quiz_answers"
}
