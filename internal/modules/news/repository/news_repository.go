package repository

import (
	"context"

	"clap/internal/modules/news/models"
	"clap/internal/shared/errors"

	"gorm.io/gorm"
)

type NewsRepository interface {
	FindAll(ctx context.Context, page, limit int) ([]models.News, int64, error)
	// FindPreview returns the newest active articles for the Home preview.
	FindPreview(ctx context.Context, limit int) ([]models.News, error)
}

type newsRepository struct {
	db *gorm.DB
}

func NewNewsRepository(db *gorm.DB) NewsRepository {
	return &newsRepository{db: db}
}

func (r *newsRepository) FindAll(ctx context.Context, page, limit int) ([]models.News, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.News{}).Where("is_active = ?", true)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to count news", err)
	}

	var items []models.News
	if err := query.
		Order("published_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to list news", err)
	}
	return items, total, nil
}

func (r *newsRepository) FindPreview(ctx context.Context, limit int) ([]models.News, error) {
	var items []models.News
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("published_at DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, errors.NewInternal("Failed to load news preview", err)
	}
	return items, nil
}
