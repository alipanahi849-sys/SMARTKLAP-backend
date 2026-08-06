package repository

import (
	"context"

	"clap/internal/modules/shop/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductSizeStockRepository interface {
	ListByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductSizeStock, error)
	ListByProductIDs(ctx context.Context, productIDs []uuid.UUID) (map[uuid.UUID][]models.ProductSizeStock, error)
	ReplaceForProduct(ctx context.Context, productID uuid.UUID, stocks []models.ProductSizeStock) error
	DecrementForOrder(ctx context.Context, productID uuid.UUID, size string, quantity int) error
}

type productSizeStockRepository struct {
	db *gorm.DB
}

func NewProductSizeStockRepository(db *gorm.DB) ProductSizeStockRepository {
	return &productSizeStockRepository{db: db}
}

func (r *productSizeStockRepository) ListByProductID(ctx context.Context, productID uuid.UUID) ([]models.ProductSizeStock, error) {
	var rows []models.ProductSizeStock
	if err := r.db.WithContext(ctx).
		Where("product_id = ?", productID).
		Order("size ASC").
		Find(&rows).Error; err != nil {
		return nil, errors.NewInternal("Failed to load product size stock", err)
	}
	return rows, nil
}

func (r *productSizeStockRepository) ListByProductIDs(ctx context.Context, productIDs []uuid.UUID) (map[uuid.UUID][]models.ProductSizeStock, error) {
	out := make(map[uuid.UUID][]models.ProductSizeStock)
	if len(productIDs) == 0 {
		return out, nil
	}

	var rows []models.ProductSizeStock
	if err := r.db.WithContext(ctx).
		Where("product_id IN ?", productIDs).
		Order("product_id ASC, size ASC").
		Find(&rows).Error; err != nil {
		return nil, errors.NewInternal("Failed to load product size stocks", err)
	}

	for _, row := range rows {
		out[row.ProductID] = append(out[row.ProductID], row)
	}
	return out, nil
}

func (r *productSizeStockRepository) ReplaceForProduct(ctx context.Context, productID uuid.UUID, stocks []models.ProductSizeStock) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("product_id = ?", productID).Delete(&models.ProductSizeStock{}).Error; err != nil {
			return errors.NewInternal("Failed to clear product size stock", err)
		}
		if len(stocks) == 0 {
			return nil
		}
		for i := range stocks {
			stocks[i].ProductID = productID
		}
		if err := tx.Create(&stocks).Error; err != nil {
			return errors.NewInternal("Failed to save product size stock", err)
		}
		return nil
	})
}

func (r *productSizeStockRepository) DecrementForOrder(ctx context.Context, productID uuid.UUID, size string, quantity int) error {
	if quantity <= 0 {
		return errors.NewBadRequest("quantity must be positive", nil)
	}

	res := r.db.WithContext(ctx).Exec(`
		UPDATE product_size_stocks
		SET stock_quantity = stock_quantity - ?,
		    sold_out = CASE WHEN stock_quantity - ? = 0 THEN TRUE ELSE sold_out END,
		    updated_at = NOW()
		WHERE product_id = ?
		  AND size = ?
		  AND stock_quantity IS NOT NULL
		  AND stock_quantity >= ?
	`, quantity, quantity, productID, size, quantity)
	if res.Error != nil {
		return errors.NewInternal("Failed to decrement product size stock", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.NewUnprocessable("Insufficient stock for product size", nil)
	}
	return nil
}
