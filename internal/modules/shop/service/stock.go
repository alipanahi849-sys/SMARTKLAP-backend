package service

import (
	"strings"

	"clap/internal/modules/shop/dto"
	"clap/internal/modules/shop/models"
	"clap/internal/shared/errors"
)

func usesSizeStock(sizes []string, sizeStocks []models.ProductSizeStock) bool {
	return len(sizes) > 0 && len(sizeStocks) > 0
}

func toSizeStockInfo(s models.ProductSizeStock) dto.SizeStockInfo {
	info := dto.SizeStockInfo{
		Size:        s.Size,
		IsUnlimited: s.IsUnlimitedStock(),
		InStock:     s.InStock(),
		SoldOut:     s.IsSoldOut(),
	}
	if !info.IsUnlimited {
		qty := *s.StockQuantity
		info.StockQuantity = &qty
	}
	return info
}

func buildProductStockInfo(p *models.Product, sizeStocks []models.ProductSizeStock, filterSize string) dto.ProductStockInfo {
	sizes := parseAvailableSizes(p.AvailableSizes)
	if usesSizeStock(sizes, sizeStocks) {
		bySize := make([]dto.SizeStockInfo, len(sizeStocks))
		anyInStock := false
		allUnlimited := true
		for i, row := range sizeStocks {
			bySize[i] = toSizeStockInfo(row)
			if bySize[i].InStock {
				anyInStock = true
			}
			if !bySize[i].IsUnlimited {
				allUnlimited = false
			}
		}

		if filterSize != "" {
			for _, row := range bySize {
				if row.Size == filterSize {
					return sizeStockInfoToProductStock(row, nil)
				}
			}
			return dto.ProductStockInfo{InStock: false, IsUnlimited: false}
		}

		info := dto.ProductStockInfo{
			IsUnlimited: allUnlimited,
			InStock:     anyInStock,
			BySize:      bySize,
		}
		return info
	}

	return toProductStockInfo(p)
}

func sizeStockInfoToProductStock(row dto.SizeStockInfo, bySize []dto.SizeStockInfo) dto.ProductStockInfo {
	info := dto.ProductStockInfo{
		IsUnlimited: row.IsUnlimited,
		InStock:     row.InStock,
		SoldOut:     row.SoldOut,
		BySize:      bySize,
	}
	if !row.IsUnlimited && row.StockQuantity != nil {
		qty := *row.StockQuantity
		info.StockQuantity = &qty
	}
	return info
}

func resolveProductStockInputs(
	productType string,
	sizes []string,
	stockQuantity *int,
	sizeStock []dto.SizeStockInput,
) (*int, []models.ProductSizeStock, error) {
	if productType != models.ProductTypeMerch || len(sizes) == 0 {
		if len(sizeStock) > 0 {
			return nil, nil, errors.NewBadRequest("size_stock is only available for merch products with available_sizes", nil)
		}
		qty, err := validateStockQuantity(stockQuantity)
		if err != nil {
			return nil, nil, err
		}
		return qty, nil, nil
	}

	if stockQuantity != nil {
		return nil, nil, errors.NewBadRequest("stock_quantity cannot be used when available_sizes is set; use size_stock instead", nil)
	}

	rows, err := validateSizeStockInputs(sizes, sizeStock)
	if err != nil {
		return nil, nil, err
	}
	return nil, rows, nil
}

func validateSizeStockInputs(sizes []string, inputs []dto.SizeStockInput) ([]models.ProductSizeStock, error) {
	if len(inputs) == 0 {
		return nil, errors.NewBadRequest("size_stock is required when available_sizes is set", nil)
	}
	if len(inputs) != len(sizes) {
		return nil, errors.NewBadRequest("size_stock must include every available size exactly once", nil)
	}

	seen := make(map[string]struct{}, len(sizes))
	allowed := make(map[string]struct{}, len(sizes))
	for _, size := range sizes {
		allowed[size] = struct{}{}
	}

	rows := make([]models.ProductSizeStock, 0, len(inputs))
	for _, input := range inputs {
		size := strings.TrimSpace(input.Size)
		if size == "" {
			return nil, errors.NewBadRequest("size_stock.size is required", nil)
		}
		if _, ok := allowed[size]; !ok {
			return nil, errors.NewBadRequest("size_stock contains size not in available_sizes", nil)
		}
		if _, dup := seen[size]; dup {
			return nil, errors.NewBadRequest("size_stock contains duplicate size", nil)
		}
		seen[size] = struct{}{}

		qty, err := validateStockQuantity(input.StockQuantity)
		if err != nil {
			return nil, err
		}
		rows = append(rows, models.ProductSizeStock{
			Size:          size,
			StockQuantity: qty,
		})
	}

	for size := range allowed {
		if _, ok := seen[size]; !ok {
			return nil, errors.NewBadRequest("size_stock is missing size: "+size, nil)
		}
	}

	return rows, nil
}

func findSizeStock(sizeStocks []models.ProductSizeStock, size string) (*models.ProductSizeStock, bool) {
	for i := range sizeStocks {
		if sizeStocks[i].Size == size {
			return &sizeStocks[i], true
		}
	}
	return nil, false
}

func validateCartQuantityAgainstSizeStock(sizeStock *models.ProductSizeStock, requestedQty int) error {
	if sizeStock.IsUnlimitedStock() {
		return nil
	}
	if requestedQty > *sizeStock.StockQuantity {
		return errors.NewUnprocessable("Requested quantity exceeds available stock for size", nil)
	}
	return nil
}
