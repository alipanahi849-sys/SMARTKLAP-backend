package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/order/dto"
	"clap/internal/modules/order/models"
	orderrepo "clap/internal/modules/order/repository"
	shopdto "clap/internal/modules/shop/dto"
	shopmodels "clap/internal/modules/shop/models"
	shoprepo "clap/internal/modules/shop/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/utils"
	"clap/pkg/payment"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

const (
	orderListImageLimit = 3
	imageURLExpiry      = 6 * time.Hour
)

// OrderService manages checkout and payment for shop orders.
type OrderService interface {
	ListOrders(ctx context.Context, userID uuid.UUID, filters dto.OrderListFilters) (*dto.OrderListResponse, error)
	GetOrder(ctx context.Context, userID, orderID uuid.UUID) (*dto.OrderDetailResponse, error)
	CalculateOrder(ctx context.Context, userID uuid.UUID, req *dto.CalculateOrderRequest) (*dto.CalculateOrderResponse, error)
	CreateOrder(ctx context.Context, userID uuid.UUID, req *dto.CreateOrderRequest) (*dto.OrderResponse, error)
	PayOrder(ctx context.Context, userID, orderID uuid.UUID, req *dto.PayOrderRequest) (*dto.PayOrderResponse, error)
	ConfirmCardPayment(ctx context.Context, userID, orderID uuid.UUID) (*dto.PayOrderResponse, error)
	HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error
}

type orderService struct {
	orderRepo       orderrepo.OrderRepository
	cartRepo        shoprepo.CartRepository
	productRepo     shoprepo.ProductRepository
	sizeStockRepo   shoprepo.ProductSizeStockRepository
	userRepo        authrepo.UserRepository
	storage         storage.StorageProvider
	paymentProvider payment.Provider
	appURLScheme    string
}

func NewOrderService(
	orderRepo orderrepo.OrderRepository,
	cartRepo shoprepo.CartRepository,
	productRepo shoprepo.ProductRepository,
	sizeStockRepo shoprepo.ProductSizeStockRepository,
	userRepo authrepo.UserRepository,
	storageProvider storage.StorageProvider,
	paymentProvider payment.Provider,
	appURLScheme string,
) OrderService {
	scheme := strings.TrimSpace(appURLScheme)
	if scheme == "" {
		scheme = "smartklap"
	}
	return &orderService{
		orderRepo:       orderRepo,
		cartRepo:        cartRepo,
		productRepo:     productRepo,
		sizeStockRepo:   sizeStockRepo,
		userRepo:        userRepo,
		storage:         storageProvider,
		paymentProvider: paymentProvider,
		appURLScheme:    scheme,
	}
}

type orderTotals struct {
	SubtotalCents  int64
	ShippingCents  int64
	TotalCents     int64
	SubtotalPoints int
	ShippingPoints int
	TotalPoints    int
}

func (s *orderService) ListOrders(ctx context.Context, userID uuid.UUID, filters dto.OrderListFilters) (*dto.OrderListResponse, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}

	var after *orderrepo.OrderCursorAnchor
	if filters.Cursor != nil {
		cursorOrder, err := s.orderRepo.FindByIDForUser(ctx, userID, *filters.Cursor)
		if err != nil {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		after = &orderrepo.OrderCursorAnchor{
			CreatedAt: cursorOrder.CreatedAt,
			ID:        cursorOrder.ID,
		}
	}

	orders, err := s.orderRepo.ListForUserAfter(ctx, userID, limit+1, after)
	if err != nil {
		return nil, err
	}

	hasMore := len(orders) > limit
	if hasMore {
		orders = orders[:limit]
	}

	imageKeys, err := s.loadOrderItemImageKeys(ctx, orders)
	if err != nil {
		return nil, err
	}

	items := make([]dto.OrderListItem, len(orders))
	for i := range orders {
		items[i] = toOrderListItem(ctx, &orders[i], imageKeys, s.resolveURL)
	}

	meta := dto.CursorListMeta{
		Limit:   limit,
		HasMore: hasMore,
	}
	if hasMore && len(orders) > 0 {
		lastID := orders[len(orders)-1].ID
		meta.NextCursor = &lastID
	}

	return &dto.OrderListResponse{
		Items: items,
		Meta:  meta,
	}, nil
}

