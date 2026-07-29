package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"clap/internal/modules/shop/dto"
	"clap/internal/modules/shop/models"
	"clap/internal/modules/shop/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

// Currency toggle values (contract §6.1 / §7.1).
const (
	CurrencyEUR   = "EUR"
	CurrencyPoint = "POINT"
)

// seatDeliveryShippingCents is the flat in-stadium delivery fee (4,00 €).
const seatDeliveryShippingCents = 400

// cartGroupPreviewSize is how many item thumbnails a cart group shows before
// collapsing into "N more item" (contract §6.3 example).
const cartGroupPreviewSize = 3

var validSnackCategories = map[string]bool{"sandwiches": true, "snacks": true, "drinks": true}
var validProductCategories = map[string]bool{"t-shirts": true, "balls": true, "stickers": true, "sport-suits": true}

// ShopService implements Snacks, Store, Cart and Orders (contract §6–§7).
type ShopService interface {
	ListSnacks(ctx context.Context, userID uuid.UUID, search, category, currency string, page, limit int) (*dto.CatalogListResponse, error)
	SnackDetail(ctx context.Context, snackID uuid.UUID, currency string) (*dto.CatalogItem, error)
	ListProducts(ctx context.Context, userID uuid.UUID, search, category, currency string, page, limit int) (*dto.CatalogListResponse, error)
	ProductDetail(ctx context.Context, productID uuid.UUID, currency string) (*dto.ProductDetailResponse, error)

	GetCart(ctx context.Context, userID uuid.UUID) (*dto.CartResponse, error)
	AddCartItem(ctx context.Context, userID uuid.UUID, req *dto.AddCartItemRequest) (*dto.CartResponse, error)
	UpdateCartItem(ctx context.Context, userID, itemID uuid.UUID, quantity int) (*dto.CartResponse, error)
	RemoveCartItem(ctx context.Context, userID, itemID uuid.UUID) error
	CartCount(ctx context.Context, userID uuid.UUID) (int, error)

	Checkout(ctx context.Context, userID uuid.UUID, req *dto.CheckoutRequest) (*dto.OrderResponse, error)
	Pay(ctx context.Context, userID, orderID uuid.UUID, paymentMethod string) (*dto.PayOrderResponse, error)

	// SnacksPreview powers the Home screen foods card (contract §3.1).
	SnacksPreview(ctx context.Context, limit int) ([]dto.CatalogItem, error)
	// ProductsPreview powers the Home club store card (contract §3.2).
	ProductsPreview(ctx context.Context, limit int) ([]dto.CatalogItem, error)
}

type shopService struct {
	snackRepo   repository.SnackRepository
	productRepo repository.ProductRepository
	cartRepo    repository.CartRepository
	orderRepo   repository.OrderRepository
}

func NewShopService(
	snackRepo repository.SnackRepository,
	productRepo repository.ProductRepository,
	cartRepo repository.CartRepository,
	orderRepo repository.OrderRepository,
) ShopService {
	return &shopService{
		snackRepo:   snackRepo,
		productRepo: productRepo,
		cartRepo:    cartRepo,
		orderRepo:   orderRepo,
	}
}

// ─── Catalog ─────────────────────────────────────────────────────────────────

func (s *shopService) ListSnacks(ctx context.Context, userID uuid.UUID, search, category, currency string, page, limit int) (*dto.CatalogListResponse, error) {
	if category != "" && !validSnackCategories[category] {
		return nil, errors.NewBadRequest("Invalid category", nil)
	}
	if err := validateCurrency(currency); err != nil {
		return nil, err
	}

	snacks, total, err := s.snackRepo.FindAll(ctx, search, category, page, limit)
	if err != nil {
		return nil, err
	}

	items := make([]dto.CatalogItem, len(snacks))
	for i, snack := range snacks {
		items[i] = snackToItem(&snack, currency)
	}

	cartCount, err := s.cartRepo.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.CatalogListResponse{
		Items:     items,
		CartCount: cartCount,
		Meta:      utils.NewListMeta(total, page, limit),
	}, nil
}

func (s *shopService) SnackDetail(ctx context.Context, snackID uuid.UUID, currency string) (*dto.CatalogItem, error) {
	if err := validateCurrency(currency); err != nil {
		return nil, err
	}
	snack, err := s.snackRepo.FindByID(ctx, snackID)
	if err != nil {
		return nil, err
	}
	item := snackToItem(snack, currency)
	return &item, nil
}

