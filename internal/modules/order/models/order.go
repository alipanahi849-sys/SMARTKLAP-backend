package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	OrderStatusPendingPayment = "pending_payment"
	OrderStatusPaid           = "paid"
	OrderStatusCancelled      = "cancelled"

	DeliveryMethodSeat   = "seat"
	DeliveryMethodPickup = "pickup"

	PaymentMethodPoints = "points"
	PaymentMethodCard   = "card"

	PickupDiscountCents  int64 = 50
	PickupDiscountPoints int   = 50

	// PendingPaymentTTL is how long an unpaid order stays payable.
	PendingPaymentTTL = 10 * time.Minute
)

type Order struct {
	ID                    uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID                uuid.UUID   `gorm:"type:uuid;not null;index" json:"user_id"`
	Status                string      `gorm:"type:varchar(30);not null;default:'pending_payment'" json:"status"`
	DeliveryMethod        string      `gorm:"type:varchar(20);not null" json:"delivery_method"`
	SeatNumber            *string     `gorm:"type:varchar(50)" json:"seat_number,omitempty"`
	SubtotalCents         int64       `gorm:"not null" json:"subtotal_cents"`
	ShippingCents         int64       `gorm:"not null;default:0" json:"shipping_cents"`
	TotalCents            int64       `gorm:"not null" json:"total_cents"`
	SubtotalPoints        int         `gorm:"not null;default:0" json:"subtotal_points"`
	ShippingPoints        int         `gorm:"not null;default:0" json:"shipping_points"`
	TotalPoints           int         `gorm:"not null;default:0" json:"total_points"`
	PaymentMethod         *string     `gorm:"type:varchar(20)" json:"payment_method,omitempty"`
	StripePaymentIntentID *string     `gorm:"type:varchar(255)" json:"stripe_payment_intent_id,omitempty"`
	PaidAt                *time.Time  `json:"paid_at,omitempty"`
	CreatedAt             time.Time   `json:"created_at"`
	UpdatedAt             time.Time   `json:"updated_at"`
	Items                 []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

func (Order) TableName() string { return "orders" }

// PendingExpiresAt is when a pending_payment order becomes cancelled.
func (o *Order) PendingExpiresAt() time.Time {
	if o == nil || o.CreatedAt.IsZero() {
		return time.Time{}
	}
	return o.CreatedAt.UTC().Add(PendingPaymentTTL)
}

// IsPendingExpired reports whether an unpaid order has passed its payment window.
func (o *Order) IsPendingExpired(now time.Time) bool {
	if o == nil || o.Status != OrderStatusPendingPayment || o.CreatedAt.IsZero() {
		return false
	}
	return !now.UTC().Before(o.PendingExpiresAt())
}

func (o *Order) BeforeCreate(_ *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	now := time.Now().UTC()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = now
	}
	if o.UpdatedAt.IsZero() {
		o.UpdatedAt = now
	}
	return nil
}

type OrderItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrderID     uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	ProductType string    `gorm:"type:varchar(20);not null" json:"product_type"`
	Size        string    `gorm:"type:varchar(50);not null;default:''" json:"size"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Subname     *string   `gorm:"type:varchar(255)" json:"subname,omitempty"`
	PriceCents  int64     `gorm:"not null" json:"price_cents"`
	PricePoints int       `gorm:"not null;default:0" json:"price_points"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
}

func (OrderItem) TableName() string { return "order_items" }

func (i *OrderItem) BeforeCreate(_ *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	if i.CreatedAt.IsZero() {
		i.CreatedAt = time.Now().UTC()
	}
	return nil
}
