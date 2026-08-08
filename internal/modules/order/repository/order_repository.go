package repository

import (
	"context"
	"strings"
	"time"

	"clap/internal/modules/order/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderCursorAnchor is the list position after which the next page is fetched.
type OrderCursorAnchor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type OrderRepository interface {
	CreateWithItems(ctx context.Context, order *models.Order, items []models.OrderItem) error
	ListForUserAfter(ctx context.Context, userID uuid.UUID, limit int, after *OrderCursorAnchor) ([]models.Order, error)
	FindByIDForUser(ctx context.Context, userID, orderID uuid.UUID) (*models.Order, error)
	FindByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error)
	FindByStripePaymentIntentID(ctx context.Context, intentID string) (*models.Order, error)
	MarkPaid(ctx context.Context, orderID uuid.UUID, paidAt time.Time, paymentMethod string) error
	UpdateStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, intentID string) error
	RecordStripeEvent(ctx context.Context, eventID, eventType string, orderID *uuid.UUID) (bool, error)
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) CreateWithItems(ctx context.Context, order *models.Order, items []models.OrderItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return errors.NewInternal("Failed to create order", err)
		}
		if len(items) == 0 {
			return nil
		}
		for i := range items {
			items[i].OrderID = order.ID
		}
		if err := tx.Create(&items).Error; err != nil {
			return errors.NewInternal("Failed to create order items", err)
		}
		order.Items = items
		return nil
	})
}

func (r *orderRepository) ListForUserAfter(ctx context.Context, userID uuid.UUID, limit int, after *OrderCursorAnchor) ([]models.Order, error) {
	q := r.db.WithContext(ctx).
		Preload("Items").
		Where("user_id = ?", userID)

	if after != nil {
		q = q.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			after.CreatedAt, after.CreatedAt, after.ID,
		)
	}

	var orders []models.Order
	if err := q.Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&orders).Error; err != nil {
		return nil, errors.NewInternal("Failed to load orders", err)
	}
	return orders, nil
}

func (r *orderRepository) FindByIDForUser(ctx context.Context, userID, orderID uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("id = ? AND user_id = ?", orderID, userID).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Order not found", nil)
		}
		return nil, errors.NewInternal("Failed to load order", err)
	}
	return &order, nil
}

func (r *orderRepository) FindByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		First(&order, "id = ?", orderID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Order not found", nil)
		}
		return nil, errors.NewInternal("Failed to load order", err)
	}
	return &order, nil
}

func (r *orderRepository) FindByStripePaymentIntentID(ctx context.Context, intentID string) (*models.Order, error) {
	var order models.Order
	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("stripe_payment_intent_id = ?", intentID).
		First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Order not found", nil)
		}
		return nil, errors.NewInternal("Failed to load order", err)
	}
	return &order, nil
}

func (r *orderRepository) UpdateStripePaymentIntentID(ctx context.Context, orderID uuid.UUID, intentID string) error {
	res := r.db.WithContext(ctx).Model(&models.Order{}).
		Where("id = ? AND status = ?", orderID, models.OrderStatusPendingPayment).
		Update("stripe_payment_intent_id", intentID)
	if res.Error != nil {
		return errors.NewInternal("Failed to update payment intent", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NewUnprocessable("Order is not pending payment", nil)
	}
	return nil
}

func (r *orderRepository) RecordStripeEvent(ctx context.Context, eventID, eventType string, orderID *uuid.UUID) (bool, error) {
	event := models.StripePaymentEvent{
		EventID:   eventID,
		EventType: eventType,
		OrderID:   orderID,
	}
	err := r.db.WithContext(ctx).Create(&event).Error
	if err != nil {
		if isUniqueViolation(err) {
			return false, nil
		}
		return false, errors.NewInternal("Failed to record stripe event", err)
	}
	return true, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "UNIQUE constraint failed"))
}

func (r *orderRepository) MarkPaid(ctx context.Context, orderID uuid.UUID, paidAt time.Time, paymentMethod string) error {
	updates := map[string]interface{}{
		"status":  models.OrderStatusPaid,
		"paid_at": paidAt,
	}
	if paymentMethod != "" {
		updates["payment_method"] = paymentMethod
	}

	res := r.db.WithContext(ctx).Model(&models.Order{}).
		Where("id = ? AND status = ?", orderID, models.OrderStatusPendingPayment).
		Updates(updates)
	if res.Error != nil {
		return errors.NewInternal("Failed to mark order paid", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NewUnprocessable("Order is not pending payment", nil)
	}
	return nil
}
