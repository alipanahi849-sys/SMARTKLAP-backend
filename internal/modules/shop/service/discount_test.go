package service

import "testing"

func TestParseDiscountRate(t *testing.T) {
	t.Parallel()

	t.Run("optional defaults to zero", func(t *testing.T) {
		t.Parallel()
		bps, err := parseDiscountRate(nil)
		if err != nil {
			t.Fatal(err)
		}
		if bps != 0 {
			t.Fatalf("got %d, want 0", bps)
		}
	})

	t.Run("zero allowed", func(t *testing.T) {
		t.Parallel()
		rate := 0.0
		bps, err := parseDiscountRate(&rate)
		if err != nil {
			t.Fatal(err)
		}
		if bps != 0 {
			t.Fatalf("got %d, want 0", bps)
		}
	})

	t.Run("percent off", func(t *testing.T) {
		t.Parallel()
		rate := 20.0
		bps, err := parseDiscountRate(&rate)
		if err != nil {
			t.Fatal(err)
		}
		if bps != 2000 {
			t.Fatalf("got %d, want 2000", bps)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		t.Parallel()
		rate := 120.0
		if _, err := parseDiscountRate(&rate); err == nil {
			t.Fatal("expected error for discount_rate above 100")
		}
	})
}
