package service

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/shop/dto"
	"clap/internal/modules/shop/models"
	"clap/internal/modules/shop/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/internal/shared/utils"
	"clap/pkg/media/optimize"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

const (
	imageURLExpiry      = 6 * time.Hour
	maxShopImageBytes   = 5 * 1024 * 1024
)

var allowedShopImageMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

var merchCategories = map[string]bool{
	models.CategoryTShirts:   true,
	models.CategoryBalls:     true,
	models.CategoryStickers:  true,
	models.CategorySportSuit: true,
}

var foodCategories = map[string]bool{
	models.CategorySandwiches: true,
	models.CategoryFoodSnacks: true,
	models.CategoryDrinks:     true,
}

// ProductService implements the mobile Shop screens (contract §6–§7).
type ProductService interface {
	List(ctx context.Context, userID uuid.UUID, filters dto.ProductListFilters) (*dto.ProductListResponse, error)
	GetByID(ctx context.Context, id uuid.UUID, filters dto.ProductDetailFilters) (*dto.ProductDetailResponse, error)
	Create(ctx context.Context, req *dto.CreateProductRequest, authCtx *utils.AuthorizationContext) (*dto.ProductDetailResponse, error)
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateProductRequest, authCtx *utils.AuthorizationContext) (*dto.ProductDetailResponse, error)
	Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error
	UploadProductImage(ctx context.Context, productID uuid.UUID, file *multipart.FileHeader, authCtx *utils.AuthorizationContext) (*dto.ImageUploadResponse, error)
}

// CartCounter supplies the user's current cart item count for shop list responses.
type CartCounter interface {
	CountItems(ctx context.Context, userID uuid.UUID) (int, error)
}

type productService struct {
	productRepo repository.ProductRepository
	userRepo    authrepo.UserRepository
	storage     storage.StorageProvider
	cartCounter CartCounter
	optimizer   optimize.Optimizer
}

func NewProductService(
	productRepo repository.ProductRepository,
	userRepo authrepo.UserRepository,
	storageProvider storage.StorageProvider,
	cartCounter CartCounter,
) ProductService {
	return NewProductServiceWithOptimizer(productRepo, userRepo, storageProvider, cartCounter, optimize.Noop{})
}

func NewProductServiceWithOptimizer(
	productRepo repository.ProductRepository,
	userRepo authrepo.UserRepository,
	storageProvider storage.StorageProvider,
	cartCounter CartCounter,
	optimizer optimize.Optimizer,
) ProductService {
	if optimizer == nil {
		optimizer = optimize.Noop{}
	}
	return &productService{
		productRepo: productRepo,
		userRepo:    userRepo,
		storage:     storageProvider,
		cartCounter: cartCounter,
		optimizer:   optimizer,
	}
}