func (s *orderService) GetOrder(ctx context.Context, userID, orderID uuid.UUID) (*dto.OrderDetailResponse, error) {
	order, err := s.orderRepo.FindByIDForUser(ctx, userID, orderID)
	if err != nil {
		return nil, err
	}

	imageKeys, err := s.loadOrderItemImageKeys(ctx, []models.Order{*order})
	if err != nil {
		return nil, err
	}

	return toOrderDetailResponse(ctx, order, imageKeys, s.resolveURL), nil
}

func (s *orderService) CalculateOrder(ctx context.Context, userID uuid.UUID, req *dto.CalculateOrderRequest) (*dto.CalculateOrderResponse, error) {
	deliveryMethod := strings.ToLower(strings.TrimSpace(req.DeliveryMethod))
	if deliveryMethod != models.DeliveryMethodSeat && deliveryMethod != models.DeliveryMethodPickup {
		return nil, errors.NewBadRequest("delivery_method must be seat or pickup", nil)
	}

	paymentMethod := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	if paymentMethod != models.PaymentMethodPoints && paymentMethod != models.PaymentMethodCard {
		return nil, errors.NewBadRequest("payment_method must be points or card", nil)
	}

	totals, _, err := s.loadCartTotals(ctx, userID, deliveryMethod)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	displayCurrency := shopdto.CurrencyEUR
	if paymentMethod == models.PaymentMethodPoints {
		displayCurrency = shopdto.CurrencyPoint
	}

	resp := &dto.CalculateOrderResponse{
		DeliveryMethod:   deliveryMethod,
		PaymentMethod:    paymentMethod,
		Subtotal:         formatOrderAmount(totals.SubtotalCents, totals.SubtotalPoints, displayCurrency),
		Total:            formatOrderAmount(totals.TotalCents, totals.TotalPoints, displayCurrency),
		PaymentAmount:    formatOrderAmount(totals.TotalCents, totals.TotalPoints, displayCurrency),
		PointsRequired:   totals.TotalPoints,
		UserPoints:       user.Points,
		SufficientPoints: user.Points >= totals.TotalPoints,
	}
	if totals.ShippingCents > 0 || totals.ShippingPoints > 0 {
		resp.Shipping = formatOrderAmount(totals.ShippingCents, totals.ShippingPoints, displayCurrency)
	}
	return resp, nil
}