func (s *shopService) ListProducts(ctx context.Context, userID uuid.UUID, search, category, currency string, page, limit int) (*dto.CatalogListResponse, error) {
	if category != "" && !validProductCategories[category] {
		return nil, errors.NewBadRequest("Invalid category", nil)
	}
	if err := validateCurrency(currency); err != nil {
		return nil, err
	}

	products, total, err := s.productRepo.FindAll(ctx, search, category, page, limit)
	if err != nil {
		return nil, err
	}

	items := make([]dto.CatalogItem, len(products))
	for i, product := range products {
		items[i] = productToItem(&product, currency)
	}

	cartCount, err := s.cartRepo.CountByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.CatalogListResponse{
		Items:     items,
		CartCount: cartCount,
		Meta:      utils.NewListMeta(total, page, limit),
	}, nil
}

func (s *shopService) ProductDetail(ctx context.Context, productID uuid.UUID, currency string) (*dto.ProductDetailResponse, error) {
	if err := validateCurrency(currency); err != nil {
		return nil, err
	}
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	return &dto.ProductDetailResponse{
		ID:             product.ID,
		Name:           product.Name,
		SellerName:     product.SellerName,
		Description:    product.Description,
		Price:          formatPrice(product.PriceCents, product.PointsPrice, currency),
		ImageURL:       product.ImageURL,
		AvailableSizes: parseSizes(product.Sizes),
	}, nil
}

// ─── Cart ────────────────────────────────────────────────────────────────────

func (s *shopService) GetCart(ctx context.Context, userID uuid.UUID) (*dto.CartResponse, error) {
	items, err := s.cartRepo.ItemsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.buildCartResponse(ctx, items)
}

func (s *shopService) AddCartItem(ctx context.Context, userID uuid.UUID, req *dto.AddCartItemRequest) (*dto.CartResponse, error) {
	size := strings.TrimSpace(req.Size)

	// Validate the referenced product and (for merch) the size.
	switch req.ProductType {
	case models.ProductTypeSnack:
		if size != "" {
			return nil, errors.NewBadRequest("Snacks do not have sizes", nil)
		}
		if _, err := s.snackRepo.FindByID(ctx, req.ProductID); err != nil {
			return nil, err
		}
	case models.ProductTypeMerch:
		product, err := s.productRepo.FindByID(ctx, req.ProductID)
		if err != nil {
			return nil, err
		}
		if size != "" && !containsFold(parseSizes(product.Sizes), size) {
			return nil, errors.NewBadRequest("Size is not available for this product", nil)
		}
	}

	item := &models.CartItem{
		UserID:      userID,
		ProductType: req.ProductType,
		ProductID:   req.ProductID,
		Quantity:    req.Quantity,
		Size:        size,
	}
	if err := s.cartRepo.Upsert(ctx, item); err != nil {
		return nil, err
	}

	logger.Info().
		Str("user_id", userID.String()).
		Str("product_type", req.ProductType).
		Str("product_id", req.ProductID.String()).
		Int("quantity", req.Quantity).
		Msg("cart_item_added")

	return s.GetCart(ctx, userID)
}

func (s *shopService) UpdateCartItem(ctx context.Context, userID, itemID uuid.UUID, quantity int) (*dto.CartResponse, error) {
	if _, err := s.cartRepo.UpdateQuantity(ctx, itemID, userID, quantity); err != nil {
		return nil, err
	}

	logger.Info().
		Str("user_id", userID.String()).
		Str("cart_item_id", itemID.String()).
		Int("quantity", quantity).
		Msg("cart_item_updated")

	return s.GetCart(ctx, userID)
}

func (s *shopService) RemoveCartItem(ctx context.Context, userID, itemID uuid.UUID) error {
	if err := s.cartRepo.Delete(ctx, itemID, userID); err != nil {
		return err
	}

	logger.Info().
		Str("user_id", userID.String()).
		Str("cart_item_id", itemID.String()).
		Msg("cart_item_removed")

	return nil
}

func (s *shopService) CartCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.cartRepo.CountByUser(ctx, userID)
}

// ─── Orders ──────────────────────────────────────────────────────────────────

