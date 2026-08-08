package repository

import (
	"context"
	"time"

	"clap/internal/modules/shop/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CartCursorAnchor is the list position after which the next page is fetched.
type CartCursorAnchor struct {
	UpdatedAt time.Time
	ID        uuid.UUID
}

type CartRepository interface {
	FindLine(ctx context.Context, userID, productID uuid.UUID, size string) (*models.CartItem, error)
	FindLineByID(ctx context.Context, userID, id uuid.UUID) (*UserCartLine, error)
	ListUserLines(ctx context.Context, userID uuid.UUID) ([]UserCartLine, error)
	ListUserLinesAfter(ctx context.Context, userID uuid.UUID, limit int, after *CartCursorAnchor) ([]UserCartLine, error)
	SumSubtotalCents(ctx context.Context, userID uuid.UUID) (int64, error)
	SumSubtotalPoints(ctx context.Context, userID uuid.UUID) (int, error)
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
	ClearUserCart(ctx context.Context, userID uuid.UUID) error
	Create(ctx context.Context, item *models.CartItem) error
	UpdateQuantity(ctx context.Context, id uuid.UUID, quantity int) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountTotalQuantity(ctx context.Context, userID uuid.UUID) (int, error)
}

// UserCartLine is a cart row joined with its product for basket display.
type UserCartLine struct {
	models.CartItem
	ImageKey    string
	Name        string
	Subname     string
	Description string
	PriceCents  int64
	PricePoints int
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

func (r *cartRepository) userLinesQueryCtx(ctx context.Context, userID uuid.UUID) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("cart_items").
		Select("cart_items.*, products.image_key, products.name, products.subname, products.description, products.price_cents, products.price_points").
		Joins("INNER JOIN products ON products.id = cart_items.product_id AND products.deleted_at IS NULL AND products.is_active = ?", true).
		Where("cart_items.user_id = ?", userID)
}

func (r *cartRepository) FindLineByID(ctx context.Context, userID, id uuid.UUID) (*UserCartLine, error) {
	var lines []UserCartLine
	err := r.userLinesQueryCtx(ctx, userID).
		Where("cart_items.id = ?", id).
		Limit(1).
		Scan(&lines).Error
	if err != nil {
		return nil, errors.NewInternal("Failed to load cart item", err)
	}
	if len(lines) == 0 {
		return nil, errors.NewNotFound("Cart item not found", nil)
	}
	return &lines[0], nil
}

func (r *cartRepository) ListUserLines(ctx context.Context, userID uuid.UUID) ([]UserCartLine, error) {
	var lines []UserCartLine
	err := r.userLinesQueryCtx(ctx, userID).
		Order("cart_items.updated_at DESC, cart_items.id DESC").
		Scan(&lines).Error
	if err != nil {
		return nil, errors.NewInternal("Failed to load cart", err)
	}
	return lines, nil
}

func (r *cartRepository) ListUserLinesAfter(ctx context.Context, userID uuid.UUID, limit int, after *CartCursorAnchor) ([]UserCartLine, error) {
	q := r.userLinesQueryCtx(ctx, userID)
	if after != nil {
		q = q.Where(
			"(cart_items.updated_at < ?) OR (cart_items.updated_at = ? AND cart_items.id < ?)",
			after.UpdatedAt, after.UpdatedAt, after.ID,
		)
	}

	var lines []UserCartLine
	if err := q.Order("cart_items.updated_at DESC, cart_items.id DESC").
		Limit(limit).
		Scan(&lines).Error; err != nil {
		return nil, errors.NewInternal("Failed to load cart", err)
	}
	return lines, nil
}

func (r *cartRepository) SumSubtotalCents(ctx context.Context, userID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Table("cart_items").
		Select("COALESCE(SUM(products.price_cents * cart_items.quantity), 0)").
		Joins("INNER JOIN products ON products.id = cart_items.product_id AND products.deleted_at IS NULL AND products.is_active = ?", true).
		Where("cart_items.user_id = ?", userID).
		Scan(&total).Error
	if err != nil {
		return 0, errors.NewInternal("Failed to sum cart subtotal", err)
	}
	return total, nil
}

func (r *cartRepository) SumSubtotalPoints(ctx context.Context, userID uuid.UUID) (int, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Table("cart_items").
		Select("COALESCE(SUM(products.price_points * cart_items.quantity), 0)").
		Joins("INNER JOIN products ON products.id = cart_items.product_id AND products.deleted_at IS NULL AND products.is_active = ?", true).
		Where("cart_items.user_id = ?", userID).
		Scan(&total).Error
	if err != nil {
		return 0, errors.NewInternal("Failed to sum cart subtotal points", err)
	}
	return int(total), nil
}

func (r *cartRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	return r.ClearUserCart(ctx, userID)
}

func (r *cartRepository) ClearUserCart(ctx context.Context, userID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.CartItem{}).Error; err != nil {
		return errors.NewInternal("Failed to clear cart", err)
	}
	return nil
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
