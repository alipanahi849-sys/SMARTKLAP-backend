package dto

import (
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
	Cursor      *uuid.UUID
	Limit       int
}

// CursorListMeta is cursor pagination meta for GET /api/v1/shop.
type CursorListMeta struct {
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
	NextCursor *uuid.UUID `json:"next_cursor,omitempty"`
}

// ProductDetailFilters are query params for GET /api/v1/shop/{id}.
type ProductDetailFilters struct {
	Currency string
	Size     string
}

// SizeStockInput is per-size inventory on create/update for sized merch.
type SizeStockInput struct {
	Size          string `json:"size" binding:"required"`
	StockQuantity *int   `json:"stock_quantity"`
}

// SizeStockInfo is inventory for one size in list/detail responses.
type SizeStockInfo struct {
	Size          string `json:"size"`
	StockQuantity *int   `json:"stock_quantity,omitempty"`
	IsUnlimited   bool   `json:"is_unlimited"`
	InStock       bool   `json:"in_stock"`
	SoldOut       bool   `json:"soldout,omitempty"`
}

// CreateProductRequest is the body for POST /api/v1/shop.
type CreateProductRequest struct {
	ProductType    string           `json:"product_type" binding:"required"`
	Name           string           `json:"name" binding:"required"`
	Subname        string           `json:"subname" binding:"max=60"`
	Description    string           `json:"description"`
	Category       string           `json:"category" binding:"required"`
	PriceCents     int64            `json:"price_cents" binding:"required,min=0"`
	PricePoints    int              `json:"price_points" binding:"required,min=0"`
	TaxRate        *float64         `json:"tax_rate" binding:"required"`
	DiscountRate   *float64         `json:"discount_rate"`
	ImageURL       string           `json:"image_url"`
	SellerName     string           `json:"seller_name"`
	AvailableSizes []string         `json:"available_sizes"`
	IsActive       *bool            `json:"is_active"`
	StockQuantity  *int             `json:"stock_quantity"`
	SizeStock      []SizeStockInput `json:"size_stock"`
}

// UpdateProductRequest is the body for PUT /api/v1/shop/{id}.
type UpdateProductRequest struct {
	ProductType    string           `json:"product_type" binding:"required"`
	Name           string           `json:"name" binding:"required"`
	Subname        string           `json:"subname" binding:"max=60"`
	Description    string           `json:"description"`
	Category       string           `json:"category" binding:"required"`
	PriceCents     int64            `json:"price_cents" binding:"required,min=0"`
	PricePoints    int              `json:"price_points" binding:"required,min=0"`
	TaxRate        *float64         `json:"tax_rate" binding:"required"`
	DiscountRate   *float64         `json:"discount_rate"`
	ImageURL       string           `json:"image_url"`
	SellerName     string           `json:"seller_name"`
	AvailableSizes []string         `json:"available_sizes"`
	IsActive       *bool            `json:"is_active"`
	StockQuantity  *int             `json:"stock_quantity"`
	SizeStock      []SizeStockInput `json:"size_stock"`
}

// ProductStockInfo describes inventory availability for list and detail responses.
type ProductStockInfo struct {
	StockQuantity *int            `json:"stock_quantity,omitempty"`
	IsUnlimited   bool            `json:"is_unlimited"`
	InStock       bool            `json:"in_stock"`
	SoldOut       bool            `json:"soldout,omitempty"`
	BySize        []SizeStockInfo `json:"by_size,omitempty"`
}

// ProductItem is a single row on the shop list screen.
type ProductItem struct {
	ID            uuid.UUID        `json:"id"`
	ProductType   string           `json:"product_type"`
	Name          string           `json:"name"`
	Subname       string           `json:"subname,omitempty"`
	Description   string           `json:"description"`
	Price         string           `json:"price"`
	OriginalPrice string           `json:"original_price,omitempty"`
	TaxRate       float64          `json:"tax_rate"`
	DiscountRate  float64          `json:"discount_rate,omitempty"`
	ImageURL      string           `json:"image_url"`
	Stock         ProductStockInfo `json:"stock"`
}

// ProductListResponse is GET /api/v1/shop.
type ProductListResponse struct {
	Items      []ProductItem  `json:"items"`
	CartCount  int            `json:"cart_count"`
	UserPoints int            `json:"user_points"`
	Meta       CursorListMeta `json:"meta"`
}

// ProductDetailResponse is GET /api/v1/shop/{id}.
type ProductDetailResponse struct {
	ID             uuid.UUID        `json:"id"`
	ProductType    string           `json:"product_type"`
	Name           string           `json:"name"`
	Subname        string           `json:"subname,omitempty"`
	SellerName     string           `json:"seller_name,omitempty"`
	Description    string           `json:"description"`
	Price          string           `json:"price"`
	OriginalPrice  string           `json:"original_price,omitempty"`
	TaxRate        float64          `json:"tax_rate"`
	DiscountRate   float64          `json:"discount_rate,omitempty"`
	ImageURL       string           `json:"image_url"`
	AvailableSizes []string         `json:"available_sizes,omitempty"`
	Stock          ProductStockInfo `json:"stock"`
}

// ImageUploadResponse is POST /api/v1/shop/{id}/image.
type ImageUploadResponse struct {
	ProductID uuid.UUID `json:"product_id"`
	ImageURL  string    `json:"image_url"`
}