func (s *orderService) CreateOrder(ctx context.Context, userID uuid.UUID, req *dto.CreateOrderRequest) (*dto.OrderResponse, error) {
	deliveryMethod := strings.ToLower(strings.TrimSpace(req.DeliveryMethod))
	if deliveryMethod != models.DeliveryMethodSeat && deliveryMethod != models.DeliveryMethodPickup {
		return nil, errors.NewBadRequest("delivery_method must be seat or pickup", nil)
	}

	currency, err := parseOrderCurrency(req.Currency)
	if err != nil {
		return nil, err
	}

	seatNumber := ""
	if req.SeatNumber != nil {
		seatNumber = strings.TrimSpace(*req.SeatNumber)
	}
	if deliveryMethod == models.DeliveryMethodSeat && seatNumber == "" {
		return nil, errors.NewBadRequest("seat_number is required for seat delivery", nil)
	}

	totals, lines, err := s.loadCartTotals(ctx, userID, deliveryMethod)
	if err != nil {
		return nil, err
	}

	orderItems := make([]models.OrderItem, len(lines))
	for i, line := range lines {
		var subname *string
		if trimmed := strings.TrimSpace(line.Subname); trimmed != "" {
			subname = &trimmed
		}
		orderItems[i] = models.OrderItem{
			ProductID:   line.ProductID,
			ProductType: line.ProductType,
			Size:        line.Size,
			Name:        line.Name,
			Subname:     subname,
			PriceCents:  line.PriceCents,
			PricePoints: line.PricePoints,
			Quantity:    line.Quantity,
		}
	}

	order := &models.Order{
		UserID:         userID,
		Status:         models.OrderStatusPendingPayment,
		DeliveryMethod: deliveryMethod,
		SubtotalCents:  totals.SubtotalCents,
		ShippingCents:  totals.ShippingCents,
		TotalCents:     totals.TotalCents,
		SubtotalPoints: totals.SubtotalPoints,
		ShippingPoints: totals.ShippingPoints,
		TotalPoints:    totals.TotalPoints,
	}
	if seatNumber != "" {
		order.SeatNumber = &seatNumber
	}

	if currency == shopdto.CurrencyPoint {
		if err := s.ensureSufficientPoints(ctx, userID, order.TotalPoints); err != nil {
			return nil, err
		}
	}

	if err := s.orderRepo.CreateWithItems(ctx, order, orderItems); err != nil {
		return nil, err
	}
	if err := s.cartRepo.ClearUserCart(ctx, userID); err != nil {
		return nil, err
	}

	return toOrderResponse(order, currency), nil
}

func (s *orderService) loadCartTotals(ctx context.Context, userID uuid.UUID, deliveryMethod string) (orderTotals, []shoprepo.UserCartLine, error) {
	lines, err := s.cartRepo.ListUserLines(ctx, userID)
	if err != nil {
		return orderTotals{}, nil, err
	}
	if len(lines) == 0 {
		return orderTotals{}, nil, errors.NewUnprocessable("Cart is empty", nil)
	}

	subtotalCents, err := s.cartRepo.SumSubtotalCents(ctx, userID)
	if err != nil {
		return orderTotals{}, nil, err
	}

	subtotalPoints, err := s.cartRepo.SumSubtotalPoints(ctx, userID)
	if err != nil {
		return orderTotals{}, nil, err
	}

	shippingCents, shippingPoints := shippingForDelivery(deliveryMethod, lines)
	return orderTotals{
		SubtotalCents:  subtotalCents,
		ShippingCents:  shippingCents,
		TotalCents:     subtotalCents + shippingCents,
		SubtotalPoints: subtotalPoints,
		ShippingPoints: shippingPoints,
		TotalPoints:    subtotalPoints + shippingPoints,
	}, lines, nil
}

func (s *orderService) PayOrder(ctx context.Context, userID, orderID uuid.UUID, req *dto.PayOrderRequest) (*dto.PayOrderResponse, error) {
	method := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
	switch method {
	case models.PaymentMethodPoints:
		return s.payWithPoints(ctx, userID, orderID)
	case models.PaymentMethodCard:
		return s.payWithCard(ctx, userID, orderID)
	default:
		return nil, errors.NewBadRequest("Unsupported payment_method", nil)
	}
}

