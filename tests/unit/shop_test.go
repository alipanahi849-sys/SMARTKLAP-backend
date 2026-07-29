package unit

import (
	"context"
	"net/http"
	"testing"

	shopdto "clap/internal/modules/shop/dto"
	shopmodels "clap/internal/modules/shop/models"
	shopsvc "clap/internal/modules/shop/service"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── stubs ────────────────────────────────────────────────────────────────────

type stubSnackRepo struct {
	snacks map[uuid.UUID]shopmodels.Snack
}

func (r *stubSnackRepo) FindByID(_ context.Context, id uuid.UUID) (*shopmodels.Snack, error) {
	if s, ok := r.snacks[id]; ok {
		return &s, nil
	}
	return nil, sharederrors.NewNotFound("Snack not found", nil)
}

func (r *stubSnackRepo) FindByIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]shopmodels.Snack, error) {
	result := map[uuid.UUID]shopmodels.Snack{}
	for _, id := range ids {
		if s, ok := r.snacks[id]; ok {
			result[id] = s
		}
	}
	return result, nil
}

func (r *stubSnackRepo) FindAll(_ context.Context, _, _ string, _, _ int) ([]shopmodels.Snack, int64, error) {
	var all []shopmodels.Snack
	for _, s := range r.snacks {
		all = append(all, s)
	}
	return all, int64(len(all)), nil
}

func (r *stubSnackRepo) FindPreview(_ context.Context, limit int) ([]shopmodels.Snack, error) {
	all, _, _ := r.FindAll(context.Background(), "", "", 1, limit)
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

type stubProductRepo struct {
	products map[uuid.UUID]shopmodels.Product
}

func (r *stubProductRepo) FindByID(_ context.Context, id uuid.UUID) (*shopmodels.Product, error) {
	if p, ok := r.products[id]; ok {
		return &p, nil
	}
	return nil, sharederrors.NewNotFound("Product not found", nil)
}

func (r *stubProductRepo) FindByIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]shopmodels.Product, error) {
	result := map[uuid.UUID]shopmodels.Product{}
	for _, id := range ids {
		if p, ok := r.products[id]; ok {
			result[id] = p
		}
	}
	return result, nil
}

func (r *stubProductRepo) FindAll(_ context.Context, _, _ string, _, _ int) ([]shopmodels.Product, int64, error) {
	var all []shopmodels.Product
	for _, p := range r.products {
		all = append(all, p)
	}
	return all, int64(len(all)), nil
}

