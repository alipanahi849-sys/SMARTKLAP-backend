package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

// CatalogItem is a snack or product row (contract §6.1 / §7.1).
type CatalogItem struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	ImageURL    string    `json:"image_url"`
}

// CatalogListResponse is the paginated snack/product listing.
type CatalogListResponse struct {
	Items     []CatalogItem  `json:"items"`
	CartCount int            `json:"cart_count"`
	Meta      utils.ListMeta `json:"meta"`
}

// ProductDetailResponse is GET /products/{id} (contract §7.2).
type ProductDetailResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	SellerName     string    `json:"seller_name"`
	Description    string    `json:"description"`
	Price          string    `json:"price"`
	ImageURL       string    `json:"image_url"`
	AvailableSizes []string  `json:"available_sizes"`
}

// AddCartItemRequest is POST /cart/items (contract §6.3).
type AddCartItemRequest struct {
	ProductType string    `json:"product_type" binding:"required,oneof=snack merch"`
	ProductID   uuid.UUID `json:"product_id" binding:"required"`
	Quantity    int       `json:"quantity" binding:"required,min=1,max=99"`
	Size        string    `json:"size" binding:"omitempty,max=10"`
}

// UpdateCartItemRequest is PATCH /cart/items/{item_id}.
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1,max=99"`
}

// CartItemView is one item inside a cart group.
type CartItemView struct {
	ID       uuid.UUID `json:"id"`
	ImageURL string    `json:"image_url"`
	Quantity int       `json:"quantity"`
}

// CartGroup groups the cart by product type ("Food Delivery" / "Store").
type CartGroup struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	Date      string         `json:"date"`
	Items     []CartItemView `json:"items"`
	ExtraText string         `json:"extra_text"`
}

// CartResponse is GET /cart (contract §6.3).
type CartResponse struct {
	Orders []CartGroup `json:"orders"`
}

// CheckoutRequest is POST /orders (contract §6.4).
type CheckoutRequest struct {
	DeliveryMethod string `json:"delivery_method" binding:"required,oneof=seat pickup"`
	SeatNumber     string `json:"seat_number" binding:"omitempty,max=20"`
}

// OrderItemView is a line in the checkout response.
type OrderItemView struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Quantity int       `json:"quantity"`
	Price    string    `json:"price"`
}

// OrderResponse is the checkout result.
type OrderResponse struct {
	OrderID  uuid.UUID       `json:"order_id"`
	Items    []OrderItemView `json:"items"`
	Subtotal string          `json:"subtotal"`
	Shipping string          `json:"shipping"`
	Total    string          `json:"total"`
	Status   string          `json:"status"`
}

// PayOrderRequest is POST /orders/{order_id}/pay.
type PayOrderRequest struct {
	PaymentMethod string `json:"payment_method" binding:"required,max=30"`
}

// PayOrderResponse confirms payment.
type PayOrderResponse struct {
	Status string `json:"status"`
}
