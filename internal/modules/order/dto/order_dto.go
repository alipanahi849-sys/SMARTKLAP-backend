package dto

import "github.com/google/uuid"

type CreateOrderRequest struct {
	DeliveryMethod string `json:"delivery_method" binding:"required"`
	SeatNumber     string `json:"seat_number"`
}

type PayOrderRequest struct {
	PaymentMethod string `json:"payment_method" binding:"required"`
}

type OrderItemResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Quantity int       `json:"quantity"`
	Price    string    `json:"price"`
}

type CreateOrderResponse struct {
	OrderID  uuid.UUID           `json:"order_id"`
	Items    []OrderItemResponse `json:"items"`
	Subtotal string              `json:"subtotal"`
	Shipping string              `json:"shipping"`
	Total    string              `json:"total"`
	Status   string              `json:"status"`
}

type PayOrderResponse struct {
	Status string `json:"status"`
}

type CheckoutPreviewResponse struct {
	Items    []CheckoutItemResponse `json:"items"`
	Subtotal string                 `json:"subtotal"`
	Shipping string                 `json:"shipping"`
	Total    string                 `json:"total"`
}

type CheckoutItemResponse struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	ImageURL    string    `json:"image_url"`
	Quantity    int       `json:"quantity"`
	Size        string    `json:"size,omitempty"`
}