func (r *stubProductRepo) FindPreview(_ context.Context, limit int) ([]shopmodels.Product, error) {
	all, _, _ := r.FindAll(context.Background(), "", "", 1, limit)
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

type stubCartRepo struct {
	items map[uuid.UUID]*shopmodels.CartItem
}

func newStubCartRepo() *stubCartRepo {
	return &stubCartRepo{items: map[uuid.UUID]*shopmodels.CartItem{}}
}

func (r *stubCartRepo) ItemsByUser(_ context.Context, userID uuid.UUID) ([]shopmodels.CartItem, error) {
	var result []shopmodels.CartItem
	for _, item := range r.items {
		if item.UserID == userID {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (r *stubCartRepo) CountByUser(_ context.Context, userID uuid.UUID) (int, error) {
	total := 0
	for _, item := range r.items {
		if item.UserID == userID {
			total += item.Quantity
		}
	}
	return total, nil
}

func (r *stubCartRepo) Upsert(_ context.Context, item *shopmodels.CartItem) error {
	for _, existing := range r.items {
		if existing.UserID == item.UserID &&
			existing.ProductType == item.ProductType &&
			existing.ProductID == item.ProductID &&
			existing.Size == item.Size {
			existing.Quantity += item.Quantity
			return nil
		}
	}
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	r.items[item.ID] = item
	return nil
}

func (r *stubCartRepo) UpdateQuantity(_ context.Context, itemID, userID uuid.UUID, quantity int) (*shopmodels.CartItem, error) {
	item, ok := r.items[itemID]
	if !ok || item.UserID != userID {
		return nil, sharederrors.NewNotFound("Cart item not found", nil)
	}
	item.Quantity = quantity
	return item, nil
}

func (r *stubCartRepo) Delete(_ context.Context, itemID, userID uuid.UUID) error {
	item, ok := r.items[itemID]
	if !ok || item.UserID != userID {
		return sharederrors.NewNotFound("Cart item not found", nil)
	}
	delete(r.items, itemID)
	return nil
}

type stubOrderRepo struct {
	cartRepo *stubCartRepo
	orders   map[uuid.UUID]*shopmodels.Order
}

func newStubOrderRepo(cartRepo *stubCartRepo) *stubOrderRepo {
	return &stubOrderRepo{cartRepo: cartRepo, orders: map[uuid.UUID]*shopmodels.Order{}}
}

func (r *stubOrderRepo) CreateFromCart(_ context.Context, order *shopmodels.Order, items []shopmodels.OrderItem) error {
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	order.Items = items
	r.orders[order.ID] = order
	// Clear the user's cart, mirroring the transactional repository.
	for id, item := range r.cartRepo.items {
		if item.UserID == order.UserID {
			delete(r.cartRepo.items, id)
		}
	}
	return nil
}

func (r *stubOrderRepo) FindByIDForUser(_ context.Context, orderID, userID uuid.UUID) (*shopmodels.Order, error) {
	if o, ok := r.orders[orderID]; ok && o.UserID == userID {
		return o, nil
	}
	return nil, sharederrors.NewNotFound("Order not found", nil)
}

func (r *stubOrderRepo) MarkPaid(_ context.Context, orderID, userID uuid.UUID, paymentMethod string) error {
	o, ok := r.orders[orderID]
	if !ok || o.UserID != userID {
		return sharederrors.NewNotFound("Order not found", nil)
	}
	if o.Status != shopmodels.OrderStatusPendingPayment {
		return sharederrors.NewConflict("Order is not awaiting payment", nil)
	}
	o.Status = shopmodels.OrderStatusPaid
	o.PaymentMethod = paymentMethod
	return nil
}

// ─── fixtures ─────────────────────────────────────────────────────────────────

func newShopFixture() (shopsvc.ShopService, *stubSnackRepo, *stubProductRepo, *stubCartRepo, *stubOrderRepo) {
	snackRepo := &stubSnackRepo{snacks: map[uuid.UUID]shopmodels.Snack{}}
	productRepo := &stubProductRepo{products: map[uuid.UUID]shopmodels.Product{}}
	cartRepo := newStubCartRepo()
	orderRepo := newStubOrderRepo(cartRepo)
	svc := shopsvc.NewShopService(snackRepo, productRepo, cartRepo, orderRepo)
	return svc, snackRepo, productRepo, cartRepo, orderRepo
}

func addSnack(repo *stubSnackRepo, name string, cents int64) shopmodels.Snack {
	s := shopmodels.Snack{ID: uuid.New(), Name: name, PriceCents: cents, PointsPrice: int(cents), IsActive: true}
	repo.snacks[s.ID] = s
	return s
}

func addProduct(repo *stubProductRepo, name string, cents int64, sizes string) shopmodels.Product {
	p := shopmodels.Product{ID: uuid.New(), Name: name, PriceCents: cents, Sizes: sizes, IsActive: true}
	repo.products[p.ID] = p
	return p
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestShop_ListSnacksInvalidCategoryRejected(t *testing.T) {
	svc, _, _, _, _ := newShopFixture()

	_, err := svc.ListSnacks(context.Background(), uuid.New(), "", "weapons", "", 1, 20)
	if err == nil {
		t.Fatal("expected validation error for invalid category")
	}
	if status := appErrorStatus(t, err); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestShop_PriceFormattingPerCurrency(t *testing.T) {
	svc, snackRepo, _, _, _ := newShopFixture()
	snack := addSnack(snackRepo, "Double Berger", 820)

	eur, err := svc.SnackDetail(context.Background(), snack.ID, shopsvc.CurrencyEUR)
	if err != nil {
		t.Fatalf("SnackDetail(EUR) failed: %v", err)
	}
	if eur.Price != "8,20 €" {
		t.Fatalf("expected \"8,20 €\", got %q", eur.Price)
	}

	points, err := svc.SnackDetail(context.Background(), snack.ID, shopsvc.CurrencyPoint)
	if err != nil {
		t.Fatalf("SnackDetail(POINT) failed: %v", err)
	}
	if points.Price != "820 P" {
		t.Fatalf("expected \"820 P\", got %q", points.Price)
	}
}

func TestShop_AddCartItemInvalidSizeRejected(t *testing.T) {
	svc, _, productRepo, _, _ := newShopFixture()
	product := addProduct(productRepo, "Sport T-shirt", 3250, `["M","L","XL"]`)

	_, err := svc.AddCartItem(context.Background(), uuid.New(), &shopdto.AddCartItemRequest{
		ProductType: shopmodels.ProductTypeMerch,
		ProductID:   product.ID,
		Quantity:    1,
		Size:        "XS",
	})
	if err == nil {
		t.Fatal("expected error for unavailable size")
	}
	if status := appErrorStatus(t, err); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestShop_AddCartItemMergesDuplicateLines(t *testing.T) {
	svc, snackRepo, _, cartRepo, _ := newShopFixture()
	snack := addSnack(snackRepo, "Hot Dog", 500)
	userID := uuid.New()

	req := &shopdto.AddCartItemRequest{
		ProductType: shopmodels.ProductTypeSnack,
		ProductID:   snack.ID,
		Quantity:    2,
	}
	if _, err := svc.AddCartItem(context.Background(), userID, req); err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	if _, err := svc.AddCartItem(context.Background(), userID, req); err != nil {
		t.Fatalf("second add failed: %v", err)
	}

	count, err := cartRepo.CountByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("CountByUser failed: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected merged quantity 4, got %d", count)
	}
}

func TestShop_CheckoutEmptyCartUnprocessable(t *testing.T) {
	svc, _, _, _, _ := newShopFixture()

	_, err := svc.Checkout(context.Background(), uuid.New(), &shopdto.CheckoutRequest{DeliveryMethod: "pickup"})
	if err == nil {
		t.Fatal("expected error for empty cart")
	}
	if status := appErrorStatus(t, err); status != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", status)
	}
}

func TestShop_CheckoutSeatWithoutSeatNumberRejected(t *testing.T) {
	svc, _, _, _, _ := newShopFixture()

	_, err := svc.Checkout(context.Background(), uuid.New(), &shopdto.CheckoutRequest{DeliveryMethod: "seat"})
	if err == nil {
		t.Fatal("expected error for missing seat number")
	}
	if status := appErrorStatus(t, err); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestShop_CheckoutComputesTotalsAndClearsCart(t *testing.T) {
	svc, snackRepo, _, cartRepo, _ := newShopFixture()
	snack := addSnack(snackRepo, "Double Berger", 820)
	userID := uuid.New()

	if _, err := svc.AddCartItem(context.Background(), userID, &shopdto.AddCartItemRequest{
		ProductType: shopmodels.ProductTypeSnack,
		ProductID:   snack.ID,
		Quantity:    2,
	}); err != nil {
		t.Fatalf("add to cart failed: %v", err)
	}

	order, err := svc.Checkout(context.Background(), userID, &shopdto.CheckoutRequest{
		DeliveryMethod: "seat",
		SeatNumber:     "A-12",
	})
	if err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	if order.Subtotal != "16,40 €" {
		t.Fatalf("expected subtotal \"16,40 €\", got %q", order.Subtotal)
	}
	if order.Shipping != "4,00 €" {
		t.Fatalf("expected shipping \"4,00 €\", got %q", order.Shipping)
	}
	if order.Total != "20,40 €" {
		t.Fatalf("expected total \"20,40 €\", got %q", order.Total)
	}
	if order.Status != shopmodels.OrderStatusPendingPayment {
		t.Fatalf("expected pending_payment, got %q", order.Status)
	}

	count, _ := cartRepo.CountByUser(context.Background(), userID)
	if count != 0 {
		t.Fatalf("expected cart to be cleared, still has %d items", count)
	}
}

func TestShop_PayTransitionsOrderOnce(t *testing.T) {
	svc, snackRepo, _, _, _ := newShopFixture()
	snack := addSnack(snackRepo, "Cola", 300)
	userID := uuid.New()

	if _, err := svc.AddCartItem(context.Background(), userID, &shopdto.AddCartItemRequest{
		ProductType: shopmodels.ProductTypeSnack,
		ProductID:   snack.ID,
		Quantity:    1,
	}); err != nil {
		t.Fatalf("add to cart failed: %v", err)
	}
	order, err := svc.Checkout(context.Background(), userID, &shopdto.CheckoutRequest{DeliveryMethod: "pickup"})
	if err != nil {
		t.Fatalf("Checkout failed: %v", err)
	}

	pay, err := svc.Pay(context.Background(), userID, order.OrderID, "card")
	if err != nil {
		t.Fatalf("Pay failed: %v", err)
	}
	if pay.Status != shopmodels.OrderStatusPaid {
		t.Fatalf("expected paid, got %q", pay.Status)
	}

	// Second payment attempt conflicts.
	_, err = svc.Pay(context.Background(), userID, order.OrderID, "card")
	if err == nil {
		t.Fatal("expected conflict for double payment")
	}
	if status := appErrorStatus(t, err); status != http.StatusConflict {
		t.Fatalf("expected 409, got %d", status)
	}
}

func TestShop_PayUnknownOrderNotFound(t *testing.T) {
	svc, _, _, _, _ := newShopFixture()

	_, err := svc.Pay(context.Background(), uuid.New(), uuid.New(), "card")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if status := appErrorStatus(t, err); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestShop_CartGroupsByProductType(t *testing.T) {
	svc, snackRepo, productRepo, _, _ := newShopFixture()
	snack := addSnack(snackRepo, "Chips", 250)
	product := addProduct(productRepo, "Scarf", 1500, `["M"]`)
	userID := uuid.New()

	for _, req := range []*shopdto.AddCartItemRequest{
		{ProductType: shopmodels.ProductTypeSnack, ProductID: snack.ID, Quantity: 1},
		{ProductType: shopmodels.ProductTypeMerch, ProductID: product.ID, Quantity: 1, Size: "M"},
	} {
		if _, err := svc.AddCartItem(context.Background(), userID, req); err != nil {
			t.Fatalf("add to cart failed: %v", err)
		}
	}

	cart, err := svc.GetCart(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetCart failed: %v", err)
	}
	if len(cart.Orders) != 2 {
		t.Fatalf("expected 2 cart groups, got %d", len(cart.Orders))
	}
}
