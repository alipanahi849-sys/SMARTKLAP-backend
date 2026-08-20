package utils

import (
	"fmt"
	"math"
)

// FormatEuro renders an amount in cents using the mobile contract's European
// display format, e.g. 820 → "8,20 €".
func FormatEuro(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d,%02d €", sign, cents/100, cents%100)
}

// FormatPoints renders a points-denominated price, e.g. 820 → "820 P".
func FormatPoints(points int) string {
	return fmt.Sprintf("%d P", points)
}

const taxRateBpsScale = 10000

// TaxInclusiveCents adds VAT to a net amount in cents, rounding half up.
func TaxInclusiveCents(netCents int64, taxRateBps int) int64 {
	if taxRateBps <= 0 {
		return netCents
	}
	return (netCents*int64(taxRateBpsScale+taxRateBps) + 5000) / taxRateBpsScale
}

// TaxAmountCents is the VAT portion of a net amount.
func TaxAmountCents(netCents int64, taxRateBps int) int64 {
	return TaxInclusiveCents(netCents, taxRateBps) - netCents
}

// DiscountedAmount applies a percent-off in basis points, rounding half up.
func DiscountedAmount(amount int64, discountBps int) int64 {
	if discountBps <= 0 {
		return amount
	}
	if discountBps >= taxRateBpsScale {
		return 0
	}
	return (amount*int64(taxRateBpsScale-discountBps) + 5000) / taxRateBpsScale
}

// DiscountedPoints applies a percent-off to a points price.
func DiscountedPoints(points int, discountBps int) int {
	return int(DiscountedAmount(int64(points), discountBps))
}

func rateBpsFromPercent(rate float64, field string) (int, error) {
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		return 0, fmt.Errorf("%s must be between 0 and 100", field)
	}
	if rate < 0 || rate > 100 {
		return 0, fmt.Errorf("%s must be between 0 and 100", field)
	}
	bps := int(math.Round(rate * 100))
	if bps < 0 || bps > taxRateBpsScale {
		return 0, fmt.Errorf("%s must be between 0 and 100", field)
	}
	return bps, nil
}

// TaxRateBpsFromPercent converts a percentage (e.g. 19 or 7.5) to basis points.
func TaxRateBpsFromPercent(rate float64) (int, error) {
	return rateBpsFromPercent(rate, "tax_rate")
}

// DiscountRateBpsFromPercent converts a percentage off (e.g. 10 or 12.5) to basis points.
func DiscountRateBpsFromPercent(rate float64) (int, error) {
	return rateBpsFromPercent(rate, "discount_rate")
}

// TaxRatePercent converts stored basis points to a percentage.
func TaxRatePercent(taxRateBps int) float64 {
	return float64(taxRateBps) / 100
}

// ListMeta is the pagination meta object defined by the Mobile API Contract:
// { "page", "limit", "total", "total_pages" }.
type ListMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// NewListMeta builds a ListMeta from a total row count and page window.
func NewListMeta(total int64, page, limit int) ListMeta {
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}
	return ListMeta{Page: page, Limit: limit, Total: total, TotalPages: totalPages}
}
