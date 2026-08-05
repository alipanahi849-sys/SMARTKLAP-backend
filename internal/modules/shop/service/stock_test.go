package service

import (
	"encoding/json"
	"testing"

	"clap/internal/modules/shop/models"
)

func TestToProductStockInfo_SoldOut(t *testing.T) {
	t.Parallel()

	depleted := 0
	info := toProductStockInfo(&models.Product{
		StockQuantity: &depleted,
		SoldOut:       true,
	})
	if !info.SoldOut {
		t.Fatal("expected soldout true when stock depleted by order")
	}
	if info.InStock {
		t.Fatal("expected in_stock false")
	}

	raw, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !jsonContainsField(raw, "soldout") {
		t.Fatalf("expected soldout in JSON, got %s", raw)
	}

	adminZero := 0
	adminInfo := toProductStockInfo(&models.Product{
		StockQuantity: &adminZero,
		SoldOut:       false,
	})
	if adminInfo.SoldOut {
		t.Fatal("admin zero stock should not set soldout")
	}
	adminRaw, err := json.Marshal(adminInfo)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if jsonContainsField(adminRaw, "soldout") {
		t.Fatalf("soldout should be omitted when false, got %s", adminRaw)
	}
}

func jsonContainsField(raw []byte, field string) bool {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m[field]
	return ok
}
