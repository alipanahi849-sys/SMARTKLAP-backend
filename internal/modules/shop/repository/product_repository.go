package repository

import (
	"context"
	"strings"

	"clap/internal/modules/shop/models"
	"clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"gorm.io/gorm"
)

// ProductFilters mirrors the list query params at the repository layer.
type ProductFilters struct {
	Search   string
	Category string
}

type ProductRepository interface {
	List(ctx context.Context, page, limit int, filters ProductFilters) ([]models.Product, int64, error)
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) List(ctx context.Context, page, limit int, filters ProductFilters) ([]models.Product, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Product{}).Where("is_active = ?", true)

	if cat := strings.TrimSpace(filters.Category); cat != "" {
		q = q.Where("category = ?", cat)
	}

	if s := strings.TrimSpace(filters.Search); s != "" {
		pattern := "%" + s + "%"
		q = q.Where(
			"name ILIKE ? OR subname ILIKE ? OR description ILIKE ?",
			pattern, pattern, pattern,
		)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to count products", err)
	}

	var products []models.Product
	if err := q.Order("created_at DESC").
		Offset(utils.GetOffset(page, limit)).
		Limit(limit).
		Find(&products).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to load products", err)
	}

	return products, total, nil
}
