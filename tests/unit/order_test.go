package unit

import (
	"context"
	"sort"
	"testing"
	"time"

	authmodels "clap/internal/modules/auth/models"
	orderdto "clap/internal/modules/order/dto"
	ordermodels "clap/internal/modules/order/models"
	orderrepo "clap/internal/modules/order/repository"
	ordersvc "clap/internal/modules/order/service"
	shopmodels "clap/internal/modules/shop/models"
	shoprepo "clap/internal/modules/shop/repository"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/database"
	"clap/pkg/payment"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubOrderRepo struct {
	orders map[uuid.UUID]*ordermodels.Order
}

func newStubOrderRepo() *stubOrderRepo {
	return &stubOrderRepo{orders: map[uuid.UUID]*ordermodels.Order{}}
}

func (r *stubOrderRepo) CreateWithItems(_ context.Context, order *ordermodels.Order, items []ordermodels.OrderItem) error {
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now().UTC()
	}
	cp := *order
	cp.Items = append([]ordermodels.OrderItem(nil), items...)
	for i := range cp.Items {
		if cp.Items[i].ID == uuid.Nil {
			cp.Items[i].ID = uuid.New()
		}
		cp.Items[i].OrderID = cp.ID
	}
	r.orders[cp.ID] = &cp
	*order = cp
	return nil
}

func (r *stubOrderRepo) FindByIDForUser(_ context.Context, userID, orderID uuid.UUID) (*ordermodels.Order, error) {
	o, ok := r.orders[orderID]
	if !ok || o.UserID != userID {
		return nil, sharederrors.NewNotFound("Order not found", nil)
	}
	cp := *o
	return &cp, nil
}

func (r *stubOrderRepo) FindByID(_ context.Context, orderID uuid.UUID) (*ordermodels.Order, error) {
	o, ok := r.orders[orderID]
	if !ok {
		return nil, sharederrors.NewNotFound("Order not found", nil)
	}
	cp := *o
	cp.Items = append([]ordermodels.OrderItem(nil), o.Items...)
	return &cp, nil
}

func (r *stubOrderRepo) ListForUserAfter(_ context.Context, userID uuid.UUID, limit int, after *orderrepo.OrderCursorAnchor) ([]ordermodels.Order, error) {
	var userOrders []ordermodels.Order
	for _, o := range r.orders {
		if o.UserID != userID {
			continue
		}
		cp := *o
		cp.Items = append([]ordermodels.OrderItem(nil), o.Items...)
		userOrders = append(userOrders, cp)
	}
	sort.Slice(userOrders, func(i, j int) bool {
		if userOrders[i].CreatedAt.Equal(userOrders[j].CreatedAt) {
			return userOrders[i].ID.String() > userOrders[j].ID.String()
		}
		return userOrders[i].CreatedAt.After(userOrders[j].CreatedAt)
	})
	if after != nil {
		filtered := userOrders[:0]
		for _, o := range userOrders {
			if o.CreatedAt.Before(after.CreatedAt) ||
				(o.CreatedAt.Equal(after.CreatedAt) && o.ID != after.ID && o.ID.String() < after.ID.String()) {
				filtered = append(filtered, o)
			}
		}
		userOrders = filtered
	}
	if limit > 0 && len(userOrders) > limit {
		userOrders = userOrders[:limit]
	}
	return userOrders, nil
}

func (r *stubOrderRepo) MarkPaid(_ context.Context, orderID uuid.UUID, _ time.Time, paymentMethod string) error {
	o, ok := r.orders[orderID]
	if !ok {
		return sharederrors.NewNotFound("Order not found", nil)
	}
	if o.Status != ordermodels.OrderStatusPendingPayment && o.Status != ordermodels.OrderStatusCancelled {
		return sharederrors.NewUnprocessable("Order is not pending payment", nil)
	}
	o.Status = ordermodels.OrderStatusPaid
	if paymentMethod != "" {
		o.PaymentMethod = &paymentMethod
	}
	return nil
}

func (r *stubOrderRepo) MarkCancelled(_ context.Context, orderID uuid.UUID) error {
	o, ok := r.orders[orderID]
	if !ok {
		return sharederrors.NewNotFound("Order not found", nil)
	}
	if o.Status != ordermodels.OrderStatusPendingPayment {
		return nil
	}
	o.Status = ordermodels.OrderStatusCancelled
	o.StripePaymentIntentID = nil
	return nil
}

