package models

import (
	"time"

	"github.com/google/uuid"
)

// ProductSizeStock holds inventory for one merch size variant.
type ProductSizeStock struct {
	ProductID     uuid.UUID `gorm:"type:uuid;primaryKey" json:"product_id"`
	Size          string    `gorm:"type:varchar(50);primaryKey" json:"size"`
	StockQuantity *int      `gorm:"column:stock_quantity" json:"stock_quantity"`
	SoldOut       bool      `gorm:"column:sold_out;not null;default:false" json:"sold_out"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ProductSizeStock) TableName() string {
	return "product_size_stocks"
}

func (s ProductSizeStock) IsUnlimitedStock() bool {
	return s.StockQuantity == nil
}

func (s ProductSizeStock) InStock() bool {
	if s.IsUnlimitedStock() {
		return true
	}
	return *s.StockQuantity > 0
}

func (s ProductSizeStock) IsSoldOut() bool {
	return !s.IsUnlimitedStock() && !s.InStock() && s.SoldOut
}
