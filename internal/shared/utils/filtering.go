package utils

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Filter struct {
	Field    string
	Operator string
	Value    interface{}
}

type FilterOptions struct {
	AllowedFields map[string]bool
}

func ApplyFilters(db *gorm.DB, c *gin.Context, options FilterOptions) *gorm.DB {
	for field := range c.Request.URL.Query() {
		if !isFilterField(field) {
			continue
		}

		if !options.AllowedFields[field] {
			continue
		}

		value := c.Query(field)
		if value == "" {
			continue
		}

		db = applyFilter(db, field, value)
	}

	return db
}

func isFilterField(field string) bool {
	return !strings.HasPrefix(field, "page") &&
		!strings.HasPrefix(field, "sort") &&
		field != "search"
}

func applyFilter(db *gorm.DB, field, value string) *gorm.DB {
	if strings.HasSuffix(field, "_gte") {
		actualField := strings.TrimSuffix(field, "_gte")
		return db.Where(actualField+" >= ?", value)
	}

	if strings.HasSuffix(field, "_lte") {
		actualField := strings.TrimSuffix(field, "_lte")
		return db.Where(actualField+" <= ?", value)
	}

	if strings.HasSuffix(field, "_gt") {
		actualField := strings.TrimSuffix(field, "_gt")
		return db.Where(actualField+" > ?", value)
	}

	if strings.HasSuffix(field, "_lt") {
		actualField := strings.TrimSuffix(field, "_lt")
		return db.Where(actualField+" < ?", value)
	}

	if strings.HasSuffix(field, "_ne") {
		actualField := strings.TrimSuffix(field, "_ne")
		return db.Where(actualField+" != ?", value)
	}

	if strings.HasSuffix(field, "_like") {
		actualField := strings.TrimSuffix(field, "_like")
		return db.Where(actualField+" ILIKE ?", "%"+value+"%")
	}

	return db.Where(field+" = ?", value)
}