func (r *stubOrderRepo) UpdatePendingCheckout(_ context.Context, orderID uuid.UUID, updates map[string]interface{}) error {
	o, ok := r.orders[orderID]
	if !ok {
		return sharederrors.NewNotFound("Order not found", nil)
	}
	if o.Status != ordermodels.OrderStatusPendingPayment {
		return sharederrors.NewUnprocessable("Order is not pending payment", nil)
	}
	if v, ok := updates["delivery_method"].(string); ok {
		o.DeliveryMethod = v
	}
	if v, exists := updates["seat_number"]; exists {
		if v == nil {
			o.SeatNumber = nil
		} else if n, ok := v.(int); ok {
			o.SeatNumber = &n
		}
	}
	if v, ok := updates["zone"].(string); ok {
		o.Zone = v
	}
	if v, ok := updates["payment_method"].(string); ok {
		o.PaymentMethod = &v
	}
	if v, ok := updates["shipping_cents"].(int64); ok {
		o.ShippingCents = v
	}
	if v, ok := updates["shipping_points"].(int); ok {
		o.ShippingPoints = v
	}
	if v, ok := updates["total_cents"].(int64); ok {
		o.TotalCents = v
	}
	if v, ok := updates["total_points"].(int); ok {
		o.TotalPoints = v
	}
	if _, ok := updates["stripe_payment_intent_id"]; ok {
		o.StripePaymentIntentID = nil
	}
	return nil
}

func (r *stubOrderRepo) ListExpiredPending(_ context.Context, cutoff time.Time) ([]ordermodels.Order, error) {
	var expired []ordermodels.Order
	for _, o := range r.orders {
		if o.Status != ordermodels.OrderStatusPendingPayment {
			continue
		}
		if o.CreatedAt.IsZero() || o.CreatedAt.After(cutoff) {
			continue
		}
		cp := *o
		cp.Items = append([]ordermodels.OrderItem(nil), o.Items...)
		expired = append(expired, cp)
	}
	return expired, nil
}

func (r *stubOrderRepo) FindByStripePaymentIntentID(_ context.Context, intentID string) (*ordermodels.Order, error) {
	for _, o := range r.orders {
		if o.StripePaymentIntentID != nil && *o.StripePaymentIntentID == intentID {
			cp := *o
			return &cp, nil
		}
	}
	return nil, sharederrors.NewNotFound("Order not found", nil)
}

func (r *stubOrderRepo) UpdateStripePaymentIntentID(_ context.Context, orderID uuid.UUID, intentID string) error {
	o, ok := r.orders[orderID]
	if !ok {
		return sharederrors.NewNotFound("Order not found", nil)
	}
	if o.Status != ordermodels.OrderStatusPendingPayment {
		return sharederrors.NewUnprocessable("Order is not pending payment", nil)
	}
	o.StripePaymentIntentID = &intentID
	return nil
}

func (r *stubOrderRepo) RecordStripeEvent(_ context.Context, eventID, _ string, _ *uuid.UUID) (bool, error) {
	return true, nil
}

type stubPaymentProvider struct {
	enabled bool
	session *payment.CheckoutSessionResult
	orderID uuid.UUID
}

func (s stubPaymentProvider) Enabled() bool { return s.enabled }

func (s stubPaymentProvider) CreateCheckoutSession(_ context.Context, _ payment.CreateCheckoutParams) (*payment.CheckoutSessionResult, error) {
	return s.session, nil
}

func (s stubPaymentProvider) GetCheckoutSession(_ context.Context, sessionID string) (*payment.CheckoutSessionStatus, error) {
	return &payment.CheckoutSessionStatus{
		SessionID:     sessionID,
		PaymentStatus: "paid",
		OrderID:       s.orderID,
	}, nil
}

func (s stubPaymentProvider) EmailCheckoutInvoice(context.Context, string) error {
	return nil
}

func (s stubPaymentProvider) ExpireCheckoutSession(context.Context, string) error {
	return nil
}

func (s stubPaymentProvider) ParseWebhookEvent(_ []byte, _ string) (*payment.WebhookEvent, error) {
	return nil, nil
}

func newOrderService(
	orderRepo orderrepo.OrderRepository,
	cart shoprepo.CartRepository,
	product shoprepo.ProductRepository,
	sizeStock shoprepo.ProductSizeStockRepository,
	user *stubUserRepo,
	paymentProvider payment.Provider,
) ordersvc.OrderService {
	return ordersvc.NewOrderService(orderRepo, cart, product, sizeStock, user, nil, paymentProvider, "smartklap")
}

type stubCartRepo struct {
	lines []shoprepo.UserCartLine
}

