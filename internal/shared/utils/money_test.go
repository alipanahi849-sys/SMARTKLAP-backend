package utils

import "testing"

func TestTaxInclusiveCents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		net       int64
		bps       int
		wantGross int64
		wantTax   int64
	}{
		{name: "zero rate", net: 820, bps: 0, wantGross: 820, wantTax: 0},
		{name: "standard 19%", net: 1000, bps: 1900, wantGross: 1190, wantTax: 190},
		{name: "reduced 7%", net: 820, bps: 700, wantGross: 877, wantTax: 57},
		{name: "round half up", net: 333, bps: 1900, wantGross: 396, wantTax: 63},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := TaxInclusiveCents(tt.net, tt.bps); got != tt.wantGross {
				t.Fatalf("TaxInclusiveCents() = %d, want %d", got, tt.wantGross)
			}
			if got := TaxAmountCents(tt.net, tt.bps); got != tt.wantTax {
				t.Fatalf("TaxAmountCents() = %d, want %d", got, tt.wantTax)
			}
		})
	}
}

func TestTaxRateBpsFromPercent(t *testing.T) {
	t.Parallel()

	bps, err := TaxRateBpsFromPercent(19)
	if err != nil {
		t.Fatal(err)
	}
	if bps != 1900 {
		t.Fatalf("got %d, want 1900", bps)
	}
	if TaxRatePercent(bps) != 19 {
		t.Fatalf("percent round-trip = %v", TaxRatePercent(bps))
	}

	if _, err := TaxRateBpsFromPercent(-1); err == nil {
		t.Fatal("expected error for negative rate")
	}
	if _, err := TaxRateBpsFromPercent(101); err == nil {
		t.Fatal("expected error for rate above 100")
	}
}

func TestDiscountedAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		amount int64
		bps    int
		want   int64
	}{
		{name: "no discount", amount: 820, bps: 0, want: 820},
		{name: "10 percent", amount: 1000, bps: 1000, want: 900},
		{name: "20 percent with tax-size cents", amount: 3250, bps: 2000, want: 2600},
		{name: "round half up", amount: 333, bps: 1500, want: 283},
		{name: "full discount", amount: 820, bps: 10000, want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DiscountedAmount(tt.amount, tt.bps); got != tt.want {
				t.Fatalf("DiscountedAmount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDiscountRateBpsFromPercent(t *testing.T) {
	t.Parallel()

	bps, err := DiscountRateBpsFromPercent(12.5)
	if err != nil {
		t.Fatal(err)
	}
	if bps != 1250 {
		t.Fatalf("got %d, want 1250", bps)
	}
	if TaxRatePercent(bps) != 12.5 {
		t.Fatalf("percent round-trip = %v", TaxRatePercent(bps))
	}
	if _, err := DiscountRateBpsFromPercent(101); err == nil {
		t.Fatal("expected error for rate above 100")
	}
}
