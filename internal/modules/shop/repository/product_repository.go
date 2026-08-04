package repository

import (
	"context"
	"strings"
	"time"

	"clap/internal/modules/shop/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductFilters mirrors the list query params at the repository layer.
type ProductFilters struct {
	Search      string
	Category    string
	ProductType string
}

// CursorAnchor is the list position after which the next page is fetched.
type CursorAnchor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type ProductRepository interface {
	List(ctx context.Context, limit int, filters ProductFilters, after *CursorAnchor) ([]models.Product, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Product, error)
	FindByIDAdmin(ctx context.Context, id uuid.UUID) (*models.Product, error)
	Create(ctx context.Context, product *models.Product) error
	Update(ctx context.Context, product *models.Product) error
	UpdateImageKey(ctx context.Context, id uuid.UUID, imageKey string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) List(ctx context.Context, limit int, filters ProductFilters, after *CursorAnchor) ([]models.Product, error) {
	q := r.db.WithContext(ctx).Model(&models.Product{}).Where("is_active = ?", true)

	if cat := strings.TrimSpace(filters.Category); cat != "" {
		q = q.Where("category = ?", cat)
	}

	if pt := strings.TrimSpace(filters.ProductType); pt != "" {
		q = q.Where("product_type = ?", pt)
	}

	if s := strings.TrimSpace(filters.Search); s != "" {
		pattern := "%" + s + "%"
		q = q.Where(
			"name ILIKE ? OR subname ILIKE ? OR description ILIKE ?",
			pattern, pattern, pattern,
		)
	}

	if after != nil {
		q = q.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			after.CreatedAt, after.CreatedAt, after.ID,
		)
	}

	var products []models.Product
	if err := q.Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&products).Error; err != nil {
		return nil, errors.NewInternal("Failed to load products", err)
	}

	return products, nil
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

func (r *productRepository) FindByIDAdmin(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	var product models.Product
	err := r.db.WithContext(ctx).First(&product, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Product not found", nil)
		}
		return nil, errors.NewInternal("Failed to find product", err)
	}
	return &product, nil
}

func (r *productRepository) UpdateImageKey(ctx context.Context, id uuid.UUID, imageKey string) error {
	res := r.db.WithContext(ctx).Model(&models.Product{}).
		Where("id = ?", id).
		Update("image_key", imageKey)
	if res.Error != nil {
		return errors.NewInternal("Failed to update product image", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NewNotFound("Product not found", nil)
	}
	return nil
}

func (r *productRepository) Create(ctx context.Context, product *models.Product) error {
	if err := r.db.WithContext(ctx).Create(product).Error; err != nil {
		return errors.NewInternal("Failed to create product", err)
	}
	return nil
}

func (r *productRepository) Update(ctx context.Context, product *models.Product) error {
	if err := r.db.WithContext(ctx).Save(product).Error; err != nil {
		return errors.NewInternal("Failed to update product", err)
	}
	return nil
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Product{}, id).Error; err != nil {
		return errors.NewInternal("Failed to delete product", err)
	}
	return nil
}