func (r *stubCartRepo) FindLine(context.Context, uuid.UUID, uuid.UUID, string) (*shopmodels.CartItem, error) {
	return nil, nil
}
func (r *stubCartRepo) FindLineByID(context.Context, uuid.UUID, uuid.UUID) (*shoprepo.UserCartLine, error) {
	return nil, sharederrors.NewNotFound("Cart item not found", nil)
}
func (r *stubCartRepo) ListUserLines(context.Context, uuid.UUID) ([]shoprepo.UserCartLine, error) {
	return append([]shoprepo.UserCartLine(nil), r.lines...), nil
}
func (r *stubCartRepo) ListUserLinesAfter(context.Context, uuid.UUID, int, *shoprepo.CartCursorAnchor) ([]shoprepo.UserCartLine, error) {
	return r.lines, nil
}
func (r *stubCartRepo) SumSubtotalCents(_ context.Context, _ uuid.UUID) (int64, error) {
	var total int64
	for _, line := range r.lines {
		total += line.PriceCents * int64(line.Quantity)
	}
	return total, nil
}
func (r *stubCartRepo) SumSubtotalPoints(_ context.Context, _ uuid.UUID) (int, error) {
	var total int
	for _, line := range r.lines {
		total += line.PricePoints * line.Quantity
	}
	return total, nil
}
func (r *stubCartRepo) Create(context.Context, *shopmodels.CartItem) error { return nil }
func (r *stubCartRepo) UpdateQuantity(context.Context, uuid.UUID, int) error {
	return nil
}
func (r *stubCartRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (r *stubCartRepo) CountTotalQuantity(_ context.Context, _ uuid.UUID) (int, error) {
	total := 0
	for _, line := range r.lines {
		total += line.Quantity
	}
	return total, nil
}
func (r *stubCartRepo) DeleteAllForUser(_ context.Context, _ uuid.UUID) error {
	r.lines = nil
	return nil
}
func (r *stubCartRepo) ClearUserCart(_ context.Context, _ uuid.UUID) error {
	r.lines = nil
	return nil
}

type stubProductRepo struct {
	imageKeys map[uuid.UUID]string
}

func (r stubProductRepo) List(context.Context, int, shoprepo.ProductFilters, *shoprepo.CursorAnchor) ([]shopmodels.Product, error) {
	return nil, nil
}
func (r stubProductRepo) FindByID(_ context.Context, id uuid.UUID) (*shopmodels.Product, error) {
	qty := 100
	return &shopmodels.Product{ID: id, StockQuantity: &qty}, nil
}
func (stubProductRepo) FindByIDAdmin(context.Context, uuid.UUID) (*shopmodels.Product, error) {
	return nil, nil
}
func (stubProductRepo) Create(context.Context, *shopmodels.Product) error { return nil }
func (stubProductRepo) Update(context.Context, *shopmodels.Product) error { return nil }
func (stubProductRepo) UpdateImageKey(context.Context, uuid.UUID, string) error {
	return nil
}
func (r stubProductRepo) ImageKeysByIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	out := make(map[uuid.UUID]string)
	for _, id := range ids {
		if r.imageKeys != nil {
			if key, ok := r.imageKeys[id]; ok {
				out[id] = key
			}
		}
	}
	return out, nil
}
func (stubProductRepo) Delete(context.Context, uuid.UUID) error { return nil }
func (stubProductRepo) DecrementStockForOrder(context.Context, uuid.UUID, int) error {
	return nil
}

type stubSizeStockRepo struct{}

func (stubSizeStockRepo) ListByProductID(context.Context, uuid.UUID) ([]shopmodels.ProductSizeStock, error) {
	return nil, nil
}
func (stubSizeStockRepo) ListByProductIDs(context.Context, []uuid.UUID) (map[uuid.UUID][]shopmodels.ProductSizeStock, error) {
	return map[uuid.UUID][]shopmodels.ProductSizeStock{}, nil
}
func (stubSizeStockRepo) ReplaceForProduct(context.Context, uuid.UUID, []shopmodels.ProductSizeStock) error {
	return nil
}
func (stubSizeStockRepo) DecrementForOrder(context.Context, uuid.UUID, string, int) error {
	return nil
}

func seedOrderUserRepo(userID uuid.UUID, points int) *stubUserRepo {
	repo := newStubUserRepo()
	repo.byID[userID] = &authmodels.User{
		BaseModel: database.BaseModel{ID: userID},
		Points:    points,
	}
	return repo
}

func TestOrderService_CreateOrderFromCart(t *testing.T) {
	userID := uuid.New()
	productID := uuid.New()
	cart := &stubCartRepo{
		lines: []shoprepo.UserCartLine{
			{
				CartItem: shopmodels.CartItem{
					ProductID:   productID,
					ProductType: shopmodels.ProductTypeFood,
					Quantity:    2,
				},
				Name:       "Burger",
				PriceCents: 820,
			},
		},
	}

	svc := newOrderService(newStubOrderRepo(), cart, stubProductRepo{}, stubSizeStockRepo{}, newStubUserRepo(), nil)
	seat := 1
	zone := "A"
	resp, err := svc.CreateOrder(context.Background(), userID, &orderdto.CreateOrderRequest{
		DeliveryMethod: ordermodels.DeliveryMethodSeat,
		SeatNumber:     &seat,
		Zone:           &zone,
	})
	require.NoError(t, err)
	assert.Equal(t, ordermodels.OrderStatusPendingPayment, resp.Status)
	assert.Equal(t, "16,40 €", resp.Subtotal)
	assert.Equal(t, "16,40 €", resp.Total)
	assert.Len(t, cart.lines, 0)

	detail, err := svc.GetOrder(context.Background(), userID, resp.OrderID)
	require.NoError(t, err)
	assert.Equal(t, "A", detail.Zone)
	require.NotNil(t, detail.SeatNumber)
	assert.Equal(t, 1, *detail.SeatNumber)
}

