package service

import (
	"context"
	"strings"
	"time"

	cartdto "clap/internal/modules/cart/dto"
	cartservice "clap/internal/modules/cart/service"
	"clap/internal/modules/order/dto"
	"clap/internal/modules/order/models"
	orderrepo "clap/internal/modules/order/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/utils"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

const (
	shippingSeatCents   int64 = 400
	shippingPickupCents int64 = 350
	imageURLExpiry            = 6 * time.Hour
)

type OrderService interface {
	CreateFromCart(ctx context.Context, userID uuid.UUID, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error)
	Pay(ctx context.Context, userID, orderID uuid.UUID, req *dto.PayOrderRequest) (*dto.PayOrderResponse, error)
	GetCheckoutPreview(ctx context.Context, userID uuid.UUID, deliveryMethod string) (*dto.CheckoutPreviewResponse, error)
}

type orderService struct {
	orderRepo orderrepo.OrderRepository
	cartSvc   cartservice.CartService
	storage   storage.StorageProvider
}

func NewOrderService(
	orderRepo orderrepo.OrderRepository,
	cartSvc cartservice.CartService,
	storageProvider storage.StorageProvider,
) OrderService {
	return &orderService{
		orderRepo: orderRepo,
		cartSvc:   cartSvc,
		storage:   storageProvider,
	}
}

func (s *orderService) GetCheckoutPreview(ctx context.Context, userID uuid.UUID, deliveryMethod string) (*dto.CheckoutPreviewResponse, error) {
	details, err := s.cartSvc.ListItemDetails(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(details) == 0 {
		return nil, errors.NewUnprocessable("Cart is empty", nil)
	}

	method, err := parseDeliveryMethod(deliveryMethod)
	if err != nil {
		return nil, err
	}

	subtotal := calcSubtotal(details)
	shipping := shippingForMethod(method)
	total := subtotal + shipping

	items := make([]dto.CheckoutItemResponse, len(details))
	for i, item := range details {
		imageURL := s.resolveURL(ctx, item.ImageKey)
		formatted := cartservice.FormatCartItemDetail(item, imageURL)
		items[i] = dto.CheckoutItemResponse{
			ID:          formatted.ID,
			ProductID:   formatted.ProductID,
			Name:        formatted.Name,
			Description: formatted.Description,
			Price:       formatted.Price,
			ImageURL:    formatted.ImageURL,
			Quantity:    formatted.Quantity,
			Size:        formatted.Size,
		}
	}

	return &dto.CheckoutPreviewResponse{
		Items:    items,
		Subtotal: utils.FormatEuro(subtotal),
		Shipping: utils.FormatEuro(shipping),
		Total:    utils.FormatEuro(total),
	}, nil
}

func (s *orderService) CreateFromCart(ctx context.Context, userID uuid.UUID, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error) {
	details, err := s.cartSvc.ListItemDetails(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(details) == 0 {
		return nil, errors.NewUnprocessable("Cart is empty", nil)
	}

	method, err := parseDeliveryMethod(req.DeliveryMethod)
	if err != nil {
		return nil, err
	}

	seatNumber := strings.TrimSpace(req.SeatNumber)
	if method == models.DeliverySeat && seatNumber == "" {
		return nil, errors.NewBadRequest("seat_number is required for seat delivery", nil)
	}

	subtotal := calcSubtotal(details)
	shipping := shippingForMethod(method)
	total := subtotal + shipping

	order := &models.Order{
		UserID:         userID,
		Status:         models.StatusPendingPayment,
		DeliveryMethod: method,
		SeatNumber:     seatNumber,
		SubtotalCents:  subtotal,
		ShippingCents:  shipping,
		TotalCents:     total,
	}

	orderItems := make([]models.OrderItem, len(details))
	itemResponses := make([]dto.OrderItemResponse, len(details))
	for i, item := range details {
		orderItems[i] = models.OrderItem{
			ProductID:   item.ProductID,
			ProductType: item.ProductType,
			Name:        item.Name,
			Description: item.Description,
			PriceCents:  item.PriceCents,
			Quantity:    item.Quantity,
			Size:        item.Size,
			ImageKey:    item.ImageKey,
		}
		itemResponses[i] = dto.OrderItemResponse{
			ID:       item.ProductID,
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    utils.FormatEuro(item.PriceCents),
		}
	}

	if err := s.orderRepo.Create(ctx, order, orderItems); err != nil {
		return nil, err
	}

	if err := s.cartSvc.ClearCart(ctx, userID); err != nil {
		return nil, err
	}

	return &dto.CreateOrderResponse{
		OrderID:  order.ID,
		Items:    itemResponses,
		Subtotal: utils.FormatEuro(subtotal),
		Shipping: utils.FormatEuro(shipping),
		Total:    utils.FormatEuro(total),
		Status:   order.Status,
	}, nil
}

func (s *orderService) Pay(ctx context.Context, userID, orderID uuid.UUID, req *dto.PayOrderRequest) (*dto.PayOrderResponse, error) {
	if strings.TrimSpace(req.PaymentMethod) == "" {
		return nil, errors.NewBadRequest("payment_method is required", nil)
	}

	order, err := s.orderRepo.FindByID(ctx, orderID, userID)
	if err != nil {
		return nil, err
	}

	if order.Status != models.StatusPendingPayment {
		return nil, errors.NewBadRequest("Order is not pending payment", nil)
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, models.StatusPaid); err != nil {
		return nil, err
	}

	return &dto.PayOrderResponse{Status: models.StatusPaid}, nil
}

func parseDeliveryMethod(raw string) (string, error) {
	method := strings.ToLower(strings.TrimSpace(raw))
	if method != models.DeliverySeat && method != models.DeliveryPickup {
		return "", errors.NewBadRequest("delivery_method must be 'seat' or 'pickup'", nil)
	}
	return method, nil
}

func calcSubtotal(details []cartdto.CartItemWithProduct) int64 {
	var subtotal int64
	for _, item := range details {
		subtotal += item.PriceCents * int64(item.Quantity)
	}
	return subtotal
}

func shippingForMethod(method string) int64 {
	if method == models.DeliveryPickup {
		return shippingPickupCents
	}
	return shippingSeatCents
}

func (s *orderService) resolveURL(ctx context.Context, stored string) string {
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
