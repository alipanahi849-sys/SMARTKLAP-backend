package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Product type discriminators for the shared cart (contract note §4: one cart
// across Snacks and Store).
const (
	ProductTypeSnack = "snack"
	ProductTypeMerch = "merch"
)

// Order lifecycle.
const (
	OrderStatusPendingPayment = "pending_payment"
	OrderStatusPaid           = "paid"
	OrderStatusCancelled      = "cancelled"
)

// Snack is a food/drink item (Mobile API Contract §6).
type Snack struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	PriceCents  int64          `gorm:"not null;default:0" json:"price_cents"`
	PointsPrice int            `gorm:"not null;default:0" json:"points_price"`
	Category    string         `gorm:"type:varchar(30);not null;default:'snacks'" json:"category"`
	ImageURL    string         `gorm:"type:varchar(500)" json:"image_url"`
	IsActive    bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy   *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy   *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (Snack) TableName() string {
	return "snacks"
}

// Product is a merch item (Mobile API Contract §7). Sizes holds a JSON array
// of labels, e.g. ["M","L","XL","XXL"].
type Product struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	SellerName  string         `gorm:"type:varchar(255)" json:"seller_name"`
	Description string         `gorm:"type:text" json:"description"`
	PriceCents  int64          `gorm:"not null;default:0" json:"price_cents"`
	PointsPrice int            `gorm:"not null;default:0" json:"points_price"`
	Category    string         `gorm:"type:varchar(30);not null;default:'t-shirts'" json:"category"`
	ImageURL    string         `gorm:"type:varchar(500)" json:"image_url"`
	Sizes       string         `gorm:"type:jsonb;not null;default:'[]'" json:"sizes"`
	IsActive    bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy   *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy   *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (Product) TableName() string {
	return "products"
}

// CartItem is one line in the user's single shared cart.
type CartItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	ProductType string    `gorm:"type:varchar(10);not null" json:"product_type"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Quantity    int       `gorm:"not null;default:1" json:"quantity"`
	Size        string    `gorm:"type:varchar(10);not null;default:''" json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (CartItem) TableName() string {
	return "cart_items"
}

// Order is a checkout of the cart (contract §6.4).
type Order struct {
	ID             uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID   `gorm:"type:uuid;not null" json:"user_id"`
	DeliveryMethod string      `gorm:"type:varchar(10);not null" json:"delivery_method"`
	SeatNumber     string      `gorm:"type:varchar(20);not null;default:''" json:"seat_number"`
	SubtotalCents  int64       `gorm:"not null;default:0" json:"subtotal_cents"`
	ShippingCents  int64       `gorm:"not null;default:0" json:"shipping_cents"`
	TotalCents     int64       `gorm:"not null;default:0" json:"total_cents"`
	Status         string      `gorm:"type:varchar(20);not null;default:'pending_payment'" json:"status"`
	PaymentMethod  string      `gorm:"type:varchar(30);not null;default:''" json:"payment_method"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	Items          []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

func (Order) TableName() string {
	return "orders"
}

// OrderItem is an immutable snapshot of a purchased line.
type OrderItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrderID     uuid.UUID `gorm:"type:uuid;not null" json:"order_id"`
	ProductType string    `gorm:"type:varchar(10);not null" json:"product_type"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	PriceCents  int64     `gorm:"not null;default:0" json:"price_cents"`
	Quantity    int       `gorm:"not null;default:1" json:"quantity"`
	Size        string    `gorm:"type:varchar(10);not null;default:''" json:"size"`
	ImageURL    string    `gorm:"type:varchar(500)" json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