func TestOrderService_CreateOrderIncludesTax(t *testing.T) {
	userID := uuid.New()
	productID := uuid.New()
	cart := &stubCartRepo{
		lines: []shoprepo.UserCartLine{
			{
				CartItem: shopmodels.CartItem{
					ProductID:   productID,
					ProductType: shopmodels.ProductTypeFood,
					Quantity:    2,
				},
				Name:       "Burger",
				PriceCents: 820,
				TaxRateBps: 700,
			},
		},
	}

	svc := newOrderService(newStubOrderRepo(), cart, stubProductRepo{}, stubSizeStockRepo{}, newStubUserRepo(), nil)
	seat := 1
	zone := "A"
	resp, err := svc.CreateOrder(context.Background(), userID, &orderdto.CreateOrderRequest{
		DeliveryMethod: ordermodels.DeliveryMethodSeat,
		SeatNumber:     &seat,
		Zone:           &zone,
	})
	require.NoError(t, err)
	assert.Equal(t, "17,54 €", resp.Subtotal)
	assert.Equal(t, "1,14 €", resp.Tax)
	assert.Equal(t, "17,54 €", resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "8,77 €", resp.Items[0].Price)

	detail, err := svc.GetOrder(context.Background(), userID, resp.OrderID)
	require.NoError(t, err)
	assert.Equal(t, "1,14 €", detail.Tax)
	require.Len(t, detail.Items, 1)
	assert.Equal(t, "8,77 €", detail.Items[0].Price)
}

func TestOrderService_CreateSeatOrderRequiresZone(t *testing.T) {
	userID := uuid.New()
	cart := &stubCartRepo{
		lines: []shoprepo.UserCartLine{
			{
				CartItem: shopmodels.CartItem{
					ProductID:   uuid.New(),
					ProductType: shopmodels.ProductTypeFood,
					Quantity:    1,
				},
				Name:       "Burger",
				PriceCents: 820,
			},
		},
	}

	svc := newOrderService(newStubOrderRepo(), cart, stubProductRepo{}, stubSizeStockRepo{}, newStubUserRepo(), nil)
	seat := 12
	_, err := svc.CreateOrder(context.Background(), userID, &orderdto.CreateOrderRequest{
		DeliveryMethod: ordermodels.DeliveryMethodSeat,
		SeatNumber:     &seat,
	})
	require.Error(t, err)
	assert.Len(t, cart.lines, 1)
}

func TestOrderService_CalculateOrderWithPoints(t *testing.T) {
	userID := uuid.New()
	cart := &stubCartRepo{
		lines: []shoprepo.UserCartLine{
			{
				CartItem: shopmodels.CartItem{
					ProductID:   uuid.New(),
					ProductType: shopmodels.ProductTypeMerch,
					Quantity:    1,
				},
				Name:        "Shirt",
				PriceCents:  3250,
				PricePoints: 3250,
			},
		},
	}

	svc := newOrderService(newStubOrderRepo(), cart, stubProductRepo{}, stubSizeStockRepo{}, seedOrderUserRepo(userID, 5000), nil)
	resp, err := svc.CalculateOrder(context.Background(), userID, &orderdto.CalculateOrderRequest{
		DeliveryMethod: ordermodels.DeliveryMethodPickup,
		PaymentMethod:  "points",
	})
	require.NoError(t, err)
	assert.Equal(t, "3250 P", resp.Subtotal)
	assert.Equal(t, "3200 P", resp.Total)
	assert.Equal(t, "50 P", resp.DeliverySavings)
	assert.Equal(t, "3200 P", resp.PaymentAmount)
	assert.Equal(t, 3200, resp.PointsRequired)
	assert.Equal(t, 5000, resp.UserPoints)
	assert.True(t, resp.SufficientPoints)
}

func TestOrderService_CalculateOrderInsufficientPoints(t *testing.T) {
	userID := uuid.New()
	cart := &stubCartRepo{
		lines: []shoprepo.UserCartLine{
			{
				CartItem: shopmodels.CartItem{
					ProductID:   uuid.New(),
					ProductType: shopmodels.ProductTypeFood,
					Quantity:    2,
				},
				Name:        "Burger",
				PriceCents:  820,
				PricePoints: 820,
			},
		},
	}

	svc := newOrderService(newStubOrderRepo(), cart, stubProductRepo{}, stubSizeStockRepo{}, seedOrderUserRepo(userID, 100), nil)
	resp, err := svc.CalculateOrder(context.Background(), userID, &orderdto.CalculateOrderRequest{
		DeliveryMethod: ordermodels.DeliveryMethodSeat,
		PaymentMethod:  "points",
	})
	require.NoError(t, err)
	assert.Equal(t, 1640, resp.PointsRequired)
	assert.False(t, resp.SufficientPoints)
}

