package service

import (
	"context"
	"encoding/json"
	"strings"

	"clap/internal/modules/cart/dto"
	cartmodels "clap/internal/modules/cart/models"
	cartrepository "clap/internal/modules/cart/repository"
	shopmodels "clap/internal/modules/shop/models"
	shoprepo "clap/internal/modules/shop/repository"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
)

// CartService manages the user shopping cart without changing product stock.
type CartService interface {
	AddItem(ctx context.Context, userID uuid.UUID, req *dto.AddCartItemRequest) (*dto.CartMutationResponse, error)
	DecreaseItem(ctx context.Context, userID uuid.UUID, req *dto.DecreaseCartItemRequest) (*dto.CartMutationResponse, error)
	CountItems(ctx context.Context, userID uuid.UUID) (int, error)
}

type cartService struct {
	cartRepo    cartrepository.CartRepository
	productRepo shoprepo.ProductRepository
}

func NewCartService(
	cartRepo cartrepository.CartRepository,
	productRepo shoprepo.ProductRepository,
) CartService {
	return &cartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
	}
}

func (s *cartService) AddItem(ctx context.Context, userID uuid.UUID, req *dto.AddCartItemRequest) (*dto.CartMutationResponse, error) {
	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}

	productType, err := normalizeProductType(req.ProductType)
	if err != nil {
		return nil, err
	}

	size, err := normalizeCartSize(productType, req.Size)
	if err != nil {
		return nil, err
	}

	product, err := s.productRepo.FindByID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}

	if product.ProductType != productType {
		return nil, errors.NewBadRequest("product_type does not match product", nil)
	}

	if err := validateProductForCart(product, size); err != nil {
		return nil, err
	}

	existing, err := s.cartRepo.FindLine(ctx, userID, req.ProductID, size)
	if err != nil {
		return nil, err
	}

	newQty := qty
	if existing != nil {
		newQty = existing.Quantity + qty
	}

	if err := validateCartQuantityAgainstStock(product, newQty); err != nil {
		return nil, err
	}

	var itemID uuid.UUID
	if existing == nil {
		item := &cartmodels.CartItem{
			UserID:      userID,
			ProductID:   req.ProductID,
			ProductType: productType,
			Size:        size,
			Quantity:    newQty,
		}
		if err := s.cartRepo.Create(ctx, item); err != nil {
			return nil, err
		}
		itemID = item.ID
	} else {
		if err := s.cartRepo.UpdateQuantity(ctx, existing.ID, newQty); err != nil {
			return nil, err
		}
		itemID = existing.ID
	}

	cartCount, err := s.cartRepo.CountTotalQuantity(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.CartMutationResponse{
		ItemID:    &itemID,
		ProductID: req.ProductID,
		Quantity:  newQty,
		CartCount: cartCount,
	}, nil
}

func (s *cartService) DecreaseItem(ctx context.Context, userID uuid.UUID, req *dto.DecreaseCartItemRequest) (*dto.CartMutationResponse, error) {
	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}

	size := strings.TrimSpace(req.Size)

	existing, err := s.cartRepo.FindLine(ctx, userID, req.ProductID, size)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.NewNotFound("Cart item not found", nil)
	}

	newQty := existing.Quantity - qty
	if newQty <= 0 {
		if err := s.cartRepo.Delete(ctx, existing.ID); err != nil {
			return nil, err
		}
		newQty = 0
	} else if err := s.cartRepo.UpdateQuantity(ctx, existing.ID, newQty); err != nil {
		return nil, err
	}

	cartCount, err := s.cartRepo.CountTotalQuantity(ctx, userID)
	if err != nil {
		return nil, err
	}

	resp := &dto.CartMutationResponse{
		ProductID: req.ProductID,
		Quantity:  newQty,
		CartCount: cartCount,
	}
	if newQty > 0 {
		itemID := existing.ID
		resp.ItemID = &itemID
	}
	return resp, nil
}

func (s *cartService) CountItems(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.cartRepo.CountTotalQuantity(ctx, userID)
}

func normalizeProductType(raw string) (string, error) {
	pt := strings.ToLower(strings.TrimSpace(raw))
	switch pt {
	case shopmodels.ProductTypeFood, shopmodels.ProductTypeMerch:
		return pt, nil
	case "snack":
		return shopmodels.ProductTypeFood, nil
	default:
		return "", errors.NewBadRequest("product_type must be 'food' or 'merch'", nil)
	}
}

func normalizeCartSize(productType, raw string) (string, error) {
	size := strings.TrimSpace(raw)
	if productType == shopmodels.ProductTypeFood {
		if size != "" {
			return "", errors.NewBadRequest("size is only available for merch products", nil)
		}
		return "", nil
	}
	return size, nil
}

func validateProductForCart(product *shopmodels.Product, size string) error {
	if !product.InStock() {
		return errors.NewUnprocessable("Product is out of stock", nil)
	}

	if product.ProductType == shopmodels.ProductTypeMerch && size != "" {
		sizes := parseAvailableSizes(product.AvailableSizes)
		if len(sizes) > 0 {
			found := false
			for _, s := range sizes {
				if s == size {
					found = true
					break
				}
			}
			if !found {
				return errors.NewBadRequest("Size not available", nil)
			}
		}
	}

	return nil
}

func validateCartQuantityAgainstStock(product *shopmodels.Product, requestedQty int) error {
	if product.IsUnlimitedStock() {
		return nil
	}
	if requestedQty > *product.StockQuantity {
		return errors.NewUnprocessable("Requested quantity exceeds available stock", nil)
	}
	return nil
}

func parseAvailableSizes(raw string) []string {
	if raw == "" || raw == "[]" {
		return nil
	}
	var sizes []string
	if err := json.Unmarshal([]byte(raw), &sizes); err != nil {
		return nil
	}
	return sizes
}
