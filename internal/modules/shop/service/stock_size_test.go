package service

import (
	"encoding/json"
	"testing"

	"clap/internal/modules/shop/dto"
	"clap/internal/modules/shop/models"
)

func TestBuildProductStockInfo_BySize(t *testing.T) {
	t.Parallel()

	qtyM := 5
	qtyL := 0
	sizeStocks := []models.ProductSizeStock{
		{Size: "M", StockQuantity: &qtyM},
		{Size: "L", StockQuantity: &qtyL, SoldOut: true},
	}
	product := &models.Product{
		ProductType:    models.ProductTypeMerch,
		AvailableSizes: `["M","L"]`,
	}

	info := buildProductStockInfo(product, sizeStocks, "")
	if !info.InStock {
		t.Fatal("expected aggregate in_stock true when one size available")
	}
	if len(info.BySize) != 2 {
		t.Fatalf("expected 2 by_size entries, got %d", len(info.BySize))
	}
	if !info.BySize[0].InStock || info.BySize[1].InStock {
		t.Fatal("expected only M in stock")
	}
	if !info.BySize[1].SoldOut {
		t.Fatal("expected L soldout")
	}

	filtered := buildProductStockInfo(product, sizeStocks, "L")
	if filtered.InStock {
		t.Fatal("expected filtered size L out of stock")
	}
	if !filtered.SoldOut {
		t.Fatal("expected filtered soldout true")
	}

	raw, err := json.Marshal(info.BySize[1])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !jsonContainsField(raw, "soldout") {
		t.Fatalf("expected soldout in size JSON, got %s", raw)
	}
}

func TestResolveProductStockInputs(t *testing.T) {
	t.Parallel()

	t.Run("sized merch requires size_stock", func(t *testing.T) {
		t.Parallel()
		_, _, err := resolveProductStockInputs(
			models.ProductTypeMerch,
			[]string{"M", "L"},
			nil,
			nil,
		)
		if err == nil {
			t.Fatal("expected error when size_stock missing")
		}
	})

	t.Run("sized merch rejects product stock_quantity", func(t *testing.T) {
		t.Parallel()
		n := 10
		_, _, err := resolveProductStockInputs(
			models.ProductTypeMerch,
			[]string{"M"},
			&n,
			[]dto.SizeStockInput{{Size: "M", StockQuantity: &n}},
		)
		if err == nil {
			t.Fatal("expected error when stock_quantity set with sizes")
		}
	})

	t.Run("unsized merch uses stock_quantity", func(t *testing.T) {
		t.Parallel()
		n := 12
		qty, rows, err := resolveProductStockInputs(models.ProductTypeMerch, nil, &n, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if qty == nil || *qty != 12 || len(rows) != 0 {
			t.Fatalf("got qty=%v rows=%d", qty, len(rows))
		}
	})
}
