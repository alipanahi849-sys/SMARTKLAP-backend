package repository

import (
	"context"
	"strings"
	"time"

	authmodels "clap/internal/modules/auth/models"
	"clap/internal/modules/chant/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ChantCursorAnchor is the list position after which the next page is fetched.
type ChantCursorAnchor struct {
	ScheduledAt time.Time
	ID          uuid.UUID
}

type ChantRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.Chant, error)
	// FindByMatchAfter returns active chants for a match ordered by schedule time.
	// Pass after=nil for the first page.
	FindByMatchAfter(ctx context.Context, matchID uuid.UUID, search string, limit int, after *ChantCursorAnchor) ([]models.Chant, error)
	// HasIncompleteAtOrBefore reports whether the user has not completed any chant
	// at or before the given anchor in the match list order.
	HasIncompleteAtOrBefore(ctx context.Context, userID, matchID uuid.UUID, search string, anchor *ChantCursorAnchor) (bool, error)
	// CompletedChantIDs returns which of the given chants the user completed.
	CompletedChantIDs(ctx context.Context, userID uuid.UUID, chantIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	// TodayPoints sums the user's chant points earned since local midnight (UTC).
	TodayPoints(ctx context.Context, userID uuid.UUID) (int, error)
	// TodayCompletions returns the user's most recent completions today with
	// their chants preloaded (Home "chant program" card).
	TodayCompletions(ctx context.Context, userID uuid.UUID, limit int) ([]models.ChantCompletion, map[uuid.UUID]models.Chant, error)
	// Complete records a completion and atomically credits the user's points.
	// Returns created=false when the chant was already completed (idempotent).
	Complete(ctx context.Context, chantID, userID uuid.UUID, points int) (totalPoints int, created bool, err error)
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

func (r *chantRepository) FindByMatchAfter(ctx context.Context, matchID uuid.UUID, search string, limit int, after *ChantCursorAnchor) ([]models.Chant, error) {
	q := r.db.WithContext(ctx).Preload("Song").
		Where("match_id = ? AND is_active = ?", matchID, true)
	if s := strings.TrimSpace(search); s != "" {
		q = q.Where("title ILIKE ?", "%"+s+"%")
	}
	if after != nil {
		q = q.Where(
			"(scheduled_at > ?) OR (scheduled_at = ? AND id > ?)",
			after.ScheduledAt, after.ScheduledAt, after.ID,
		)
	}

	var chants []models.Chant
	if err := q.Order("scheduled_at ASC, id ASC").
		Limit(limit).
		Find(&chants).Error; err != nil {
		return nil, errors.NewInternal("Failed to list chants", err)
	}
	return chants, nil
}

func (r *chantRepository) HasIncompleteAtOrBefore(ctx context.Context, userID, matchID uuid.UUID, search string, anchor *ChantCursorAnchor) (bool, error) {
	if anchor == nil {
		return false, nil
	}

	q := r.db.WithContext(ctx).Table("chants c").
		Joins("LEFT JOIN chant_completions cc ON cc.chant_id = c.id AND cc.user_id = ?", userID).
		Where("c.match_id = ? AND c.is_active = ? AND cc.id IS NULL", matchID, true).
		Where(
			"(c.scheduled_at < ?) OR (c.scheduled_at = ? AND c.id <= ?)",
			anchor.ScheduledAt, anchor.ScheduledAt, anchor.ID,
		)
	if s := strings.TrimSpace(search); s != "" {
		q = q.Where("c.title ILIKE ?", "%"+s+"%")
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, errors.NewInternal("Failed to check chant progress", err)
	}
	return count > 0, nil
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

func (r *chantRepository) Complete(ctx context.Context, chantID, userID uuid.UUID, points int) (int, bool, error) {
	var totalPoints int
	created := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		completion := models.ChantCompletion{
			ChantID:      chantID,
			UserID:       userID,
			PointsEarned: points,
		}
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chant_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).Create(&completion)
		if res.Error != nil {
			return errors.NewInternal("Failed to record completion", res.Error)
		}
		if res.RowsAffected == 0 {
			if err := tx.Model(&authmodels.User{}).
				Where("id = ?", userID).
				Pluck("points", &totalPoints).Error; err != nil {
				return errors.NewInternal("Failed to read points balance", err)
			}
			return nil
		}

		created = true
		if err := tx.Model(&authmodels.User{}).
			Where("id = ?", userID).
			UpdateColumn("points", gorm.Expr("points + ?", points)).Error; err != nil {
			return errors.NewInternal("Failed to award points", err)
		}

		if err := tx.Model(&authmodels.User{}).
			Where("id = ?", userID).
			Pluck("points", &totalPoints).Error; err != nil {
			return errors.NewInternal("Failed to read points balance", err)
		}
		return nil
	})
	if err != nil {
		return 0, false, err
	}
	return totalPoints, created, nil
}
