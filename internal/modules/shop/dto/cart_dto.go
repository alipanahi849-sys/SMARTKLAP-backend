package dto

import "github.com/google/uuid"

// AddCartItemRequest is the body for POST /api/v1/shop/cart/items.
type AddCartItemRequest struct {
	ProductID   uuid.UUID `json:"product_id" binding:"required"`
	ProductType string    `json:"product_type" binding:"required"`
	Quantity    int       `json:"quantity" binding:"omitempty,min=1"`
	Size        string    `json:"size"`
}

// DecreaseCartItemRequest is the body for POST /api/v1/shop/cart/items/decrease.
type DecreaseCartItemRequest struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"omitempty,min=1"`
	Size      string    `json:"size"`
}

// CartMutationResponse is returned after add/decrease operations.
type CartMutationResponse struct {
	ItemID    *uuid.UUID `json:"item_id,omitempty"`
	ProductID uuid.UUID  `json:"product_id"`
	Quantity  int        `json:"quantity"`
	CartCount int        `json:"cart_count"`
}

// BasketOrderItem is one preview row on the Basket screen.
type BasketOrderItem struct {
	ID       uuid.UUID `json:"id"`
	ImageURL string    `json:"image_url"`
	Quantity int       `json:"quantity"`
}

// BasketOrder groups cart lines by product type for the Basket screen.
type BasketOrder struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Date      string            `json:"date"`
	Items     []BasketOrderItem `json:"items"`
	ExtraText string            `json:"extra_text,omitempty"`
}

// BasketResponse is GET /api/v1/shop/cart.
type BasketResponse struct {
	Orders []BasketOrder `json:"orders"`
}
