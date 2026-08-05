package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Product types (Mobile API Contract §6–§7).
const (
	ProductTypeFood  = "food"
	ProductTypeMerch = "merch"
)

// Merch categories (§7).
const (
	CategoryTShirts   = "t-shirts"
	CategoryBalls     = "balls"
	CategoryStickers  = "stickers"
	CategorySportSuit = "sport-suits"
)

// Food categories (§6).
const (
	CategorySandwiches = "sandwiches"
	CategoryFoodSnacks = "snacks"
	CategoryDrinks     = "drinks"
)

// Product is a shop catalog item (food or merch).
type Product struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ProductType    string         `gorm:"type:varchar(20);not null;default:merch" json:"product_type"`
	Name           string         `gorm:"type:varchar(200);not null" json:"name"`
	Subname        string         `gorm:"type:varchar(200);not null;default:''" json:"subname"`
	Description    string         `gorm:"type:text;not null;default:''" json:"description"`
	Category       string         `gorm:"type:varchar(50);not null" json:"category"`
	PriceCents     int64          `gorm:"not null" json:"price_cents"`
	PricePoints    int            `gorm:"not null" json:"price_points"`
	ImageKey       string         `gorm:"type:varchar(500);not null;default:''" json:"-"`
	SellerName     string         `gorm:"type:varchar(200);not null;default:''" json:"seller_name"`
	AvailableSizes string         `gorm:"type:jsonb;not null;default:'[]'" json:"available_sizes"`
	StockQuantity  *int           `gorm:"column:stock_quantity" json:"stock_quantity"`
	SoldOut        bool           `gorm:"column:sold_out;not null;default:false" json:"sold_out"`
	IsActive       bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Product) TableName() string {
	return "products"
}

func (p Product) HasSizes() bool {
	return p.ProductType == ProductTypeMerch
}

func (p Product) IsUnlimitedStock() bool {
	return p.StockQuantity == nil
}

func (p Product) InStock() bool {
	if p.IsUnlimitedStock() {
		return true
	}
	return *p.StockQuantity > 0
}

// IsSoldOut is true when limited inventory was depleted by orders (not admin zero-stock).
func (p Product) IsSoldOut() bool {
	return !p.IsUnlimitedStock() && !p.InStock() && p.SoldOut
}