func (s *shopService) Checkout(ctx context.Context, userID uuid.UUID, req *dto.CheckoutRequest) (*dto.OrderResponse, error) {
	if req.DeliveryMethod == "seat" && strings.TrimSpace(req.SeatNumber) == "" {
		return nil, errors.NewBadRequest("seat_number is required for seat delivery", nil)
	}

	cartItems, err := s.cartRepo.ItemsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(cartItems) == 0 {
		return nil, errors.NewUnprocessable("Cart is empty", nil)
	}

	snacks, products, err := s.loadCartProducts(ctx, cartItems)
	if err != nil {
		return nil, err
	}

	var subtotal int64
	orderItems := make([]models.OrderItem, 0, len(cartItems))
	for _, ci := range cartItems {
		var name, imageURL string
		var priceCents int64
		switch ci.ProductType {
		case models.ProductTypeSnack:
			snack, ok := snacks[ci.ProductID]
			if !ok {
				return nil, errors.NewUnprocessable("A cart item is no longer available", nil)
			}
			name, imageURL, priceCents = snack.Name, snack.ImageURL, snack.PriceCents
		case models.ProductTypeMerch:
			product, ok := products[ci.ProductID]
			if !ok {
				return nil, errors.NewUnprocessable("A cart item is no longer available", nil)
			}
			name, imageURL, priceCents = product.Name, product.ImageURL, product.PriceCents
		}

		subtotal += priceCents * int64(ci.Quantity)
		orderItems = append(orderItems, models.OrderItem{
			ProductType: ci.ProductType,
			ProductID:   ci.ProductID,
			Name:        name,
			PriceCents:  priceCents,
			Quantity:    ci.Quantity,
			Size:        ci.Size,
			ImageURL:    imageURL,
		})
	}

	var shipping int64
	if req.DeliveryMethod == "seat" {
		shipping = seatDeliveryShippingCents
	}

	order := &models.Order{
		UserID:         userID,
		DeliveryMethod: req.DeliveryMethod,
		SeatNumber:     strings.TrimSpace(req.SeatNumber),
		SubtotalCents:  subtotal,
		ShippingCents:  shipping,
		TotalCents:     subtotal + shipping,
		Status:         models.OrderStatusPendingPayment,
	}

	if err := s.orderRepo.CreateFromCart(ctx, order, orderItems); err != nil {
		return nil, err
	}

	logger.Info().
		Str("user_id", userID.String()).
		Str("order_id", order.ID.String()).
		Int64("total_cents", order.TotalCents).
		Str("delivery_method", order.DeliveryMethod).
		Msg("order_created")

	return orderToResponse(order, orderItems), nil
}

func (s *shopService) Pay(ctx context.Context, userID, orderID uuid.UUID, paymentMethod string) (*dto.PayOrderResponse, error) {
	if err := s.orderRepo.MarkPaid(ctx, orderID, userID, paymentMethod); err != nil {
		return nil, err
	}

	logger.Info().
		Str("user_id", userID.String()).
		Str("order_id", orderID.String()).
		Str("payment_method", paymentMethod).
		Msg("order_paid")

	return &dto.PayOrderResponse{Status: models.OrderStatusPaid}, nil
}

// ─── Home previews ───────────────────────────────────────────────────────────

func (s *shopService) SnacksPreview(ctx context.Context, limit int) ([]dto.CatalogItem, error) {
	snacks, err := s.snackRepo.FindPreview(ctx, limit)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CatalogItem, len(snacks))
	for i, snack := range snacks {
		items[i] = snackToItem(&snack, CurrencyEUR)
	}
	return items, nil
}

func (s *shopService) ProductsPreview(ctx context.Context, limit int) ([]dto.CatalogItem, error) {
	products, err := s.productRepo.FindPreview(ctx, limit)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CatalogItem, len(products))
	for i, product := range products {
		items[i] = productToItem(&product, CurrencyEUR)
	}
	return items, nil
}

// ─── internals ────────────────────────────────────────────────────────────────

func (s *shopService) loadCartProducts(ctx context.Context, items []models.CartItem) (map[uuid.UUID]models.Snack, map[uuid.UUID]models.Product, error) {
	var snackIDs, productIDs []uuid.UUID
	for _, ci := range items {
		if ci.ProductType == models.ProductTypeSnack {
			snackIDs = append(snackIDs, ci.ProductID)
		} else {
			productIDs = append(productIDs, ci.ProductID)
		}
	}

	snacks, err := s.snackRepo.FindByIDs(ctx, snackIDs)
	if err != nil {
		return nil, nil, err
	}
	products, err := s.productRepo.FindByIDs(ctx, productIDs)
	if err != nil {
		return nil, nil, err
	}
	return snacks, products, nil
}

