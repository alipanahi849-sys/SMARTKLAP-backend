package dto

import (
	"time"

	"github.com/google/uuid"
)

type AddCartItemRequest struct {
	ProductType string `json:"product_type" binding:"required"`
	ProductID   string `json:"product_id" binding:"required"`
	Quantity    int    `json:"quantity" binding:"required,min=1"`
	Size        string `json:"size"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

type CartItemPreview struct {
	ID       uuid.UUID `json:"id"`
	ImageURL string    `json:"image_url"`
	Quantity int       `json:"quantity"`
}

type CartOrderGroup struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Date      string            `json:"date"`
	Items     []CartItemPreview `json:"items"`
	ExtraText string            `json:"extra_text,omitempty"`
}

type CartResponse struct {
	Orders []CartOrderGroup `json:"orders"`
	Items  []CartItemLine   `json:"items"`
}

type CartItemLine struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

type CartMutationResponse struct {
	CartCount int `json:"cart_count"`
}

type CartItemDetail struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	ProductType string    `json:"product_type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	ImageURL    string    `json:"image_url"`
	Quantity    int       `json:"quantity"`
	Size        string    `json:"size,omitempty"`
}

type CartItemWithProduct struct {
	ItemID      uuid.UUID
	ProductID   uuid.UUID
	ProductType string
	Name        string
	Description string
	PriceCents  int64
	ImageKey    string
	Quantity    int
	Size        string
	CreatedAt   time.Time
}