func (s *productService) List(ctx context.Context, userID uuid.UUID, filters dto.ProductListFilters) (*dto.ProductListResponse, error) {
	currency, err := parseCurrency(filters.Currency)
	if err != nil {
		return nil, err
	}

	productType, err := normalizeProductType(filters.ProductType, false)
	if err != nil {
		return nil, err
	}

	category := strings.TrimSpace(filters.Category)
	if category != "" {
		if err := validateCategory(category, productType); err != nil {
			return nil, err
		}
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}

	var after *repository.CursorAnchor
	if filters.Cursor != nil {
		cursorProduct, err := s.productRepo.FindByID(ctx, *filters.Cursor)
		if err != nil {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		after = &repository.CursorAnchor{
			CreatedAt: cursorProduct.CreatedAt,
			ID:        cursorProduct.ID,
		}
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	repoFilters := repository.ProductFilters{
		Search:      filters.Search,
		Category:    category,
		ProductType: productType,
	}

	products, err := s.productRepo.List(ctx, limit+1, repoFilters, after)
	if err != nil {
		return nil, err
	}

	hasMore := len(products) > limit
	if hasMore {
		products = products[:limit]
	}

	items := make([]dto.ProductItem, len(products))
	for i, p := range products {
		items[i] = dto.ProductItem{
			ID:          p.ID,
			ProductType: p.ProductType,
			Name:        p.Name,
			Subname:     p.Subname,
			Description: p.Description,
			Price:       formatPrice(p, currency),
			ImageURL:    s.resolveURL(ctx, p.ImageKey),
			Stock:       toProductStockInfo(&p),
		}
	}

	meta := dto.CursorListMeta{
		Limit:   limit,
		HasMore: hasMore,
	}
	if hasMore && len(products) > 0 {
		lastID := products[len(products)-1].ID
		meta.NextCursor = &lastID
	}

	cartCount := 0
	if s.cartCounter != nil {
		if count, err := s.cartCounter.CountItems(ctx, userID); err == nil {
			cartCount = count
		}
	}

	return &dto.ProductListResponse{
		Items:      items,
		CartCount:  cartCount,
		UserPoints: user.Points,
		Meta:       meta,
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

	sizes, err := resolveDetailSizes(product, filters.Size)
	if err != nil {
		return nil, err
	}

	return toProductDetail(product, currency, sizes, s.resolveURL(ctx, product.ImageKey)), nil
}

func (s *productService) Create(ctx context.Context, req *dto.CreateProductRequest, authCtx *utils.AuthorizationContext) (*dto.ProductDetailResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	productType, err := normalizeProductType(req.ProductType, true)
	if err != nil {
		return nil, err
	}

	category := strings.TrimSpace(req.Category)
	if err := validateCategory(category, productType); err != nil {
		return nil, err
	}

	sizes, err := validateCreateSizes(productType, req.AvailableSizes)
	if err != nil {
		return nil, err
	}

	imageRef := strings.TrimSpace(req.ImageURL)

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.NewBadRequest("name is required", nil)
	}

	subname, err := validateSubname(req.Subname)
	if err != nil {
		return nil, err
	}

	stockQuantity, err := validateStockQuantity(req.StockQuantity)
	if err != nil {
		return nil, err
	}

	product := &models.Product{
		ProductType:    productType,
		Name:           name,
		Subname:        subname,
		Description:    strings.TrimSpace(req.Description),
		Category:       category,
		PriceCents:     req.PriceCents,
		PricePoints:    req.PricePoints,
		ImageKey:       imageRef,
		SellerName:     strings.TrimSpace(req.SellerName),
		AvailableSizes: marshalAvailableSizes(sizes),
		StockQuantity:  stockQuantity,
		IsActive:       isActive,
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	return toProductDetail(product, dto.CurrencyEUR, sizesForResponse(product, sizes), s.resolveURL(ctx, product.ImageKey)), nil
}

func (s *productService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateProductRequest, authCtx *utils.AuthorizationContext) (*dto.ProductDetailResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	product, err := s.productRepo.FindByIDAdmin(ctx, id)
	if err != nil {
		return nil, err
	}

	productType, err := normalizeProductType(req.ProductType, true)
	if err != nil {
		return nil, err
	}

	category := strings.TrimSpace(req.Category)
	if err := validateCategory(category, productType); err != nil {
		return nil, err
	}

	sizes, err := validateCreateSizes(productType, req.AvailableSizes)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.NewBadRequest("name is required", nil)
	}

	subname, err := validateSubname(req.Subname)
	if err != nil {
		return nil, err
	}

	stockQuantity, err := validateStockQuantity(req.StockQuantity)
	if err != nil {
		return nil, err
	}

	if imageURL := strings.TrimSpace(req.ImageURL); imageURL != "" {
		oldKey := product.ImageKey
		product.ImageKey = imageURL
		s.deleteStoredImage(ctx, oldKey, imageURL)
	}

	product.ProductType = productType
	product.Name = name
	product.Subname = subname
	product.Description = strings.TrimSpace(req.Description)
	product.Category = category
	product.PriceCents = req.PriceCents
	product.PricePoints = req.PricePoints
	product.SellerName = strings.TrimSpace(req.SellerName)
	product.AvailableSizes = marshalAvailableSizes(sizes)
	product.StockQuantity = stockQuantity
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	return toProductDetail(product, dto.CurrencyEUR, sizesForResponse(product, sizes), s.resolveURL(ctx, product.ImageKey)), nil
}

func (s *productService) Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error {
	if err := authCtx.RequireAdmin(); err != nil {
		return err
	}

	product, err := s.productRepo.FindByIDAdmin(ctx, id)
	if err != nil {
		return err
	}

	if err := s.productRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.deleteStoredImage(ctx, product.ImageKey, "")
	return nil
}

func (s *productService) UploadProductImage(ctx context.Context, productID uuid.UUID, file *multipart.FileHeader, authCtx *utils.AuthorizationContext) (*dto.ImageUploadResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	product, err := s.productRepo.FindByIDAdmin(ctx, productID)
	if err != nil {
		return nil, err
	}

	if file.Size > maxShopImageBytes {
		return nil, errors.NewPayloadTooLarge("Image must be at most 5 MB", nil)
	}

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(file.Filename))
	}
	ext, ok := allowedShopImageMimeTypes[strings.ToLower(mimeType)]
	if !ok {
		return nil, errors.NewUnsupportedMedia("Image must be a JPEG, PNG or WebP file", nil)
	}

	if s.storage == nil {
		return nil, errors.NewInternal("Storage is not configured", nil)
	}

	src, err := file.Open()
	if err != nil {
		return nil, errors.NewInternal("Failed to open uploaded file", err)
	}
	defer src.Close()

	prepared, err := s.optimizer.OptimizeImage(ctx, src, ext, optimize.ImageProfileProduct)
	if err != nil {
		return nil, errors.NewInternal("Failed to optimize image", err)
	}
	defer prepared.Cleanup()

	reader, err := prepared.Open()
	if err != nil {
		return nil, errors.NewInternal("Failed to read optimized image", err)
	}
	defer reader.Close()

	key := fmt.Sprintf("shop/images/%s/%s%s", productID.String(), uuid.NewString(), prepared.Extension)
	if err := s.storage.Upload(ctx, key, reader, prepared.ContentType, prepared.Size); err != nil {
		logger.Error().Err(err).Str("product_id", productID.String()).Str("key", key).Msg("shop_image_store_failed")
		return nil, errors.NewInternal("Failed to store image", err)
	}

	if err := s.productRepo.UpdateImageKey(ctx, productID, key); err != nil {
		return nil, err
	}

	s.deleteStoredImage(ctx, product.ImageKey, key)

	logger.Info().
		Str("product_id", productID.String()).
		Str("storage_key", key).
		Msg("shop_product_image_uploaded")

	return &dto.ImageUploadResponse{
		ProductID: productID,
		ImageURL:  s.resolveURL(ctx, key),
	}, nil
}

