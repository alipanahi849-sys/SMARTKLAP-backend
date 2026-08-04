package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Product categories (Mobile API Contract §7.1).
const (
	CategoryTShirts   = "t-shirts"
	CategoryBalls     = "balls"
	CategoryStickers  = "stickers"
	CategorySportSuit = "sport-suits"
)

// Product is a merch catalog item (Mobile API Contract §7).
type Product struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name           string         `gorm:"type:varchar(200);not null" json:"name"`
	Subname        string         `gorm:"type:varchar(200);not null;default:''" json:"subname"`
	Description    string         `gorm:"type:text;not null;default:''" json:"description"`
	Category       string         `gorm:"type:varchar(50);not null" json:"category"`
	PriceCents     int64          `gorm:"not null" json:"price_cents"`
	PricePoints    int            `gorm:"not null" json:"price_points"`
	ImageKey       string         `gorm:"type:varchar(500);not null;default:''" json:"image_key"`
	SellerName     string         `gorm:"type:varchar(200);not null;default:''" json:"seller_name"`
	AvailableSizes string         `gorm:"type:jsonb;not null;default:'[]'" json:"available_sizes"`
	IsActive       bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Product) TableName() string {
	return "products"
}
