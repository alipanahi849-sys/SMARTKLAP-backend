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

type GuessRepository interface {
	ListByMatchID(ctx context.Context, matchID uuid.UUID) ([]models.Quiz, error)
	FindByID(ctx context.Context, quizID uuid.UUID) (*models.Quiz, error)
	CreateWithOptions(ctx context.Context, quiz *models.Quiz, options []models.QuizOption) error
	AnsweredQuizIDs(ctx context.Context, userID uuid.UUID, quizIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	FindAnswer(ctx context.Context, quizID, userID uuid.UUID) (*models.QuizAnswer, error)
	SubmitAnswer(ctx context.Context, answer *models.QuizAnswer) (created bool, err error)
}

type guessRepository struct {
	db *gorm.DB
}

func NewGuessRepository(db *gorm.DB) GuessRepository {
	return &guessRepository{db: db}
}

func (r *guessRepository) ListByMatchID(ctx context.Context, matchID uuid.UUID) ([]models.Quiz, error) {
	var quizzes []models.Quiz
	err := r.db.WithContext(ctx).
		Where("match_id = ? AND is_active = ?", matchID, true).
		Preload("Options", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, created_at ASC")
		}).
		Order("created_at ASC").
		Find(&quizzes).Error
	if err != nil {
		return nil, errors.NewInternal("Failed to list quizzes", err)
	}
	return quizzes, nil
}

func (r *guessRepository) FindByID(ctx context.Context, quizID uuid.UUID) (*models.Quiz, error) {
	var quiz models.Quiz
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_active = ?", quizID, true).
		Preload("Options", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, created_at ASC")
		}).
		First(&quiz).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Quiz not found", nil)
		}
		return nil, errors.NewInternal("Failed to load quiz", err)
	}
	return &quiz, nil
}

func (r *guessRepository) CreateWithOptions(ctx context.Context, quiz *models.Quiz, options []models.QuizOption) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(quiz).Error; err != nil {
			return errors.NewInternal("Failed to create quiz", err)
		}
		if len(options) == 0 {
			return nil
		}
		for i := range options {
			options[i].QuizID = quiz.ID
		}
		if err := tx.Create(&options).Error; err != nil {
			return errors.NewInternal("Failed to create quiz options", err)
		}
		quiz.Options = options
		return nil
	})
}

func (r *guessRepository) AnsweredQuizIDs(ctx context.Context, userID uuid.UUID, quizIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := make(map[uuid.UUID]bool, len(quizIDs))
	if len(quizIDs) == 0 {
		return out, nil
	}
	var answers []models.QuizAnswer
	if err := r.db.WithContext(ctx).
		Select("quiz_id").
		Where("user_id = ? AND quiz_id IN ?", userID, quizIDs).
		Find(&answers).Error; err != nil {
		return nil, errors.NewInternal("Failed to load quiz answers", err)
	}
	for _, answer := range answers {
		out[answer.QuizID] = true
	}
	return out, nil
}

func (r *guessRepository) FindAnswer(ctx context.Context, quizID, userID uuid.UUID) (*models.QuizAnswer, error) {
	var answer models.QuizAnswer
	err := r.db.WithContext(ctx).
		Where("quiz_id = ? AND user_id = ?", quizID, userID).
		First(&answer).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.NewInternal("Failed to load quiz answer", err)
	}
	return &answer, nil
}

func (r *guessRepository) SubmitAnswer(ctx context.Context, answer *models.QuizAnswer) (bool, error) {
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "quiz_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).Create(answer)
		if res.Error != nil {
			return errors.NewInternal("Failed to submit quiz answer", res.Error)
		}
		if res.RowsAffected == 0 {
			return nil
		}
		created = true
		if answer.PointsEarned == 0 {
			return nil
		}
		if err := tx.Model(&authmodels.User{}).
			Where("id = ?", answer.UserID).
			UpdateColumn("points", gorm.Expr("points + ?", answer.PointsEarned)).Error; err != nil {
			return errors.NewInternal("Failed to award participation points", err)
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return created, nil
}
