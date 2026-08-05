package dto

import "github.com/google/uuid"

// AddCartItemRequest is the body for POST /api/v1/cart/items.
type AddCartItemRequest struct {
	ProductID   uuid.UUID `json:"product_id" binding:"required"`
	ProductType string    `json:"product_type" binding:"required"`
	Quantity    int       `json:"quantity" binding:"omitempty,min=1"`
	Size        string    `json:"size"`
}

// DecreaseCartItemRequest is the body for POST /api/v1/cart/items/decrease.
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
