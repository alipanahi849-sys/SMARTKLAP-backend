package repository

import (
	"context"
	"time"

	"clap/internal/modules/news/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NewsCursorAnchor struct {
	PublishedAt time.Time
	ID          uuid.UUID
}

type NewsRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.News, error)
	ListAfter(ctx context.Context, limit int, after *NewsCursorAnchor) ([]models.News, error)
}

type newsRepository struct {
	db *gorm.DB
}

func NewNewsRepository(db *gorm.DB) NewsRepository {
	return &newsRepository{db: db}
}

func (r *newsRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.News, error) {
	var item models.News
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("News article not found", nil)
		}
		return nil, errors.NewInternal("Failed to load news article", err)
	}
	return &item, nil
}

func (r *newsRepository) ListAfter(ctx context.Context, limit int, after *NewsCursorAnchor) ([]models.News, error) {
	q := r.db.WithContext(ctx).Model(&models.News{}).Where("is_active = ?", true)
	if after != nil {
		q = q.Where(
			"(published_at < ?) OR (published_at = ? AND id < ?)",
			after.PublishedAt, after.PublishedAt, after.ID,
		)
	}

	var items []models.News
	if err := q.Order("published_at DESC, id DESC").Limit(limit).Find(&items).Error; err != nil {
		return nil, errors.NewInternal("Failed to list news", err)
	}
	return items, nil
}
