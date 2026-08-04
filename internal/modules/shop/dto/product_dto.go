package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

// Currency display modes (Mobile API Contract §6–§7).
const (
	CurrencyEUR   = "EUR"
	CurrencyPoint = "POINT"
)

// ProductListFilters are query params for GET /api/v1/shop.
type ProductListFilters struct {
	Search      string
	Category    string
	Currency    string
	ProductType string
}

// ProductDetailFilters are query params for GET /api/v1/shop/{id}.
type ProductDetailFilters struct {
	Currency string
	Size     string
}

// CreateProductRequest is the body for POST /api/v1/shop.
type CreateProductRequest struct {
	ProductType    string   `json:"product_type" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	Subname        string   `json:"subname"`
	Description    string   `json:"description"`
	Category       string   `json:"category" binding:"required"`
	PriceCents     int64    `json:"price_cents" binding:"required,min=0"`
	PricePoints    int      `json:"price_points" binding:"required,min=0"`
	ImageKey       string   `json:"image_key"`
	ImageURL       string   `json:"image_url"`
	SellerName     string   `json:"seller_name"`
	AvailableSizes []string `json:"available_sizes"`
	IsActive       *bool    `json:"is_active"`
}

// ProductItem is a single row on the shop list screen.
type ProductItem struct {
	ID          uuid.UUID `json:"id"`
	ProductType string    `json:"product_type"`
	Name        string    `json:"name"`
	Subname     string    `json:"subname"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	ImageURL    string    `json:"image_url"`
}

// ProductListResponse is GET /api/v1/shop.
type ProductListResponse struct {
	Items      []ProductItem  `json:"items"`
	CartCount  int            `json:"cart_count"`
	UserPoints int            `json:"user_points"`
	Meta       utils.ListMeta `json:"meta"`
}

// ProductDetailResponse is GET /api/v1/shop/{id}.
type ProductDetailResponse struct {
	ID             uuid.UUID `json:"id"`
	ProductType    string    `json:"product_type"`
	Name           string    `json:"name"`
	SellerName     string    `json:"seller_name,omitempty"`
	Description    string    `json:"description"`
	Price          string    `json:"price"`
	ImageURL       string    `json:"image_url"`
	AvailableSizes []string  `json:"available_sizes,omitempty"`
}

// ImageUploadResponse is POST /api/v1/shop/{id}/image.
type ImageUploadResponse struct {
	ProductID uuid.UUID `json:"product_id"`
	ImageKey  string    `json:"image_key"`
	ImageURL  string    `json:"image_url"`
}
