package service

import (
	"context"
	"fmt"
	"strings"

	"clap/internal/modules/shop/dto"
	"clap/internal/modules/shop/models"
	"clap/internal/modules/shop/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/utils"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

const basketPreviewLimit = 3

// CartService manages the user shopping cart without changing product stock.
type CartService interface {
	GetBasket(ctx context.Context, userID uuid.UUID, filters dto.BasketListFilters) (*dto.BasketResponse, error)
	AddItem(ctx context.Context, userID uuid.UUID, req *dto.AddCartItemRequest) (*dto.CartMutationResponse, error)
	DecreaseItem(ctx context.Context, userID uuid.UUID, req *dto.DecreaseCartItemRequest) (*dto.CartMutationResponse, error)
	CountItems(ctx context.Context, userID uuid.UUID) (int, error)
}

type cartService struct {
	cartRepo    repository.CartRepository
	productRepo repository.ProductRepository
	storage     storage.StorageProvider
}

func NewCartService(
	cartRepo repository.CartRepository,
	productRepo repository.ProductRepository,
	storageProvider storage.StorageProvider,
) CartService {
	return &cartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
		storage:     storageProvider,
	}
}

func (s *cartService) GetBasket(ctx context.Context, userID uuid.UUID, filters dto.BasketListFilters) (*dto.BasketResponse, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}

	var after *repository.CartCursorAnchor
	if filters.Cursor != nil {
		cursorLine, err := s.cartRepo.FindLineByID(ctx, userID, *filters.Cursor)
		if err != nil {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		after = &repository.CartCursorAnchor{
			UpdatedAt: cursorLine.UpdatedAt,
			ID:        cursorLine.ID,
		}
	}

	lines, err := s.cartRepo.ListUserLinesAfter(ctx, userID, limit+1, after)
	if err != nil {
		return nil, err
	}

	hasMore := len(lines) > limit
	if hasMore {
		lines = lines[:limit]
	}

	orders := []dto.BasketOrder{}
	if filters.Cursor == nil {
		allLines, err := s.cartRepo.ListUserLines(ctx, userID)
		if err != nil {
			return nil, err
		}
		grouped := map[string][]repository.UserCartLine{
			models.ProductTypeFood:  {},
			models.ProductTypeMerch: {},
		}
		for _, line := range allLines {
			pt := basketGroupType(line.ProductType)
			if pt != models.ProductTypeFood && pt != models.ProductTypeMerch {
				continue
			}
			grouped[pt] = append(grouped[pt], line)
		}
		if foodLines := grouped[models.ProductTypeFood]; len(foodLines) > 0 {
			orders = append(orders, buildBasketOrder(ctx, models.ProductTypeFood, foodLines, s.resolveURL))
		}
		if merchLines := grouped[models.ProductTypeMerch]; len(merchLines) > 0 {
			orders = append(orders, buildBasketOrder(ctx, models.ProductTypeMerch, merchLines, s.resolveURL))
		}
	}

	cartCount, err := s.cartRepo.CountTotalQuantity(ctx, userID)
	if err != nil {
		return nil, err
	}

	subtotalCents, err := s.cartRepo.SumSubtotalCents(ctx, userID)
	if err != nil {
		return nil, err
	}
	subtotal := ""
	if subtotalCents > 0 {
		subtotal = utils.FormatEuro(subtotalCents)
	}

	meta := dto.CursorListMeta{
		Limit:   limit,
		HasMore: hasMore,
	}
	if hasMore && len(lines) > 0 {
		lastID := lines[len(lines)-1].ID
		meta.NextCursor = &lastID
	}

	return &dto.BasketResponse{
		Orders:    orders,
		Items:     buildCheckoutItems(ctx, lines, s.resolveURL),
		Subtotal:  subtotal,
		Shipping:  "",
		Total:     subtotal,
		CartCount: cartCount,
		Meta:      meta,
	}, nil
}

