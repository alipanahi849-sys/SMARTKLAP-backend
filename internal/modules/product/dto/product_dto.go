package dto

import (
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

// Currency display modes (Mobile API Contract §7.1).
const (
	CurrencyEUR   = "EUR"
	CurrencyPoint = "POINT"
)

// ProductListFilters are query params for GET /api/v1/products.
type ProductListFilters struct {
	Search   string
	Category string
	Currency string
}

// ProductItem is a single row on the Store screen.
type ProductItem struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Subname     string    `json:"subname"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	ImageURL    string    `json:"image_url"`
}

// ProductListResponse is the Store screen payload (contract §7.1).
type ProductListResponse struct {
	Items      []ProductItem  `json:"items"`
	CartCount  int            `json:"cart_count"`
	UserPoints int            `json:"user_points"`
	Meta       utils.ListMeta `json:"meta"`
}