func TestOrderService_CreateOrderInsufficientPoints(t *testing.T) {
	userID := uuid.New()
	cart := &stubCartRepo{
		lines: []shoprepo.UserCartLine{
			{
				CartItem: shopmodels.CartItem{
					ProductID:   uuid.New(),
					ProductType: shopmodels.ProductTypeFood,
					Quantity:    2,
				},
				Name:        "Burger",
				PriceCents:  820,
				PricePoints: 820,
			},
		},
	}

	svc := newOrderService(newStubOrderRepo(), cart, stubProductRepo{}, stubSizeStockRepo{}, seedOrderUserRepo(userID, 100), nil)
	seat := 12
	zone := "B"
	_, err := svc.CreateOrder(context.Background(), userID, &orderdto.CreateOrderRequest{
		DeliveryMethod: ordermodels.DeliveryMethodSeat,
		SeatNumber:     &seat,
		Zone:           &zone,
		Currency:       "POINT",
	})
	require.Error(t, err)
	assert.Len(t, cart.lines, 1)
}

func TestOrderService_ListOrders(t *testing.T) {
	userID := uuid.New()
	otherUser := uuid.New()
	orderRepo := newStubOrderRepo()
	now := time.Now().UTC()
	older := now.Add(-time.Hour)
	newer := now.Add(-time.Minute)

	order1 := uuid.New()
	order2 := uuid.New()
	shirtProduct := uuid.New()
	burgerProduct := uuid.New()
	orderRepo.orders[order1] = &ordermodels.Order{
		UserID:         userID,
		Status:         ordermodels.OrderStatusPaid,
		DeliveryMethod: ordermodels.DeliveryMethodSeat,
		SubtotalCents:  1640,
		TotalCents:     1640,
		CreatedAt:      older,
		Items:          []ordermodels.OrderItem{{ProductID: burgerProduct, Name: "Burger", Quantity: 2}},
	}
	orderRepo.orders[order1].ID = order1
	orderRepo.orders[order2] = &ordermodels.Order{
		UserID:         userID,
		Status:         ordermodels.OrderStatusPendingPayment,
		DeliveryMethod: ordermodels.DeliveryMethodPickup,
		SubtotalPoints: 3600,
		TotalPoints:    3600,
		CreatedAt:      newer,
		Items:          []ordermodels.OrderItem{{ProductID: shirtProduct, Name: "Shirt", Quantity: 1}},
	}
	orderRepo.orders[order2].ID = order2
	otherOrder := uuid.New()
	orderRepo.orders[otherOrder] = &ordermodels.Order{
		UserID: otherUser,
		Status: ordermodels.OrderStatusPaid,
	}
	orderRepo.orders[otherOrder].ID = otherOrder

	productRepo := stubProductRepo{imageKeys: map[uuid.UUID]string{
		shirtProduct:  "https://example.com/shirt.jpg",
		burgerProduct: "https://example.com/burger.jpg",
	}}
	svc := newOrderService(orderRepo, &stubCartRepo{}, productRepo, stubSizeStockRepo{}, newStubUserRepo(), nil)
	resp, err := svc.ListOrders(context.Background(), userID, orderdto.OrderListFilters{Limit: 1})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, order2, resp.Items[0].OrderID)
	require.Len(t, resp.Items[0].Items, 1)
	assert.Equal(t, shirtProduct, resp.Items[0].Items[0].ProductID)
	assert.Equal(t, "Shirt", resp.Items[0].Items[0].Name)
	assert.Equal(t, 1, resp.Items[0].Items[0].Quantity)
	assert.Equal(t, "https://example.com/shirt.jpg", resp.Items[0].Items[0].ImageURL)
	assert.Equal(t, 1, resp.Items[0].ItemCount)
	assert.True(t, resp.Meta.HasMore)
	require.NotNil(t, resp.Meta.NextCursor)
	assert.Equal(t, order2, *resp.Meta.NextCursor)

	page2, err := svc.ListOrders(context.Background(), userID, orderdto.OrderListFilters{
		Cursor: resp.Meta.NextCursor,
		Limit:  1,
	})
	require.NoError(t, err)
	require.Len(t, page2.Items, 1)
	assert.Equal(t, order1, page2.Items[0].OrderID)
	assert.False(t, page2.Meta.HasMore)
}

