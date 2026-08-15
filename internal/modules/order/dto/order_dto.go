package dto

import "github.com/google/uuid"

// CreateOrderRequest is POST /api/v1/orders.
type CreateOrderRequest struct {
	DeliveryMethod string  `json:"delivery_method" binding:"required,oneof=seat pickup"`
	SeatNumber     *string `json:"seat_number"`
	Currency       string  `json:"currency"`
}

// PayOrderRequest is POST /api/v1/orders/{order_id}/pay.
type PayOrderRequest struct {
	PaymentMethod string `json:"payment_method" binding:"required,oneof=points card"`
}

// UpdateOrderRequest is PATCH /api/v1/orders/{order_id} for unpaid orders.
type UpdateOrderRequest struct {
	DeliveryMethod *string `json:"delivery_method"`
	SeatNumber     *string `json:"seat_number"`
	PaymentMethod  *string `json:"payment_method"`
}

// CalculateOrderRequest is POST /api/v1/orders/calculate.
type CalculateOrderRequest struct {
	DeliveryMethod string `json:"delivery_method" binding:"required,oneof=seat pickup"`
	PaymentMethod  string `json:"payment_method" binding:"required,oneof=points card"`
}

// CalculateOrderResponse previews checkout totals for the current cart.
type CalculateOrderResponse struct {
	DeliveryMethod   string `json:"delivery_method"`
	PaymentMethod    string `json:"payment_method"`
	Subtotal         string `json:"subtotal"`
	Total            string `json:"total"`
	DeliverySavings  string `json:"delivery_savings,omitempty"`
	PaymentAmount    string `json:"payment_amount"`
	PointsRequired   int    `json:"points_required"`
	UserPoints       int    `json:"user_points"`
	SufficientPoints bool   `json:"sufficient_points"`
}

// OrderListFilters are query params for GET /api/v1/orders.
type OrderListFilters struct {
	Cursor *uuid.UUID
	Limit  int
}

// CursorListMeta is cursor pagination meta for GET /api/v1/orders.
type CursorListMeta struct {
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
	NextCursor *uuid.UUID `json:"next_cursor,omitempty"`
}

// OrderListPreviewItem is a preview row on the orders list (up to 3 per order).
type OrderListPreviewItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Name      string    `json:"name"`
	ImageURL  string    `json:"image_url"`
	Quantity  int       `json:"quantity"`
}

// OrderListItem is one row in GET /api/v1/orders.
type OrderListItem struct {
	OrderID        uuid.UUID              `json:"order_id"`
	Status         string                 `json:"status"`
	DeliveryMethod string                 `json:"delivery_method"`
	SeatNumber     string                 `json:"seat_number,omitempty"`
	Subtotal       string                 `json:"subtotal"`
	Shipping       string                 `json:"shipping,omitempty"`
	Total          string                 `json:"total"`
	PaymentMethod  string                 `json:"payment_method,omitempty"`
	ItemCount      int                    `json:"item_count"`
	Items          []OrderListPreviewItem `json:"items,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	ExpiresAt      string                 `json:"expires_at,omitempty"`
}

// OrderListResponse is GET /api/v1/orders.
type OrderListResponse struct {
	Items []OrderListItem `json:"items"`
	Meta  CursorListMeta  `json:"meta"`
}

// OrderDetailItem is one line in GET /api/v1/orders/{order_id}.
type OrderDetailItem struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	ProductType string    `json:"product_type"`
	Size        string    `json:"size,omitempty"`
	Name        string    `json:"name"`
	Subname     string    `json:"subname,omitempty"`
	Price       string    `json:"price"`
	ImageURL    string    `json:"image_url"`
	Quantity    int       `json:"quantity"`
}

// OrderDetailResponse is GET /api/v1/orders/{order_id}.
type OrderDetailResponse struct {
	OrderID        uuid.UUID         `json:"order_id"`
	Status         string            `json:"status"`
	DeliveryMethod string            `json:"delivery_method"`
	SeatNumber     string            `json:"seat_number,omitempty"`
	Subtotal       string            `json:"subtotal"`
	Shipping       string            `json:"shipping,omitempty"`
	Total          string            `json:"total"`
	PaymentMethod  string            `json:"payment_method,omitempty"`
	ItemCount      int               `json:"item_count"`
	Items          []OrderDetailItem `json:"items"`
	CreatedAt      string            `json:"created_at"`
	ExpiresAt      string            `json:"expires_at,omitempty"`
	PaidAt         string            `json:"paid_at,omitempty"`
}

// OrderLineItem is one row in checkout/order responses.
type OrderLineItem struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Quantity int       `json:"quantity"`
	Price    string    `json:"price"`
}

// OrderResponse matches Mobile API Contract §6.4.
type OrderResponse struct {
	OrderID  uuid.UUID       `json:"order_id"`
	Items    []OrderLineItem `json:"items"`
	Subtotal string          `json:"subtotal"`
	Shipping string          `json:"shipping,omitempty"`
	Total    string          `json:"total"`
	Status   string          `json:"status"`
}

// PayOrderResponse is returned after initiating or completing payment.
type PayOrderResponse struct {
	Status          string  `json:"status"`
	PointsRemaining *int    `json:"points_remaining,omitempty"`
	CheckoutURL     *string `json:"checkout_url,omitempty"`
}
