package service

import (
	"context"
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

// ProductService implements the mobile Store screens (contract §7).
type ProductService interface {
	List(ctx context.Context, userID uuid.UUID, page, limit int, filters dto.ProductListFilters) (*dto.ProductListResponse, error)
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
	currency := strings.ToUpper(strings.TrimSpace(filters.Currency))
	if currency == "" {
		currency = dto.CurrencyEUR
	}
	if currency != dto.CurrencyEUR && currency != dto.CurrencyPoint {
		return nil, errors.NewBadRequest("currency must be 'EUR' or 'POINT'", nil)
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