func TestOrderService_GetOrder(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	shirtProduct := uuid.New()
	burgerProduct := uuid.New()
	orderID := uuid.New()
	paidAt := time.Now().UTC().Add(-time.Hour)
	paymentMethod := ordermodels.PaymentMethodPoints
	orderRepo.orders[orderID] = &ordermodels.Order{
		UserID:         userID,
		Status:         ordermodels.OrderStatusPaid,
		DeliveryMethod: ordermodels.DeliveryMethodPickup,
		SubtotalPoints: 3600,
		ShippingPoints: 350,
		TotalPoints:    3950,
		PaymentMethod:  &paymentMethod,
		PaidAt:         &paidAt,
		CreatedAt:      paidAt.Add(-time.Minute),
		Items: []ordermodels.OrderItem{
			{ID: uuid.New(), ProductID: shirtProduct, ProductType: shopmodels.ProductTypeMerch, Name: "Shirt", Quantity: 1, PricePoints: 3250},
			{ID: uuid.New(), ProductID: burgerProduct, ProductType: shopmodels.ProductTypeFood, Name: "Burger", Quantity: 2, PricePoints: 820},
		},
	}
	orderRepo.orders[orderID].ID = orderID

	productRepo := stubProductRepo{imageKeys: map[uuid.UUID]string{
		shirtProduct:  "https://example.com/shirt.jpg",
		burgerProduct: "https://example.com/burger.jpg",
	}}
	svc := newOrderService(orderRepo, &stubCartRepo{}, productRepo, stubSizeStockRepo{}, newStubUserRepo(), nil)

	resp, err := svc.GetOrder(context.Background(), userID, orderID)
	require.NoError(t, err)
	assert.Equal(t, orderID, resp.OrderID)
	assert.Equal(t, ordermodels.OrderStatusPaid, resp.Status)
	assert.Equal(t, 3, resp.ItemCount)
	require.Len(t, resp.Items, 2)
	assert.Equal(t, "Shirt", resp.Items[0].Name)
	assert.Equal(t, 1, resp.Items[0].Quantity)
	assert.Equal(t, "https://example.com/shirt.jpg", resp.Items[0].ImageURL)
	assert.Equal(t, "Burger", resp.Items[1].Name)
	assert.Equal(t, 2, resp.Items[1].Quantity)
	assert.NotEmpty(t, resp.PaidAt)

	_, err = svc.GetOrder(context.Background(), uuid.New(), orderID)
	require.Error(t, err)
}

func TestOrderService_PayOrderWithPoints(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	userRepo := seedOrderUserRepo(userID, 5000)
	orderID := uuid.New()
	orderRepo.orders[orderID] = &ordermodels.Order{
		UserID:         userID,
		Status:         ordermodels.OrderStatusPendingPayment,
		TotalPoints:    1640,
		SubtotalPoints: 1640,
		Items: []ordermodels.OrderItem{
			{ProductID: uuid.New(), Quantity: 2, PricePoints: 820, Name: "Burger"},
		},
	}
	orderRepo.orders[orderID].ID = orderID

	svc := newOrderService(orderRepo, &stubCartRepo{}, stubProductRepo{}, stubSizeStockRepo{}, userRepo, nil)
	resp, err := svc.PayOrder(context.Background(), userID, orderID, &orderdto.PayOrderRequest{PaymentMethod: "points"})
	require.NoError(t, err)
	assert.Equal(t, ordermodels.OrderStatusPaid, resp.Status)
	require.NotNil(t, resp.PointsRemaining)
	assert.Equal(t, 3360, *resp.PointsRemaining)
}

func TestOrderService_PayOrderInsufficientPoints(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	userRepo := seedOrderUserRepo(userID, 100)
	orderID := uuid.New()
	orderRepo.orders[orderID] = &ordermodels.Order{
		UserID:      userID,
		Status:      ordermodels.OrderStatusPendingPayment,
		TotalPoints: 1640,
	}
	orderRepo.orders[orderID].ID = orderID

	svc := newOrderService(orderRepo, &stubCartRepo{}, stubProductRepo{}, stubSizeStockRepo{}, userRepo, nil)
	_, err := svc.PayOrder(context.Background(), userID, orderID, &orderdto.PayOrderRequest{PaymentMethod: "points"})
	require.Error(t, err)
}

func TestOrderService_CreateOrderWithCardWithoutPoints(t *testing.T) {
	userID := uuid.New()
	cart := &stubCartRepo{
		lines: []shoprepo.UserCartLine{
			{
				CartItem: shopmodels.CartItem{
					ProductID:   uuid.New(),
					ProductType: shopmodels.ProductTypeFood,
					Quantity:    1,
				},
				Name:       "Burger",
				PriceCents: 820,
			},
		},
	}

	svc := newOrderService(newStubOrderRepo(), cart, stubProductRepo{}, stubSizeStockRepo{}, seedOrderUserRepo(userID, 0), nil)
	seat := 5
	zone := "C"
	resp, err := svc.CreateOrder(context.Background(), userID, &orderdto.CreateOrderRequest{
		DeliveryMethod: ordermodels.DeliveryMethodSeat,
		SeatNumber:     &seat,
		Zone:           &zone,
		Currency:       "EUR",
	})
	require.NoError(t, err)
	assert.Equal(t, ordermodels.OrderStatusPendingPayment, resp.Status)
}

