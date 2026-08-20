package service

import "testing"

func TestParseTaxRate(t *testing.T) {
	t.Parallel()

	t.Run("required", func(t *testing.T) {
		t.Parallel()
		if _, err := parseTaxRate(nil); err == nil {
			t.Fatal("expected error when tax_rate is missing")
		}
	})

	t.Run("zero allowed", func(t *testing.T) {
		t.Parallel()
		rate := 0.0
		bps, err := parseTaxRate(&rate)
		if err != nil {
			t.Fatal(err)
		}
		if bps != 0 {
			t.Fatalf("got %d, want 0", bps)
		}
	})

	t.Run("standard vat", func(t *testing.T) {
		t.Parallel()
		rate := 19.0
		bps, err := parseTaxRate(&rate)
		if err != nil {
			t.Fatal(err)
		}
		if bps != 1900 {
			t.Fatalf("got %d, want 1900", bps)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		t.Parallel()
		rate := 120.0
		if _, err := parseTaxRate(&rate); err == nil {
			t.Fatal("expected error for tax_rate above 100")
		}
	})
}
