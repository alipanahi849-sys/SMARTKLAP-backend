package repository

import (
	"context"

	"clap/internal/modules/order/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(ctx context.Context, order *models.Order, items []models.OrderItem) error
	FindByID(ctx context.Context, orderID, userID uuid.UUID) (*models.Order, error)
	ListItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.OrderItem, error)
	UpdateStatus(ctx context.Context, orderID uuid.UUID, status string) error
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, order *models.Order, items []models.OrderItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return errors.NewInternal("Failed to create order", err)
		}

		for i := range items {
			items[i].OrderID = order.ID
		}

		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return errors.NewInternal("Failed to create order items", err)
			}
		}

		return nil
	})
}

func (r *orderRepository) FindByID(ctx context.Context, orderID, userID uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).First(&order, "id = ? AND user_id = ?", orderID, userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Order not found", nil)
		}
		return nil, errors.NewInternal("Failed to find order", err)
	}
	return &order, nil
}

func (r *orderRepository) ListItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]models.OrderItem, error) {
	var items []models.OrderItem
	if err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		return nil, errors.NewInternal("Failed to load order items", err)
	}
	return items, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, orderID uuid.UUID, status string) error {
	res := r.db.WithContext(ctx).Model(&models.Order{}).
		Where("id = ?", orderID).
		Update("status", status)
	if res.Error != nil {
		return errors.NewInternal("Failed to update order status", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NewNotFound("Order not found", nil)
	}
	return nil
}