func TestOrderService_PayOrderWithCard(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	orderID := uuid.New()
	orderRepo.orders[orderID] = &ordermodels.Order{
		UserID:      userID,
		Status:      ordermodels.OrderStatusPendingPayment,
		TotalCents:  1640,
		SubtotalCents: 1640,
	}
	orderRepo.orders[orderID].ID = orderID

	checkoutURL := "https://checkout.stripe.test/session"
	sessionID := "cs_test"
	svc := newOrderService(
		orderRepo,
		&stubCartRepo{},
		stubProductRepo{},
		stubSizeStockRepo{},
		newStubUserRepo(),
		stubPaymentProvider{
			enabled: true,
			orderID: orderID,
			session: &payment.CheckoutSessionResult{SessionID: sessionID, CheckoutURL: checkoutURL},
		},
	)

	resp, err := svc.PayOrder(context.Background(), userID, orderID, &orderdto.PayOrderRequest{PaymentMethod: "card"})
	require.NoError(t, err)
	assert.Equal(t, ordermodels.OrderStatusPendingPayment, resp.Status)
	require.NotNil(t, resp.CheckoutURL)
	assert.Equal(t, checkoutURL, *resp.CheckoutURL)
}

func TestOrderService_ConfirmCardPayment(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	orderID := uuid.New()
	sessionID := "cs_test_confirm"
	orderRepo.orders[orderID] = &ordermodels.Order{
		UserID:                userID,
		Status:                ordermodels.OrderStatusPendingPayment,
		TotalCents:            2200,
		StripePaymentIntentID: &sessionID,
		Items: []ordermodels.OrderItem{
			{ProductID: uuid.New(), Quantity: 1, PriceCents: 2200, Name: "Burger"},
		},
	}
	orderRepo.orders[orderID].ID = orderID

	svc := newOrderService(
		orderRepo,
		&stubCartRepo{},
		stubProductRepo{},
		stubSizeStockRepo{},
		newStubUserRepo(),
		stubPaymentProvider{enabled: true, orderID: orderID},
	)

	resp, err := svc.ConfirmCardPayment(context.Background(), userID, orderID)
	require.NoError(t, err)
	assert.Equal(t, ordermodels.OrderStatusPaid, resp.Status)
	assert.Equal(t, ordermodels.OrderStatusPaid, orderRepo.orders[orderID].Status)
}

func TestOrderService_PayOrderWithCardNotConfigured(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	orderID := uuid.New()
	orderRepo.orders[orderID] = &ordermodels.Order{
		UserID:     userID,
		Status:     ordermodels.OrderStatusPendingPayment,
		TotalCents: 1640,
		CreatedAt:  time.Now().UTC(),
	}
	orderRepo.orders[orderID].ID = orderID

	svc := newOrderService(orderRepo, &stubCartRepo{}, stubProductRepo{}, stubSizeStockRepo{}, newStubUserRepo(), nil)
	_, err := svc.PayOrder(context.Background(), userID, orderID, &orderdto.PayOrderRequest{PaymentMethod: "card"})
	require.Error(t, err)
}

func TestOrderService_UpdatePendingDeliveryAppliesPickupDiscount(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	orderID := uuid.New()
	seat := 101
	zone := "A"
	card := ordermodels.PaymentMethodCard
	orderRepo.orders[orderID] = &ordermodels.Order{
		UserID:         userID,
		Status:         ordermodels.OrderStatusPendingPayment,
		DeliveryMethod: ordermodels.DeliveryMethodSeat,
		Zone:           zone,
		SeatNumber:     &seat,
		PaymentMethod:  &card,
		SubtotalCents:  3250,
		TotalCents:     3250,
		SubtotalPoints: 3250,
		TotalPoints:    3250,
		CreatedAt:      time.Now().UTC(),
		Items: []ordermodels.OrderItem{
			{ProductID: uuid.New(), ProductType: shopmodels.ProductTypeMerch, Name: "Shirt", Quantity: 1, PriceCents: 3250, PricePoints: 3250},
		},
	}
	orderRepo.orders[orderID].ID = orderID

	svc := newOrderService(orderRepo, &stubCartRepo{}, stubProductRepo{}, stubSizeStockRepo{}, newStubUserRepo(), nil)
	pickup := ordermodels.DeliveryMethodPickup
	resp, err := svc.UpdateOrder(context.Background(), userID, orderID, &orderdto.UpdateOrderRequest{
		DeliveryMethod: &pickup,
	})
	require.NoError(t, err)
	assert.Equal(t, ordermodels.DeliveryMethodPickup, resp.DeliveryMethod)
	assert.Empty(t, resp.Shipping)
	assert.Equal(t, "32,50 €", resp.Subtotal)
	assert.Equal(t, "32,00 €", resp.Total)
	assert.Equal(t, ordermodels.OrderStatusPendingPayment, resp.Status)
	assert.NotEmpty(t, resp.ExpiresAt)
}

