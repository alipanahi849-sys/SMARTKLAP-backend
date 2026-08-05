package models

import (
	"time"

	"github.com/google/uuid"
)

// CartItem is one line in a user's shopping cart.
type CartItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	ProductType string    `gorm:"type:varchar(20);not null" json:"product_type"`
	Size        string    `gorm:"type:varchar(50);not null;default:''" json:"size"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (CartItem) TableName() string {
	return "cart_items"
}
