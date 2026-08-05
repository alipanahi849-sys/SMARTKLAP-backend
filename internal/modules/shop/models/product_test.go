package models

import "testing"

func intPtr(v int) *int { return &v }

func TestProduct_IsSoldOut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		product Product
		want    bool
	}{
		{
			name: "unlimited stock",
			product: Product{
				StockQuantity: nil,
				SoldOut:       true,
			},
			want: false,
		},
		{
			name: "in stock limited",
			product: Product{
				StockQuantity: intPtr(5),
				SoldOut:       false,
			},
			want: false,
		},
		{
			name: "admin zero stock without sold_out flag",
			product: Product{
				StockQuantity: intPtr(0),
				SoldOut:       false,
			},
			want: false,
		},
		{
			name: "depleted by order",
			product: Product{
				StockQuantity: intPtr(0),
				SoldOut:       true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.product.IsSoldOut(); got != tt.want {
				t.Fatalf("IsSoldOut() = %v, want %v", got, tt.want)
			}
		})
	}
}