func TestOrderService_UpdatePaymentMethodThenPay(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	orderID := uuid.New()
	card := ordermodels.PaymentMethodCard
	orderRepo.orders[orderID] = &ordermodels.Order{
		UserID:         userID,
		Status:         ordermodels.OrderStatusPendingPayment,
		DeliveryMethod: ordermodels.DeliveryMethodSeat,
		PaymentMethod:  &card,
		SubtotalPoints: 1640,
		TotalPoints:    1640,
		CreatedAt:      time.Now().UTC(),
		Items: []ordermodels.OrderItem{
			{ProductID: uuid.New(), ProductType: shopmodels.ProductTypeFood, Name: "Burger", Quantity: 2, PricePoints: 820},
		},
	}
	orderRepo.orders[orderID].ID = orderID

	svc := newOrderService(orderRepo, &stubCartRepo{}, stubProductRepo{}, stubSizeStockRepo{}, seedOrderUserRepo(userID, 5000), nil)
	points := ordermodels.PaymentMethodPoints
	updated, err := svc.UpdateOrder(context.Background(), userID, orderID, &orderdto.UpdateOrderRequest{
		PaymentMethod: &points,
	})
	require.NoError(t, err)
	assert.Equal(t, ordermodels.PaymentMethodPoints, updated.PaymentMethod)

	resp, err := svc.PayOrder(context.Background(), userID, orderID, &orderdto.PayOrderRequest{PaymentMethod: "points"})
	require.NoError(t, err)
	assert.Equal(t, ordermodels.OrderStatusPaid, resp.Status)
}

func TestOrderService_PayExpiredOrderCancels(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	orderID := uuid.New()
	orderRepo.orders[orderID] = &ordermodels.Order{
		UserID:      userID,
		Status:      ordermodels.OrderStatusPendingPayment,
		TotalPoints: 1640,
		CreatedAt:   time.Now().UTC().Add(-11 * time.Minute),
	}
	orderRepo.orders[orderID].ID = orderID

	svc := newOrderService(orderRepo, &stubCartRepo{}, stubProductRepo{}, stubSizeStockRepo{}, seedOrderUserRepo(userID, 5000), nil)
	_, err := svc.PayOrder(context.Background(), userID, orderID, &orderdto.PayOrderRequest{PaymentMethod: "points"})
	require.Error(t, err)
	assert.Equal(t, ordermodels.OrderStatusCancelled, orderRepo.orders[orderID].Status)
}

func TestOrderService_GetExpiredOrderMarksCancelled(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	orderID := uuid.New()
	orderRepo.orders[orderID] = &ordermodels.Order{
		UserID:    userID,
		Status:    ordermodels.OrderStatusPendingPayment,
		CreatedAt: time.Now().UTC().Add(-11 * time.Minute),
		Items:     []ordermodels.OrderItem{{ProductID: uuid.New(), Name: "Burger", Quantity: 1}},
	}
	orderRepo.orders[orderID].ID = orderID

	svc := newOrderService(orderRepo, &stubCartRepo{}, stubProductRepo{}, stubSizeStockRepo{}, newStubUserRepo(), nil)
	resp, err := svc.GetOrder(context.Background(), userID, orderID)
	require.NoError(t, err)
	assert.Equal(t, ordermodels.OrderStatusCancelled, resp.Status)
	assert.Empty(t, resp.ExpiresAt)
}

func TestOrderService_ExpirePendingOrders(t *testing.T) {
	userID := uuid.New()
	orderRepo := newStubOrderRepo()
	expiredID := uuid.New()
	freshID := uuid.New()
	orderRepo.orders[expiredID] = &ordermodels.Order{
		UserID:    userID,
		Status:    ordermodels.OrderStatusPendingPayment,
		CreatedAt: time.Now().UTC().Add(-11 * time.Minute),
	}
	orderRepo.orders[expiredID].ID = expiredID
	orderRepo.orders[freshID] = &ordermodels.Order{
		UserID:    userID,
		Status:    ordermodels.OrderStatusPendingPayment,
		CreatedAt: time.Now().UTC(),
	}
	orderRepo.orders[freshID].ID = freshID

	svc := newOrderService(orderRepo, &stubCartRepo{}, stubProductRepo{}, stubSizeStockRepo{}, newStubUserRepo(), nil)
	n, err := svc.ExpirePendingOrders(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, ordermodels.OrderStatusCancelled, orderRepo.orders[expiredID].Status)
	assert.Equal(t, ordermodels.OrderStatusPendingPayment, orderRepo.orders[freshID].Status)
}

var (
	_ orderrepo.OrderRepository           = (*stubOrderRepo)(nil)
	_ shoprepo.CartRepository             = (*stubCartRepo)(nil)
	_ shoprepo.ProductRepository          = stubProductRepo{}
	_ shoprepo.ProductSizeStockRepository = stubSizeStockRepo{}
)