func (s *orderService) ensureSufficientPoints(ctx context.Context, userID uuid.UUID, required int) error {
	if required <= 0 {
		return nil
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.Points < required {
		return errors.NewUnprocessable("Insufficient points balance", nil)
	}
	return nil
}

func (s *orderService) payWithPoints(ctx context.Context, userID, orderID uuid.UUID) (*dto.PayOrderResponse, error) {
	order, err := s.orderRepo.FindByIDForUser(ctx, userID, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status == models.OrderStatusPaid {
		return &dto.PayOrderResponse{Status: models.OrderStatusPaid}, nil
	}
	if order.Status != models.OrderStatusPendingPayment {
		return nil, errors.NewUnprocessable("Order is not payable", nil)
	}

	if err := s.ensureSufficientPoints(ctx, userID, order.TotalPoints); err != nil {
		return nil, err
	}

	if order.TotalPoints <= 0 {
		if err := s.fulfillPaidOrder(ctx, order.ID, models.PaymentMethodPoints); err != nil {
			return nil, err
		}
		return &dto.PayOrderResponse{Status: models.OrderStatusPaid}, nil
	}

	remaining, err := s.userRepo.SpendPoints(ctx, userID, order.TotalPoints)
	if err != nil {
		return nil, err
	}

	if err := s.fulfillPaidOrder(ctx, order.ID, models.PaymentMethodPoints); err != nil {
		if _, refundErr := s.userRepo.AddPoints(ctx, userID, order.TotalPoints); refundErr != nil {
			return nil, errors.NewInternal("Payment failed and points refund failed", err)
		}
		return nil, err
	}

	return &dto.PayOrderResponse{
		Status:          models.OrderStatusPaid,
		PointsRemaining: &remaining,
	}, nil
}

func (s *orderService) payWithCard(ctx context.Context, userID, orderID uuid.UUID) (*dto.PayOrderResponse, error) {
	if s.paymentProvider == nil || !s.paymentProvider.Enabled() {
		return nil, errors.NewUnprocessable("Card payments are not configured", nil)
	}

	order, err := s.orderRepo.FindByIDForUser(ctx, userID, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status == models.OrderStatusPaid {
		return &dto.PayOrderResponse{Status: models.OrderStatusPaid}, nil
	}
	if order.Status != models.OrderStatusPendingPayment {
		return nil, errors.NewUnprocessable("Order is not payable", nil)
	}
	if order.TotalCents <= 0 {
		return nil, errors.NewUnprocessable("Order total must be greater than zero for card payment", nil)
	}

	successURL := fmt.Sprintf("%s://checkout/success?order_id=%s", s.appURLScheme, order.ID)
	cancelURL := fmt.Sprintf("%s://checkout/cancel?order_id=%s", s.appURLScheme, order.ID)

	customerEmail := ""
	if user, userErr := s.userRepo.FindByID(ctx, userID); userErr == nil && user != nil {
		customerEmail = strings.TrimSpace(user.Email)
	}

	checkoutSession, err := s.paymentProvider.CreateCheckoutSession(ctx, payment.CreateCheckoutParams{
		OrderID:       order.ID,
		UserID:        userID,
		AmountCents:   order.TotalCents,
		Currency:      shopdto.CurrencyEUR,
		SuccessURL:    successURL,
		CancelURL:     cancelURL,
		CustomerEmail: customerEmail,
	})
	if err != nil {
		return nil, errors.NewInternal("Failed to create checkout session", err)
	}

	if err := s.orderRepo.UpdateStripePaymentIntentID(ctx, order.ID, checkoutSession.SessionID); err != nil {
		return nil, err
	}

	return &dto.PayOrderResponse{
		Status:      models.OrderStatusPendingPayment,
		CheckoutURL: &checkoutSession.CheckoutURL,
	}, nil
}

func (s *orderService) ConfirmCardPayment(ctx context.Context, userID, orderID uuid.UUID) (*dto.PayOrderResponse, error) {
	if s.paymentProvider == nil || !s.paymentProvider.Enabled() {
		return nil, errors.NewUnprocessable("Card payments are not configured", nil)
	}

	order, err := s.orderRepo.FindByIDForUser(ctx, userID, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status == models.OrderStatusPaid {
		return &dto.PayOrderResponse{Status: models.OrderStatusPaid}, nil
	}
	if order.Status != models.OrderStatusPendingPayment {
		return nil, errors.NewUnprocessable("Order is not pending payment", nil)
	}
	if order.StripePaymentIntentID == nil || strings.TrimSpace(*order.StripePaymentIntentID) == "" {
		return nil, errors.NewUnprocessable("No checkout session for order", nil)
	}

	sessionID := strings.TrimSpace(*order.StripePaymentIntentID)
	if !strings.HasPrefix(sessionID, "cs_") {
		return nil, errors.NewUnprocessable("Order has no browser checkout session", nil)
	}

	sessionStatus, err := s.paymentProvider.GetCheckoutSession(ctx, sessionID)
	if err != nil {
		return nil, errors.NewInternal("Failed to verify checkout session", err)
	}
	if sessionStatus.OrderID != order.ID {
		return nil, errors.NewUnprocessable("Checkout session does not match order", nil)
	}

	if sessionStatus.PaymentStatus != "paid" {
		return &dto.PayOrderResponse{Status: models.OrderStatusPendingPayment}, nil
	}

	if err := s.fulfillPaidOrder(ctx, order.ID, models.PaymentMethodCard); err != nil {
		updated, reloadErr := s.orderRepo.FindByID(ctx, order.ID)
		if reloadErr == nil && updated.Status == models.OrderStatusPaid {
			s.emailOrderInvoice(ctx, sessionID)
			return &dto.PayOrderResponse{Status: models.OrderStatusPaid}, nil
		}
		return nil, err
	}

	s.emailOrderInvoice(ctx, sessionID)
	return &dto.PayOrderResponse{Status: models.OrderStatusPaid}, nil
}

func (s *orderService) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	if s.paymentProvider == nil || !s.paymentProvider.Enabled() {
		return errors.NewUnprocessable("Card payments are not configured", nil)
	}

	event, err := s.paymentProvider.ParseWebhookEvent(payload, signature)
	if err != nil {
		return errors.NewBadRequest("Invalid Stripe webhook", err)
	}
	if event == nil {
		return nil
	}

	orderID := event.OrderID
	recorded, err := s.orderRepo.RecordStripeEvent(ctx, event.ID, event.Type, &orderID)
	if err != nil {
		return err
	}
	if !recorded {
		return nil
	}

	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status == models.OrderStatusPaid {
		s.emailOrderInvoice(ctx, sessionIDFromOrder(order))
		return nil
	}
	if !event.Succeeded {
		return nil
	}

	if err := s.fulfillPaidOrder(ctx, order.ID, models.PaymentMethodCard); err != nil {
		return err
	}
	s.emailOrderInvoice(ctx, sessionIDFromOrder(order))
	return nil
}

func (s *orderService) emailOrderInvoice(ctx context.Context, sessionID string) {
	if s.paymentProvider == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	_ = s.paymentProvider.EmailCheckoutInvoice(ctx, sessionID)
}

func sessionIDFromOrder(order *models.Order) string {
	if order == nil || order.StripePaymentIntentID == nil {
		return ""
	}
	return strings.TrimSpace(*order.StripePaymentIntentID)
}

func (s *orderService) fulfillPaidOrder(ctx context.Context, orderID uuid.UUID, paymentMethod string) error {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status == models.OrderStatusPaid {
		return nil
	}
	if order.Status != models.OrderStatusPendingPayment {
		return errors.NewUnprocessable("Order is not pending payment", nil)
	}

	for _, item := range order.Items {
		if err := s.decrementStock(ctx, item); err != nil {
			return err
		}
	}

	return s.orderRepo.MarkPaid(ctx, orderID, time.Now().UTC(), paymentMethod)
}

func (s *orderService) decrementStock(ctx context.Context, item models.OrderItem) error {
	sizeStocks, err := s.sizeStockRepo.ListByProductID(ctx, item.ProductID)
	if err != nil {
		return err
	}

	if len(sizeStocks) > 0 || strings.TrimSpace(item.Size) != "" {
		return s.sizeStockRepo.DecrementForOrder(ctx, item.ProductID, item.Size, item.Quantity)
	}

	product, err := s.productRepo.FindByID(ctx, item.ProductID)
	if err != nil {
		return err
	}
	if product.IsUnlimitedStock() {
		return nil
	}
	return s.productRepo.DecrementStockForOrder(ctx, item.ProductID, item.Quantity)
}

func shippingForDelivery(method string, lines []shoprepo.UserCartLine) (cents int64, points int) {
	if method == models.DeliveryMethodPickup {
		return models.PickupShippingCents, models.PickupShippingPoints
	}
	if hasMerch(lines) {
		return models.SeatDeliveryShippingCents, models.SeatDeliveryShippingPoints
	}
	return 0, 0
}

func hasMerch(lines []shoprepo.UserCartLine) bool {
	for _, line := range lines {
		if line.ProductType == shopmodels.ProductTypeMerch {
			return true
		}
	}
	return false
}

func parseOrderCurrency(raw string) (string, error) {
	currency := strings.ToUpper(strings.TrimSpace(raw))
	if currency == "" {
		return shopdto.CurrencyEUR, nil
	}
	if currency != shopdto.CurrencyEUR && currency != shopdto.CurrencyPoint {
		return "", errors.NewBadRequest("currency must be EUR or POINT", nil)
	}
	return currency, nil
}

func toOrderResponse(order *models.Order, currency string) *dto.OrderResponse {
	items := make([]dto.OrderLineItem, len(order.Items))
	for i, item := range order.Items {
		items[i] = dto.OrderLineItem{
			ID:       item.ID,
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    formatOrderLinePrice(item, currency),
		}
	}

	resp := &dto.OrderResponse{
		OrderID:  order.ID,
		Items:    items,
		Subtotal: formatOrderAmount(order.SubtotalCents, order.SubtotalPoints, currency),
		Total:    formatOrderAmount(order.TotalCents, order.TotalPoints, currency),
		Status:   order.Status,
	}
	if order.ShippingCents > 0 || order.ShippingPoints > 0 {
		resp.Shipping = formatOrderAmount(order.ShippingCents, order.ShippingPoints, currency)
	}
	return resp
}

func toOrderListItem(ctx context.Context, order *models.Order, imageKeys map[uuid.UUID]string, resolve func(context.Context, string) string) dto.OrderListItem {
	currency := listDisplayCurrency(order)
	previewItems := orderPreviewItems(ctx, order, imageKeys, resolve)
	item := dto.OrderListItem{
		OrderID:        order.ID,
		Status:         order.Status,
		DeliveryMethod: order.DeliveryMethod,
		Subtotal:       formatOrderAmount(order.SubtotalCents, order.SubtotalPoints, currency),
		Total:          formatOrderAmount(order.TotalCents, order.TotalPoints, currency),
		ItemCount:      orderTotalQuantity(order.Items),
		Items:          previewItems,
		CreatedAt:      order.CreatedAt.UTC().Format(time.RFC3339),
	}
	if order.SeatNumber != nil {
		item.SeatNumber = strings.TrimSpace(*order.SeatNumber)
	}
	if order.PaymentMethod != nil {
		item.PaymentMethod = strings.TrimSpace(*order.PaymentMethod)
	}
	if order.ShippingCents > 0 || order.ShippingPoints > 0 {
		item.Shipping = formatOrderAmount(order.ShippingCents, order.ShippingPoints, currency)
	}
	return item
}

func toOrderDetailResponse(ctx context.Context, order *models.Order, imageKeys map[uuid.UUID]string, resolve func(context.Context, string) string) *dto.OrderDetailResponse {
	currency := listDisplayCurrency(order)
	resp := &dto.OrderDetailResponse{
		OrderID:        order.ID,
		Status:         order.Status,
		DeliveryMethod: order.DeliveryMethod,
		Subtotal:       formatOrderAmount(order.SubtotalCents, order.SubtotalPoints, currency),
		Total:          formatOrderAmount(order.TotalCents, order.TotalPoints, currency),
		ItemCount:      orderTotalQuantity(order.Items),
		Items:          orderDetailItems(ctx, order, imageKeys, resolve, currency),
		CreatedAt:      order.CreatedAt.UTC().Format(time.RFC3339),
	}
	if order.SeatNumber != nil {
		resp.SeatNumber = strings.TrimSpace(*order.SeatNumber)
	}
	if order.PaymentMethod != nil {
		resp.PaymentMethod = strings.TrimSpace(*order.PaymentMethod)
	}
	if order.ShippingCents > 0 || order.ShippingPoints > 0 {
		resp.Shipping = formatOrderAmount(order.ShippingCents, order.ShippingPoints, currency)
	}
	if order.PaidAt != nil {
		resp.PaidAt = order.PaidAt.UTC().Format(time.RFC3339)
	}
	return resp
}

func (s *orderService) loadOrderItemImageKeys(ctx context.Context, orders []models.Order) (map[uuid.UUID]string, error) {
	seen := make(map[uuid.UUID]struct{})
	ids := make([]uuid.UUID, 0)
	for i := range orders {
		for _, item := range orders[i].Items {
			if _, ok := seen[item.ProductID]; ok {
				continue
			}
			seen[item.ProductID] = struct{}{}
			ids = append(ids, item.ProductID)
		}
	}
	return s.productRepo.ImageKeysByIDs(ctx, ids)
}

func orderTotalQuantity(items []models.OrderItem) int {
	total := 0
	for _, item := range items {
		total += item.Quantity
	}
	return total
}

func orderPreviewItems(ctx context.Context, order *models.Order, imageKeys map[uuid.UUID]string, resolve func(context.Context, string) string) []dto.OrderListPreviewItem {
	if len(order.Items) == 0 {
		return nil
	}

	out := make([]dto.OrderListPreviewItem, 0, orderListImageLimit)
	for _, item := range order.Items {
		out = append(out, dto.OrderListPreviewItem{
			ProductID: item.ProductID,
			Name:      strings.TrimSpace(item.Name),
			ImageURL:  resolve(ctx, imageKeys[item.ProductID]),
			Quantity:  item.Quantity,
		})
		if len(out) >= orderListImageLimit {
			break
		}
	}
	return out
}

func orderDetailItems(ctx context.Context, order *models.Order, imageKeys map[uuid.UUID]string, resolve func(context.Context, string) string, currency string) []dto.OrderDetailItem {
	if len(order.Items) == 0 {
		return nil
	}

	out := make([]dto.OrderDetailItem, len(order.Items))
	for i, item := range order.Items {
		subname := ""
		if item.Subname != nil {
			subname = strings.TrimSpace(*item.Subname)
		}
		out[i] = dto.OrderDetailItem{
			ID:          item.ID,
			ProductID:   item.ProductID,
			ProductType: item.ProductType,
			Size:        strings.TrimSpace(item.Size),
			Name:        strings.TrimSpace(item.Name),
			Subname:     subname,
			Price:       formatOrderLinePrice(item, currency),
			ImageURL:    resolve(ctx, imageKeys[item.ProductID]),
			Quantity:    item.Quantity,
		}
	}
	return out
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

func listDisplayCurrency(order *models.Order) string {
	if order.PaymentMethod != nil && *order.PaymentMethod == models.PaymentMethodPoints {
		return shopdto.CurrencyPoint
	}
	return shopdto.CurrencyEUR
}

func formatOrderLinePrice(item models.OrderItem, currency string) string {
	if currency == shopdto.CurrencyPoint {
		return utils.FormatPoints(item.PricePoints)
	}
	return utils.FormatEuro(item.PriceCents)
}

func formatOrderAmount(cents int64, points int, currency string) string {
	if currency == shopdto.CurrencyPoint {
		return utils.FormatPoints(points)
	}
	return utils.FormatEuro(cents)
}
