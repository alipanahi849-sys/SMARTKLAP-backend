package repository

import (
	"context"

	"clap/internal/modules/shop/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartRepository interface {
	FindLine(ctx context.Context, userID, productID uuid.UUID, size string) (*models.CartItem, error)
	Create(ctx context.Context, item *models.CartItem) error
	UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountTotalQuantity(ctx context.Context, userID uuid.UUID) (int, error)
}

type cartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) CartRepository {
	return &cartRepository{db: db}
}

func (r *cartRepository) FindLine(ctx context.Context, userID, productID uuid.UUID, size string) (*models.CartItem, error) {
	var item models.CartItem
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND product_id = ? AND size = ?", userID, productID, size).
		First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.NewInternal("Failed to load cart item", err)
	}
	return &item, nil
}

func (r *cartRepository) Create(ctx context.Context, item *models.CartItem) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return errors.NewInternal("Failed to add cart item", err)
	}
	return nil
}

func (r *cartRepository) UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) error {
	res := r.db.WithContext(ctx).Model(&models.CartItem{}).
		Where("id = ?", id).
		Update("quantity", quantity)
	if res.Error != nil {
		return errors.NewInternal("Failed to update cart item", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NewNotFound("Cart item not found", nil)
	}
	return nil
}

func (r *cartRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&models.CartItem{}, id)
	if res.Error != nil {
		return errors.NewInternal("Failed to remove cart item", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NewNotFound("Cart item not found", nil)
	}
	return nil
}

func (r *cartRepository) CountTotalQuantity(ctx context.Context, userID uuid.UUID) (int, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&models.CartItem{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, errors.NewInternal("Failed to count cart items", err)
	}
	return int(total), nil
}
