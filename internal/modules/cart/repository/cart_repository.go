package repository

import (
	"context"

	"clap/internal/modules/cart/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartRepository interface {
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error)
	FindByID(ctx context.Context, itemID, userID uuid.UUID) (*models.CartItem, error)
	FindByUserProductSize(ctx context.Context, userID, productID uuid.UUID, size string) (*models.CartItem, error)
	Create(ctx context.Context, item *models.CartItem) error
	Update(ctx context.Context, item *models.CartItem) error
	Delete(ctx context.Context, itemID, userID uuid.UUID) error
	DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}

type cartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) CartRepository {
	return &cartRepository{db: db}
}

func (r *cartRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error) {
	var items []models.CartItem
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, errors.NewInternal("Failed to load cart items", err)
	}
	return items, nil
}

func (r *cartRepository) FindByID(ctx context.Context, itemID, userID uuid.UUID) (*models.CartItem, error) {
	var item models.CartItem
	err := r.db.WithContext(ctx).First(&item, "id = ? AND user_id = ?", itemID, userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Cart item not found", nil)
		}
		return nil, errors.NewInternal("Failed to find cart item", err)
	}
	return &item, nil
}

func (r *cartRepository) FindByUserProductSize(ctx context.Context, userID, productID uuid.UUID, size string) (*models.CartItem, error) {
	var item models.CartItem
	err := r.db.WithContext(ctx).
		First(&item, "user_id = ? AND product_id = ? AND size = ?", userID, productID, size).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.NewInternal("Failed to find cart item", err)
	}
	return &item, nil
}

func (r *cartRepository) Create(ctx context.Context, item *models.CartItem) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return errors.NewInternal("Failed to add cart item", err)
	}
	return nil
}

func (r *cartRepository) Update(ctx context.Context, item *models.CartItem) error {
	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		return errors.NewInternal("Failed to update cart item", err)
	}
	return nil
}

func (r *cartRepository) Delete(ctx context.Context, itemID, userID uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&models.CartItem{}, "id = ? AND user_id = ?", itemID, userID)
	if res.Error != nil {
		return errors.NewInternal("Failed to remove cart item", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NewNotFound("Cart item not found", nil)
	}
	return nil
}

func (r *cartRepository) DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.CartItem{}).Error; err != nil {
		return errors.NewInternal("Failed to clear cart", err)
	}
	return nil
}

func (r *cartRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.CartItem{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&total).Error; err != nil {
		return 0, errors.NewInternal("Failed to count cart items", err)
	}
	return int(total), nil
}
