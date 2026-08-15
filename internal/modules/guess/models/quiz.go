package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	QuizTypeResult = "result"
	QuizTypePlayer = "player"
	QuizTypeCustom = "custom"

	ParticipationPoints = 100
	ResultQuizPoints    = 600
	PlayerQuizPoints    = 200

	ResultQuizTitle = "Result of the game"
	PlayerQuizTitle = "Best Player"

	OptionHome = "home"
	OptionAway = "away"
	OptionDraw = "draw"
)

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

func (Quiz) TableName() string { return "quizzes" }

func (q *Quiz) BeforeCreate(_ *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	now := time.Now().UTC()
	if q.CreatedAt.IsZero() {
		q.CreatedAt = now
	}
	if q.UpdatedAt.IsZero() {
		q.UpdatedAt = now
	}
	return nil
}

type QuizOption struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	QuizID    uuid.UUID `gorm:"type:uuid;not null" json:"quiz_id"`
	Label     string    `gorm:"type:varchar(255);not null" json:"label"`
	Value     string    `gorm:"type:varchar(255);not null" json:"value"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

func (QuizOption) TableName() string { return "quiz_options" }

func (o *QuizOption) BeforeCreate(_ *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}
	return nil
}

type QuizAnswer struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	QuizID       uuid.UUID `gorm:"type:uuid;not null" json:"quiz_id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Choice       string    `gorm:"type:varchar(255);not null" json:"choice"`
	PointsEarned int       `gorm:"not null;default:0" json:"points_earned"`
	CreatedAt    time.Time `json:"created_at"`
}

func (QuizAnswer) TableName() string { return "quiz_answers" }

func (a *QuizAnswer) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	return nil
}

func QuestionForType(quizType, title string) string {
	switch quizType {
	case QuizTypeResult:
		return "What will be the result of this game?"
	case QuizTypePlayer:
		return "Who will be the best player?"
	default:
		if title != "" {
			return title
		}
		return "Make your guess"
	}
}
