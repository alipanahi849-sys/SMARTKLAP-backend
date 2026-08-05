package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clap/internal/modules/cart/dto"
	"clap/internal/modules/cart/models"
	cartrepo "clap/internal/modules/cart/repository"
	shopmodels "clap/internal/modules/shop/models"
	shoprepo "clap/internal/modules/shop/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/utils"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

const imageURLExpiry = 6 * time.Hour

const (
	maxPreviewItems = 3
	groupTitleFood  = "Food Delivery"
	groupTitleMerch = "Club Online Store"
)

type CartService interface {
	GetCart(ctx context.Context, userID uuid.UUID) (*dto.CartResponse, error)
	AddItem(ctx context.Context, userID uuid.UUID, req *dto.AddCartItemRequest) (*dto.CartMutationResponse, error)
	UpdateItem(ctx context.Context, userID, itemID uuid.UUID, req *dto.UpdateCartItemRequest) (*dto.CartMutationResponse, error)
	RemoveItem(ctx context.Context, userID, itemID uuid.UUID) (*dto.CartMutationResponse, error)
	CountItems(ctx context.Context, userID uuid.UUID) (int, error)
	ClearCart(ctx context.Context, userID uuid.UUID) error
	ListItemDetails(ctx context.Context, userID uuid.UUID) ([]dto.CartItemWithProduct, error)
}

type cartService struct {
	cartRepo    cartrepo.CartRepository
	productRepo shoprepo.ProductRepository
	storage     storage.StorageProvider
}

func NewCartService(
	cartRepo cartrepo.CartRepository,
	productRepo shoprepo.ProductRepository,
	storageProvider storage.StorageProvider,
) CartService {
	return &cartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
		storage:     storageProvider,
	}
}

