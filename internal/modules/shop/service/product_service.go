package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/shop/dto"
	"clap/internal/modules/shop/models"
	"clap/internal/modules/shop/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/utils"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

const imageURLExpiry = 6 * time.Hour

var validCategories = map[string]bool{
	models.CategoryTShirts:   true,
	models.CategoryBalls:     true,
	models.CategoryStickers:  true,
	models.CategorySportSuit: true,
}

// ProductService implements the mobile Shop screens (contract §7).
type ProductService interface {
	List(ctx context.Context, userID uuid.UUID, page, limit int, filters dto.ProductListFilters) (*dto.ProductListResponse, error)
	GetByID(ctx context.Context, id uuid.UUID, filters dto.ProductDetailFilters) (*dto.ProductDetailResponse, error)
	Create(ctx context.Context, req *dto.CreateProductRequest, authCtx *utils.AuthorizationContext) (*dto.ProductDetailResponse, error)
}

type productService struct {
	productRepo repository.ProductRepository
	userRepo    authrepo.UserRepository
	storage     storage.StorageProvider
}

func NewProductService(
	productRepo repository.ProductRepository,
	userRepo authrepo.UserRepository,
	storageProvider storage.StorageProvider,
) ProductService {
	return &productService{
		productRepo: productRepo,
		userRepo:    userRepo,
		storage:     storageProvider,
	}
}

func (s *productService) List(ctx context.Context, userID uuid.UUID, page, limit int, filters dto.ProductListFilters) (*dto.ProductListResponse, error) {
	currency, err := parseCurrency(filters.Currency)
	if err != nil {
		return nil, err
	}

	category := strings.TrimSpace(filters.Category)
	if category != "" && !validCategories[category] {
		return nil, errors.NewBadRequest("Invalid category", nil)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	products, total, err := s.productRepo.List(ctx, page, limit, repository.ProductFilters{
		Search:   filters.Search,
		Category: category,
	})
	if err != nil {
		return nil, err
	}

	items := make([]dto.ProductItem, len(products))
	for i, p := range products {
		items[i] = dto.ProductItem{
			ID:          p.ID,
			Name:        p.Name,
			Subname:     p.Subname,
			Description: p.Description,
			Price:       formatPrice(p, currency),
			ImageURL:    s.resolveURL(ctx, p.ImageKey),
		}
	}

	return &dto.ProductListResponse{
		Items:      items,
		CartCount:  0, // cart module not implemented yet
		UserPoints: user.Points,
		Meta:       utils.NewListMeta(total, page, limit),
	}, nil
}

func (s *productService) GetByID(ctx context.Context, id uuid.UUID, filters dto.ProductDetailFilters) (*dto.ProductDetailResponse, error) {
	currency, err := parseCurrency(filters.Currency)
	if err != nil {
		return nil, err
	}

	product, err := s.productRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	sizes, err := filterAvailableSizes(product.AvailableSizes, filters.Size)
	if err != nil {
		return nil, err
	}

	return toProductDetail(product, currency, sizes, s.resolveURL(ctx, product.ImageKey)), nil
}

func (s *productService) Create(ctx context.Context, req *dto.CreateProductRequest, authCtx *utils.AuthorizationContext) (*dto.ProductDetailResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	category := strings.TrimSpace(req.Category)
	if !validCategories[category] {
		return nil, errors.NewBadRequest("Invalid category", nil)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	product := &models.Product{
		Name:           strings.TrimSpace(req.Name),
		Subname:        strings.TrimSpace(req.Subname),
		Description:    strings.TrimSpace(req.Description),
		Category:       category,
		PriceCents:     req.PriceCents,
		PricePoints:    req.PricePoints,
		ImageKey:       strings.TrimSpace(req.ImageURL),
		SellerName:     strings.TrimSpace(req.SellerName),
		AvailableSizes: marshalAvailableSizes(req.AvailableSizes),
		IsActive:       isActive,
	}

	if product.Name == "" {
		return nil, errors.NewBadRequest("name is required", nil)
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	sizes := parseAvailableSizes(product.AvailableSizes)
	return toProductDetail(product, dto.CurrencyEUR, sizes, s.resolveURL(ctx, product.ImageKey)), nil
}

func parseCurrency(raw string) (string, error) {
	currency := strings.ToUpper(strings.TrimSpace(raw))
	if currency == "" {
		return dto.CurrencyEUR, nil
	}
	if currency != dto.CurrencyEUR && currency != dto.CurrencyPoint {
		return "", errors.NewBadRequest("currency must be 'EUR' or 'POINT'", nil)
	}
	return currency, nil
}

func toProductDetail(p *models.Product, currency string, sizes []string, imageURL string) *dto.ProductDetailResponse {
	return &dto.ProductDetailResponse{
		ID:             p.ID,
		Name:           p.Name,
		SellerName:     p.SellerName,
		Description:    p.Description,
		Price:          formatPrice(*p, currency),
		ImageURL:       imageURL,
		AvailableSizes: sizes,
	}
}

func parseAvailableSizes(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var sizes []string
	if err := json.Unmarshal([]byte(raw), &sizes); err != nil {
		return []string{}
	}
	return sizes
}

func marshalAvailableSizes(sizes []string) string {
	if len(sizes) == 0 {
		return "[]"
	}
	data, err := json.Marshal(sizes)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func filterAvailableSizes(raw, size string) ([]string, error) {
	sizes := parseAvailableSizes(raw)
	size = strings.TrimSpace(size)
	if size == "" {
		return sizes, nil
	}
	for _, s := range sizes {
		if s == size {
			return []string{size}, nil
		}
	}
	return nil, errors.NewNotFound("Size not available", nil)
}

func formatPrice(p models.Product, currency string) string {
	if currency == dto.CurrencyPoint {
		return utils.FormatPoints(p.PricePoints)
	}
	return utils.FormatEuro(p.PriceCents)
}

func (s *productService) resolveURL(ctx context.Context, stored string) string {
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
