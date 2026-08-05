package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPendingPayment = "pending_payment"
	StatusPaid           = "paid"
	StatusCancelled      = "cancelled"
)

const (
	DeliverySeat   = "seat"
	DeliveryPickup = "pickup"
)

type Order struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Status         string    `gorm:"type:varchar(30);not null;default:pending_payment" json:"status"`
	DeliveryMethod string    `gorm:"type:varchar(20);not null" json:"delivery_method"`
	SeatNumber     string    `gorm:"type:varchar(50);not null;default:''" json:"seat_number"`
	SubtotalCents  int64     `gorm:"not null;default:0" json:"subtotal_cents"`
	ShippingCents  int64     `gorm:"not null;default:0" json:"shipping_cents"`
	TotalCents     int64     `gorm:"not null;default:0" json:"total_cents"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrderID     uuid.UUID `gorm:"type:uuid;not null" json:"order_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	ProductType string    `gorm:"type:varchar(20);not null" json:"product_type"`
	Name        string    `gorm:"type:varchar(200);not null" json:"name"`
	Description string    `gorm:"type:text;not null;default:''" json:"description"`
	PriceCents  int64     `gorm:"not null" json:"price_cents"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	Size        string    `gorm:"type:varchar(50);not null;default:''" json:"size"`
	ImageKey    string    `gorm:"type:varchar(500);not null;default:''" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
