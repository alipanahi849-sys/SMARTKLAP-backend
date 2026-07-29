package repository

import (
	"context"

	authmodels "clap/internal/modules/auth/models"
	"clap/internal/modules/guess/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QuizRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.Quiz, error)
	FindByMatch(ctx context.Context, matchID uuid.UUID) ([]models.Quiz, error)
	// AnsweredQuizIDs returns which of the given quizzes the user answered.
	AnsweredQuizIDs(ctx context.Context, userID uuid.UUID, quizIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	// Answer stores the user's choice and credits participation points in one
	// transaction. Fails with 409 when the quiz was already answered.
	Answer(ctx context.Context, quizID, userID uuid.UUID, choice string, points int) error
}

type quizRepository struct {
	db *gorm.DB
}

func NewQuizRepository(db *gorm.DB) QuizRepository {
	return &quizRepository{db: db}
}

func (r *quizRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Quiz, error) {
	var quiz models.Quiz
	err := r.db.WithContext(ctx).
		Preload("Options", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order ASC") }).
		First(&quiz, "id = ? AND is_active = ?", id, true).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Quiz not found", nil)
		}
		return nil, errors.NewInternal("Failed to find quiz", err)
	}
	return &quiz, nil
}

func (r *quizRepository) FindByMatch(ctx context.Context, matchID uuid.UUID) ([]models.Quiz, error) {
	var quizzes []models.Quiz
	err := r.db.WithContext(ctx).
		Where("match_id = ? AND is_active = ?", matchID, true).
		Order("created_at ASC").
		Find(&quizzes).Error
	if err != nil {
		return nil, errors.NewInternal("Failed to list quizzes", err)
	}
	return quizzes, nil
}

func (r *quizRepository) AnsweredQuizIDs(ctx context.Context, userID uuid.UUID, quizIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	answered := make(map[uuid.UUID]bool, len(quizIDs))
	if len(quizIDs) == 0 {
		return answered, nil
	}

	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&models.QuizAnswer{}).
		Where("user_id = ? AND quiz_id IN ?", userID, quizIDs).
		Pluck("quiz_id", &ids).Error; err != nil {
		return nil, errors.NewInternal("Failed to load answers", err)
	}
	for _, id := range ids {
		answered[id] = true
	}
	return answered, nil
}

func (r *quizRepository) Answer(ctx context.Context, quizID, userID uuid.UUID, choice string, points int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		answer := models.QuizAnswer{
			QuizID:       quizID,
			UserID:       userID,
			Choice:       choice,
			PointsEarned: points,
		}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&answer)
		if res.Error != nil {
			return errors.NewInternal("Failed to store answer", res.Error)
		}
		if res.RowsAffected == 0 {
			return errors.NewConflict("Quiz already answered", nil)
		}

		if points > 0 {
			if err := tx.Model(&authmodels.User{}).
				Where("id = ?", userID).
				UpdateColumn("points", gorm.Expr("points + ?", points)).Error; err != nil {
				return errors.NewInternal("Failed to award points", err)
			}
		}
		return nil
	})
}
