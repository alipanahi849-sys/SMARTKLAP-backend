package utils

import (
	"fmt"
	"math"

	"github.com/gin-gonic/gin"
)

type PaginationRequest struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type PaginationResponse struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

func GetPagination(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20

	if p, ok := c.GetQuery("page"); ok {
		if parsed := parseInt(p); parsed > 0 {
			page = parsed
		}
	}

	if ps, ok := c.GetQuery("page_size"); ok {
		if parsed := parseInt(ps); parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	return page, pageSize
}

func CalculatePagination(total int64, page, pageSize int) PaginationResponse {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	return PaginationResponse{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

func GetOffset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// GetMobilePagination reads ?page= and ?limit= as defined by the Mobile API
// Contract (page defaults to 1, limit defaults to 20, capped at 100).
func GetMobilePagination(c *gin.Context) (int, int) {
	page := 1
	limit := 20

	if p, ok := c.GetQuery("page"); ok {
		if parsed := parseInt(p); parsed > 0 {
			page = parsed
		}
	}

	if l, ok := c.GetQuery("limit"); ok {
		if parsed := parseInt(l); parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	return page, limit
}

func parseInt(s string) int {
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err == nil {
		return i
	}
	return 0
}