func normalizeProductType(raw string, required bool) (string, error) {
	pt := strings.ToLower(strings.TrimSpace(raw))
	if pt == "" {
		if required {
			return "", errors.NewBadRequest("product_type is required", nil)
		}
		return "", nil
	}
	if pt == "snack" {
		pt = models.ProductTypeFood
	}
	if pt != models.ProductTypeFood && pt != models.ProductTypeMerch {
		return "", errors.NewBadRequest("product_type must be 'food' or 'merch'", nil)
	}
	return pt, nil
}

func validateSubname(raw string) (string, error) {
	subname := strings.TrimSpace(raw)
	if subname == "" {
		return "", nil
	}
	if len(strings.Fields(subname)) > 3 {
		return "", errors.NewBadRequest("subname must be at most 3 words", nil)
	}
	return subname, nil
}

func validateCategory(category, productType string) error {
	category = strings.TrimSpace(category)
	if category == "" {
		return errors.NewBadRequest("category is required", nil)
	}

	switch productType {
	case models.ProductTypeFood:
		if !foodCategories[category] {
			return errors.NewBadRequest("Invalid food category", nil)
		}
	case models.ProductTypeMerch:
		if !merchCategories[category] {
			return errors.NewBadRequest("Invalid merch category", nil)
		}
	default:
		if !foodCategories[category] && !merchCategories[category] {
			return errors.NewBadRequest("Invalid category", nil)
		}
	}
	return nil
}

func validateCreateSizes(productType string, sizes []string) ([]string, error) {
	clean := make([]string, 0, len(sizes))
	for _, size := range sizes {
		size = strings.TrimSpace(size)
		if size == "" {
			continue
		}
		clean = append(clean, size)
	}

	if productType == models.ProductTypeFood {
		if len(clean) > 0 {
			return nil, errors.NewBadRequest("food products cannot have sizes", nil)
		}
		return nil, nil
	}

	return clean, nil
}

func resolveDetailSizes(product *models.Product, size string) ([]string, error) {
	if product.ProductType == models.ProductTypeFood {
		if strings.TrimSpace(size) != "" {
			return nil, errors.NewBadRequest("size is only available for merch products", nil)
		}
		return nil, nil
	}

	return filterAvailableSizes(product.AvailableSizes, size)
}

func sizesForResponse(product *models.Product, sizes []string) []string {
	if product.ProductType == models.ProductTypeFood {
		return nil
	}
	return sizes
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

func validateStockQuantity(raw *int) (*int, error) {
	if raw == nil {
		return nil, nil
	}
	if *raw < 0 {
		return nil, errors.NewBadRequest("stock_quantity must be >= 0", nil)
	}
	qty := *raw
	return &qty, nil
}

func toProductStockInfo(p *models.Product) dto.ProductStockInfo {
	info := dto.ProductStockInfo{
		IsUnlimited: p.IsUnlimitedStock(),
		InStock:     p.InStock(),
	}
	if !info.IsUnlimited {
		qty := *p.StockQuantity
		info.StockQuantity = &qty
	}
	return info
}

func toProductDetail(p *models.Product, currency string, sizes []string, imageURL string) *dto.ProductDetailResponse {
	resp := &dto.ProductDetailResponse{
		ID:          p.ID,
		ProductType: p.ProductType,
		Name:        p.Name,
		Description: p.Description,
		Price:       formatPrice(*p, currency),
		ImageURL:    imageURL,
		Stock:       toProductStockInfo(p),
	}
	if p.Subname != "" {
		resp.Subname = p.Subname
	}
	if p.SellerName != "" {
		resp.SellerName = p.SellerName
	}
	if len(sizes) > 0 {
		resp.AvailableSizes = sizes
	}
	return resp
}

func parseAvailableSizes(raw string) []string {
	if raw == "" {
		return nil
	}
	var sizes []string
	if err := json.Unmarshal([]byte(raw), &sizes); err != nil {
		return nil
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

func (s *productService) deleteStoredImage(ctx context.Context, oldKey, newKey string) {
	if oldKey == "" || oldKey == newKey {
		return
	}
	if strings.HasPrefix(oldKey, "http://") || strings.HasPrefix(oldKey, "https://") {
		return
	}
	if s.storage == nil {
		return
	}
	_ = s.storage.Delete(ctx, oldKey)
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
