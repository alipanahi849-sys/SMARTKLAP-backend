package utils

import "fmt"

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
