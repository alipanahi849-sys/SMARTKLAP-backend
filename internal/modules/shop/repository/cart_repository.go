package repository

import (
	"context"

	"clap/internal/modules/shop/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CartRepository interface {
	ItemsByUser(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error)
	// CountByUser sums cart quantities — the "cart_count" badge.
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
	// Upsert adds an item or increments the quantity of the matching line.
	Upsert(ctx context.Context, item *models.CartItem) error
	UpdateQuantity(ctx context.Context, itemID, userID uuid.UUID, quantity int) (*models.CartItem, error)
	Delete(ctx context.Context, itemID, userID uuid.UUID) error
}

type cartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) CartRepository {
	return &cartRepository{db: db}
}

func (r *cartRepository) ItemsByUser(ctx context.Context, userID uuid.UUID) ([]models.CartItem, error) {
	var items []models.CartItem
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, errors.NewInternal("Failed to load cart", err)
	}
	return items, nil
}

func (r *cartRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.CartItem{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&total).Error; err != nil {
		return 0, errors.NewInternal("Failed to count cart items", err)
	}
	return int(total), nil
}

func (r *cartRepository) Upsert(ctx context.Context, item *models.CartItem) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"}, {Name: "product_type"}, {Name: "product_id"}, {Name: "size"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quantity":   gorm.Expr("cart_items.quantity + EXCLUDED.quantity"),
			"updated_at": gorm.Expr("NOW()"),
		}),
	}).Create(item).Error
	if err != nil {
		return errors.NewInternal("Failed to add item to cart", err)
	}
	return nil
}

func (r *cartRepository) UpdateQuantity(ctx context.Context, itemID, userID uuid.UUID, quantity int) (*models.CartItem, error) {
	res := r.db.WithContext(ctx).Model(&models.CartItem{}).
		Where("id = ? AND user_id = ?", itemID, userID).
		Update("quantity", quantity)
	if res.Error != nil {
		return nil, errors.NewInternal("Failed to update cart item", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, errors.NewNotFound("Cart item not found", nil)
	}

	var item models.CartItem
	if err := r.db.WithContext(ctx).First(&item, "id = ?", itemID).Error; err != nil {
		return nil, errors.NewInternal("Failed to reload cart item", err)
	}
	return &item, nil
}

func (r *cartRepository) Delete(ctx context.Context, itemID, userID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", itemID, userID).
		Delete(&models.CartItem{})
	if res.Error != nil {
		return errors.NewInternal("Failed to remove cart item", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NewNotFound("Cart item not found", nil)
	}
	return nil
}
