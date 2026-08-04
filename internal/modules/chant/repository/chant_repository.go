package repository

import (
	"context"
	"strings"
	"time"

	"clap/internal/modules/chant/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChantRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.Chant, error)
	// FindByMatch returns active chants for a match ordered by schedule time,
	// optionally filtered by a title search term.
	FindByMatch(ctx context.Context, matchID uuid.UUID, search string) ([]models.Chant, error)
	// CompletedChantIDs returns which of the given chants the user completed.
	CompletedChantIDs(ctx context.Context, userID uuid.UUID, chantIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	// TodayPoints sums the user's chant points earned since local midnight (UTC).
	TodayPoints(ctx context.Context, userID uuid.UUID) (int, error)
	// TodayCompletions returns the user's most recent completions today with
	// their chants preloaded (Home "chant program" card).
	TodayCompletions(ctx context.Context, userID uuid.UUID, limit int) ([]models.ChantCompletion, map[uuid.UUID]models.Chant, error)
}

type chantRepository struct {
	db *gorm.DB
}

func NewChantRepository(db *gorm.DB) ChantRepository {
	return &chantRepository{db: db}
}

func (r *chantRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Chant, error) {
	var chant models.Chant
	err := r.db.WithContext(ctx).Preload("Song").First(&chant, "id = ? AND is_active = ?", id, true).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Chant not found", nil)
		}
		return nil, errors.NewInternal("Failed to find chant", err)
	}
	return &chant, nil
}

func (r *chantRepository) FindByMatch(ctx context.Context, matchID uuid.UUID, search string) ([]models.Chant, error) {
	q := r.db.WithContext(ctx).Preload("Song").
		Where("match_id = ? AND is_active = ?", matchID, true)
	if s := strings.TrimSpace(search); s != "" {
		q = q.Where("title ILIKE ?", "%"+s+"%")
	}

	var chants []models.Chant
	if err := q.Order("scheduled_at ASC").Find(&chants).Error; err != nil {
		return nil, errors.NewInternal("Failed to list chants", err)
	}
	return chants, nil
}

func (r *chantRepository) CompletedChantIDs(ctx context.Context, userID uuid.UUID, chantIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	done := make(map[uuid.UUID]bool, len(chantIDs))
	if len(chantIDs) == 0 {
		return done, nil
	}

	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&models.ChantCompletion{}).
		Where("user_id = ? AND chant_id IN ?", userID, chantIDs).
		Pluck("chant_id", &ids).Error; err != nil {
		return nil, errors.NewInternal("Failed to load completions", err)
	}
	for _, id := range ids {
		done[id] = true
	}
	return done, nil
}

func (r *chantRepository) TodayPoints(ctx context.Context, userID uuid.UUID) (int, error) {
	var total int64
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	err := r.db.WithContext(ctx).Model(&models.ChantCompletion{}).
		Where("user_id = ? AND created_at >= ?", userID, startOfDay).
		Select("COALESCE(SUM(points_earned), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, errors.NewInternal("Failed to sum today's points", err)
	}
	return int(total), nil
}

func (r *chantRepository) TodayCompletions(ctx context.Context, userID uuid.UUID, limit int) ([]models.ChantCompletion, map[uuid.UUID]models.Chant, error) {
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)

	var completions []models.ChantCompletion
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND created_at >= ?", userID, startOfDay).
		Order("created_at DESC").
		Limit(limit).
		Find(&completions).Error; err != nil {
		return nil, nil, errors.NewInternal("Failed to load completions", err)
	}

	chantByID := make(map[uuid.UUID]models.Chant, len(completions))
	if len(completions) > 0 {
		ids := make([]uuid.UUID, len(completions))
		for i, c := range completions {
			ids[i] = c.ChantID
		}
		var chants []models.Chant
		if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&chants).Error; err != nil {
			return nil, nil, errors.NewInternal("Failed to load chants", err)
		}
		for _, c := range chants {
			chantByID[c.ID] = c
		}
	}
	return completions, chantByID, nil
}