func (s *shopService) buildCartResponse(ctx context.Context, items []models.CartItem) (*dto.CartResponse, error) {
	snacks, products, err := s.loadCartProducts(ctx, items)
	if err != nil {
		return nil, err
	}

	type groupDef struct {
		id    string
		title string
	}
	groups := map[string]groupDef{
		models.ProductTypeSnack: {id: "food", title: "Food Delivery"},
		models.ProductTypeMerch: {id: "store", title: "Club Store"},
	}

	byType := map[string][]models.CartItem{}
	for _, ci := range items {
		byType[ci.ProductType] = append(byType[ci.ProductType], ci)
	}

	orders := make([]dto.CartGroup, 0, len(byType))
	for _, productType := range []string{models.ProductTypeSnack, models.ProductTypeMerch} {
		groupItems := byType[productType]
		if len(groupItems) == 0 {
			continue
		}

		views := make([]dto.CartItemView, 0, cartGroupPreviewSize)
		date := groupItems[0].CreatedAt
		for i, ci := range groupItems {
			if ci.CreatedAt.After(date) {
				date = ci.CreatedAt
			}
			if i >= cartGroupPreviewSize {
				continue
			}
			imageURL := ""
			switch ci.ProductType {
			case models.ProductTypeSnack:
				if snack, ok := snacks[ci.ProductID]; ok {
					imageURL = snack.ImageURL
				}
			case models.ProductTypeMerch:
				if product, ok := products[ci.ProductID]; ok {
					imageURL = product.ImageURL
				}
			}
			views = append(views, dto.CartItemView{ID: ci.ID, ImageURL: imageURL, Quantity: ci.Quantity})
		}

		extraText := ""
		if extra := len(groupItems) - cartGroupPreviewSize; extra > 0 {
			extraText = fmt.Sprintf("%d more item", extra)
		}

		def := groups[productType]
		orders = append(orders, dto.CartGroup{
			ID:        def.id,
			Title:     def.title,
			Date:      date.Format("2006-01-02"),
			Items:     views,
			ExtraText: extraText,
		})
	}

	return &dto.CartResponse{Orders: orders}, nil
}

func orderToResponse(order *models.Order, items []models.OrderItem) *dto.OrderResponse {
	views := make([]dto.OrderItemView, len(items))
	for i, item := range items {
		views[i] = dto.OrderItemView{
			ID:       item.ProductID,
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    utils.FormatEuro(item.PriceCents),
		}
	}
	return &dto.OrderResponse{
		OrderID:  order.ID,
		Items:    views,
		Subtotal: utils.FormatEuro(order.SubtotalCents),
		Shipping: utils.FormatEuro(order.ShippingCents),
		Total:    utils.FormatEuro(order.TotalCents),
		Status:   order.Status,
	}
}

func snackToItem(snack *models.Snack, currency string) dto.CatalogItem {
	return dto.CatalogItem{
		ID:          snack.ID,
		Name:        snack.Name,
		Description: snack.Description,
		Price:       formatPrice(snack.PriceCents, snack.PointsPrice, currency),
		ImageURL:    snack.ImageURL,
	}
}

func productToItem(product *models.Product, currency string) dto.CatalogItem {
	return dto.CatalogItem{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       formatPrice(product.PriceCents, product.PointsPrice, currency),
		ImageURL:    product.ImageURL,
	}
}

func formatPrice(priceCents int64, pointsPrice int, currency string) string {
	if currency == CurrencyPoint {
		return utils.FormatPoints(pointsPrice)
	}
	return utils.FormatEuro(priceCents)
}

func validateCurrency(currency string) error {
	if currency != "" && currency != CurrencyEUR && currency != CurrencyPoint {
		return errors.NewBadRequest("currency must be EUR or POINT", nil)
	}
	return nil
}

func parseSizes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var sizes []string
	if err := json.Unmarshal([]byte(raw), &sizes); err != nil {
		return []string{}
	}
	return sizes
}

func containsFold(list []string, target string) bool {
	for _, v := range list {
		if strings.EqualFold(v, target) {
			return true
		}
	}
	return false
}