func (s *cartService) GetCart(ctx context.Context, userID uuid.UUID) (*dto.CartResponse, error) {
	details, err := s.ListItemDetails(ctx, userID)
	if err != nil {
		return nil, err
	}

	groups := buildCartGroups(details, s.resolveURL)

	lines := make([]dto.CartItemLine, len(details))
	for i, item := range details {
		lines[i] = dto.CartItemLine{
			ID:        item.ItemID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	return &dto.CartResponse{Orders: groups, Items: lines}, nil
}

func (s *cartService) AddItem(ctx context.Context, userID uuid.UUID, req *dto.AddCartItemRequest) (*dto.CartMutationResponse, error) {
	productType, err := normalizeProductType(req.ProductType)
	if err != nil {
		return nil, err
	}

	productID, err := uuid.Parse(strings.TrimSpace(req.ProductID))
	if err != nil {
		return nil, errors.NewBadRequest("Invalid product_id", nil)
	}

	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	if product.ProductType != productType {
		return nil, errors.NewBadRequest("product_type does not match product", nil)
	}

	size := strings.TrimSpace(req.Size)
	if productType == shopmodels.ProductTypeFood && size != "" {
		return nil, errors.NewBadRequest("size is only available for merch products", nil)
	}

	if productType == shopmodels.ProductTypeMerch && size != "" {
		if err := validateMerchSize(product, size); err != nil {
			return nil, err
		}
	}

	existing, err := s.cartRepo.FindByUserProductSize(ctx, userID, productID, size)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		existing.Quantity += req.Quantity
		if err := s.cartRepo.Update(ctx, existing); err != nil {
			return nil, err
		}
	} else {
		item := &models.CartItem{
			UserID:      userID,
			ProductID:   productID,
			ProductType: productType,
			Size:        size,
			Quantity:    req.Quantity,
		}
		if err := s.cartRepo.Create(ctx, item); err != nil {
			return nil, err
		}
	}

	count, err := s.cartRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.CartMutationResponse{CartCount: count}, nil
}

func (s *cartService) UpdateItem(ctx context.Context, userID, itemID uuid.UUID, req *dto.UpdateCartItemRequest) (*dto.CartMutationResponse, error) {
	item, err := s.cartRepo.FindByID(ctx, itemID, userID)
	if err != nil {
		return nil, err
	}

	item.Quantity = req.Quantity
	if err := s.cartRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	count, err := s.cartRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.CartMutationResponse{CartCount: count}, nil
}

func (s *cartService) RemoveItem(ctx context.Context, userID, itemID uuid.UUID) (*dto.CartMutationResponse, error) {
	if err := s.cartRepo.Delete(ctx, itemID, userID); err != nil {
		return nil, err
	}

	count, err := s.cartRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.CartMutationResponse{CartCount: count}, nil
}

func (s *cartService) CountItems(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.cartRepo.CountByUserID(ctx, userID)
}

func (s *cartService) ClearCart(ctx context.Context, userID uuid.UUID) error {
	return s.cartRepo.DeleteAllByUserID(ctx, userID)
}

func (s *cartService) ListItemDetails(ctx context.Context, userID uuid.UUID) ([]dto.CartItemWithProduct, error) {
	items, err := s.cartRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	details := make([]dto.CartItemWithProduct, 0, len(items))
	for _, item := range items {
		product, err := s.productRepo.FindByID(ctx, item.ProductID)
		if err != nil {
			continue
		}

		details = append(details, dto.CartItemWithProduct{
			ItemID:      item.ID,
			ProductID:   item.ProductID,
			ProductType: item.ProductType,
			Name:        product.Name,
			Description: product.Description,
			PriceCents:  product.PriceCents,
			ImageKey:    product.ImageKey,
			Quantity:    item.Quantity,
			Size:        item.Size,
			CreatedAt:   item.CreatedAt,
		})
	}

	return details, nil
}

func normalizeProductType(raw string) (string, error) {
	pt := strings.ToLower(strings.TrimSpace(raw))
	if pt == "snack" {
		pt = shopmodels.ProductTypeFood
	}
	if pt != shopmodels.ProductTypeFood && pt != shopmodels.ProductTypeMerch {
		return "", errors.NewBadRequest("product_type must be 'food' or 'merch'", nil)
	}
	return pt, nil
}

func validateMerchSize(product *shopmodels.Product, size string) error {
	// Size validation is best-effort; empty size is allowed for merch without size selection.
	if size == "" {
		return nil
	}
	raw := product.AvailableSizes
	if raw == "" || raw == "[]" {
		return nil
	}
	if strings.Contains(raw, fmt.Sprintf(`"%s"`, size)) {
		return nil
	}
	return errors.NewBadRequest("Invalid size for product", nil)
}

func buildCartGroups(details []dto.CartItemWithProduct, resolveURL func(context.Context, string) string) []dto.CartOrderGroup {
	foodItems := filterByType(details, shopmodels.ProductTypeFood)
	merchItems := filterByType(details, shopmodels.ProductTypeMerch)

	groups := make([]dto.CartOrderGroup, 0, 2)
	if group := buildGroup("food", groupTitleFood, foodItems, resolveURL); group != nil {
		groups = append(groups, *group)
	}
	if group := buildGroup("merch", groupTitleMerch, merchItems, resolveURL); group != nil {
		groups = append(groups, *group)
	}
	return groups
}

func filterByType(details []dto.CartItemWithProduct, productType string) []dto.CartItemWithProduct {
	out := make([]dto.CartItemWithProduct, 0)
	for _, d := range details {
		if d.ProductType == productType {
			out = append(out, d)
		}
	}
	return out
}

func buildGroup(id, title string, items []dto.CartItemWithProduct, resolveURL func(context.Context, string) string) *dto.CartOrderGroup {
	if len(items) == 0 {
		return nil
	}

	preview := items
	extraCount := 0
	if len(items) > maxPreviewItems {
		preview = items[:maxPreviewItems]
		extraCount = len(items) - maxPreviewItems
	}

	previewItems := make([]dto.CartItemPreview, len(preview))
	for i, item := range preview {
		previewItems[i] = dto.CartItemPreview{
			ID:       item.ItemID,
			ImageURL: resolveURL(context.Background(), item.ImageKey),
			Quantity: item.Quantity,
		}
	}

	date := items[len(items)-1].CreatedAt.Format("02/01/2006")

	group := &dto.CartOrderGroup{
		ID:    id,
		Title: title,
		Date:  date,
		Items: previewItems,
	}

	if extraCount > 0 {
		noun := "item"
		if extraCount > 1 {
			noun = "items"
		}
		group.ExtraText = fmt.Sprintf("%d more %s", extraCount, noun)
	}

	return group
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

// FormatCartItemDetail converts a cart line to checkout display format.
func FormatCartItemDetail(item dto.CartItemWithProduct, imageURL string) dto.CartItemDetail {
	detail := dto.CartItemDetail{
		ID:          item.ItemID,
		ProductID:   item.ProductID,
		ProductType: item.ProductType,
		Name:        item.Name,
		Description: item.Description,
		Price:       utils.FormatEuro(item.PriceCents),
		ImageURL:    imageURL,
		Quantity:    item.Quantity,
	}
	if item.Size != "" {
		detail.Size = item.Size
	}
	return detail
}
