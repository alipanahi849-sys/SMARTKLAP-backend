package repository

import (
	"context"

	"clap/internal/modules/shop/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepository interface {
	// CreateFromCart atomically persists the order + item snapshots and clears
	// the user's cart (checkout is a single transaction).
	CreateFromCart(ctx context.Context, order *models.Order, items []models.OrderItem) error
	FindByIDForUser(ctx context.Context, orderID, userID uuid.UUID) (*models.Order, error)
	// MarkPaid transitions pending_payment → paid; returns 409 on any other state.
	MarkPaid(ctx context.Context, orderID, userID uuid.UUID, paymentMethod string) error
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) CreateFromCart(ctx context.Context, order *models.Order, items []models.OrderItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return errors.NewInternal("Failed to create order", err)
		}
		for i := range items {
			items[i].OrderID = order.ID
		}
		if err := tx.Create(&items).Error; err != nil {
			return errors.NewInternal("Failed to create order items", err)
		}
		if err := tx.Where("user_id = ?", order.UserID).Delete(&models.CartItem{}).Error; err != nil {
			return errors.NewInternal("Failed to clear cart", err)
		}
		return nil
	})
}

func (r *orderRepository) FindByIDForUser(ctx context.Context, orderID, userID uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).Preload("Items").
		First(&order, "id = ? AND user_id = ?", orderID, userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Order not found", nil)
		}
		return nil, errors.NewInternal("Failed to find order", err)
	}
	return &order, nil
}

func (r *orderRepository) MarkPaid(ctx context.Context, orderID, userID uuid.UUID, paymentMethod string) error {
	res := r.db.WithContext(ctx).Model(&models.Order{}).
		Where("id = ? AND user_id = ? AND status = ?", orderID, userID, models.OrderStatusPendingPayment).
		Updates(map[string]interface{}{
			"status":         models.OrderStatusPaid,
			"payment_method": paymentMethod,
		})
	if res.Error != nil {
		return errors.NewInternal("Failed to update order", res.Error)
	}
	if res.RowsAffected == 0 {
		// Either the order doesn't exist for this user or it isn't payable.
		var exists int64
		r.db.WithContext(ctx).Model(&models.Order{}).
			Where("id = ? AND user_id = ?", orderID, userID).
			Count(&exists)
		if exists == 0 {
			return errors.NewNotFound("Order not found", nil)
		}
		return errors.NewConflict("Order is not awaiting payment", nil)
	}
	return nil
}