func buildBasketOrder(
	ctx context.Context,
	productType string,
	lines []repository.UserCartLine,
	resolve func(context.Context, string) string,
) dto.BasketOrder {
	title := "Club Online Store"
	if productType == models.ProductTypeFood {
		title = "Food Delivery"
	}

	latest := lines[0].UpdatedAt
	for _, line := range lines[1:] {
		if line.UpdatedAt.After(latest) {
			latest = line.UpdatedAt
		}
	}

	preview := lines
	extraText := ""
	if len(lines) > basketPreviewLimit {
		preview = lines[:basketPreviewLimit]
		remaining := len(lines) - basketPreviewLimit
		if remaining == 1 {
			extraText = "1 more item"
		} else {
			extraText = fmt.Sprintf("%d more items", remaining)
		}
	}

	items := make([]dto.BasketOrderItem, len(preview))
	for i, line := range preview {
		items[i] = dto.BasketOrderItem{
			ID:          line.ID,
			ProductID:   line.ProductID,
			ProductType: line.ProductType,
			Size:        line.Size,
			ImageURL:    resolve(ctx, line.ImageKey),
			Quantity:    line.Quantity,
		}
	}

	return dto.BasketOrder{
		ID:        productType,
		Title:     title,
		Date:      latest.Format("2006-01-02"),
		Items:     items,
		ExtraText: extraText,
	}
}

func buildCheckoutItems(ctx context.Context, lines []repository.UserCartLine, resolve func(context.Context, string) string) []dto.CheckoutLineItem {
	items := make([]dto.CheckoutLineItem, len(lines))
	for i, line := range lines {
		description := strings.TrimSpace(line.Description)
		subname := strings.TrimSpace(line.Subname)
		items[i] = dto.CheckoutLineItem{
			ID:          line.ID,
			ProductID:   line.ProductID,
			ProductType: line.ProductType,
			Size:        line.Size,
			Name:        line.Name,
			Subname:     subname,
			Description: description,
			Price:       utils.FormatEuro(line.PriceCents),
			ImageURL:    resolve(ctx, line.ImageKey),
			Quantity:    line.Quantity,
		}
	}
	return items
}

func basketGroupType(productType string) string {
	pt := strings.ToLower(strings.TrimSpace(productType))
	if pt == "snack" {
		return models.ProductTypeFood
	}
	return pt
}

func (s *cartService) resolveURL(ctx context.Context, stored string) string {
	if stored == "" {
		return ""
	}
	if strings.HasPrefix(stored, "http://") || strings.HasPrefix(stored, "https://") {
		return stored
	}
	if s.storage == nil {
		return ""
	}
	url, err := s.storage.GenerateSignedURL(ctx, stored, imageURLExpiry)
	if err != nil {
		return ""
	}
	return url
}

func (s *cartService) AddItem(ctx context.Context, userID uuid.UUID, req *dto.AddCartItemRequest) (*dto.CartMutationResponse, error) {
	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}

	productType, err := normalizeProductType(req.ProductType, true)
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
		item := &models.CartItem{
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

func normalizeCartSize(productType, raw string) (string, error) {
	size := strings.TrimSpace(raw)
	if productType == models.ProductTypeFood {
		if size != "" {
			return "", errors.NewBadRequest("size is only available for merch products", nil)
		}
		return "", nil
	}
	return size, nil
}

func validateProductForCart(product *models.Product, size string) error {
	if !product.InStock() {
		return errors.NewUnprocessable("Product is out of stock", nil)
	}

	if product.ProductType == models.ProductTypeMerch && size != "" {
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

func validateCartQuantityAgainstStock(product *models.Product, requestedQty int) error {
	if product.IsUnlimitedStock() {
		return nil
	}
	if requestedQty > *product.StockQuantity {
		return errors.NewUnprocessable("Requested quantity exceeds available stock", nil)
	}
	return nil
}
