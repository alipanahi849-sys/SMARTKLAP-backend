package repository

import (
	"context"
	"strings"

	"clap/internal/modules/shop/models"
	"clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SnackRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.Snack, error)
	// FindByIDs batch-loads snacks keyed by ID (cart/checkout rendering).
	FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]models.Snack, error)
	FindAll(ctx context.Context, search, category string, page, limit int) ([]models.Snack, int64, error)
	// FindPreview returns the first N active snacks (Home screen preview).
	FindPreview(ctx context.Context, limit int) ([]models.Snack, error)
}

type ProductRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.Product, error)
	FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]models.Product, error)
	FindAll(ctx context.Context, search, category string, page, limit int) ([]models.Product, int64, error)
	FindPreview(ctx context.Context, limit int) ([]models.Product, error)
}

type snackRepository struct {
	db *gorm.DB
}

func NewSnackRepository(db *gorm.DB) SnackRepository {
	return &snackRepository{db: db}
}

func (r *snackRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Snack, error) {
	var snack models.Snack
	err := r.db.WithContext(ctx).First(&snack, "id = ? AND is_active = ?", id, true).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Snack not found", nil)
		}
		return nil, errors.NewInternal("Failed to find snack", err)
	}
	return &snack, nil
}

func (r *snackRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]models.Snack, error) {
	result := make(map[uuid.UUID]models.Snack, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var snacks []models.Snack
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&snacks).Error; err != nil {
		return nil, errors.NewInternal("Failed to load snacks", err)
	}
	for _, s := range snacks {
		result[s.ID] = s
	}
	return result, nil
}

func (r *snackRepository) FindAll(ctx context.Context, search, category string, page, limit int) ([]models.Snack, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Snack{}).Where("is_active = ?", true)
	if s := strings.TrimSpace(search); s != "" {
		q = q.Where("name ILIKE ?", "%"+s+"%")
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to count snacks", err)
	}

	var snacks []models.Snack
	if err := q.Order("created_at ASC").
		Offset(utils.GetOffset(page, limit)).
		Limit(limit).
		Find(&snacks).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to list snacks", err)
	}
	return snacks, total, nil
}

func (r *snackRepository) FindPreview(ctx context.Context, limit int) ([]models.Snack, error) {
	var snacks []models.Snack
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("created_at ASC").
		Limit(limit).
		Find(&snacks).Error; err != nil {
		return nil, errors.NewInternal("Failed to load snacks preview", err)
	}
	return snacks, nil
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	var product models.Product
	err := r.db.WithContext(ctx).First(&product, "id = ? AND is_active = ?", id, true).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Product not found", nil)
		}
		return nil, errors.NewInternal("Failed to find product", err)
	}
	return &product, nil
}

func (r *productRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]models.Product, error) {
	result := make(map[uuid.UUID]models.Product, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var products []models.Product
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, errors.NewInternal("Failed to load products", err)
	}
	for _, p := range products {
		result[p.ID] = p
	}
	return result, nil
}

func (r *productRepository) FindAll(ctx context.Context, search, category string, page, limit int) ([]models.Product, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Product{}).Where("is_active = ?", true)
	if s := strings.TrimSpace(search); s != "" {
		q = q.Where("name ILIKE ?", "%"+s+"%")
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to count products", err)
	}

	var products []models.Product
	if err := q.Order("created_at ASC").
		Offset(utils.GetOffset(page, limit)).
		Limit(limit).
		Find(&products).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to list products", err)
	}
	return products, total, nil
}

func (r *productRepository) FindPreview(ctx context.Context, limit int) ([]models.Product, error) {
	var products []models.Product
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("created_at ASC").
		Limit(limit).
		Find(&products).Error; err != nil {
		return nil, errors.NewInternal("Failed to load products preview", err)
	}
	return products, nil
}
