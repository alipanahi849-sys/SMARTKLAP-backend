package utils

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SortOptions struct {
	AllowedFields map[string]bool
	DefaultField  string
	DefaultOrder  string
}

func ApplySort(db *gorm.DB, c *gin.Context, options SortOptions) *gorm.DB {
	sortBy := c.Query("sort")
	order := c.Query("order")

	if sortBy == "" {
		sortBy = options.DefaultField
	}

	if order == "" {
		order = options.DefaultOrder
	}

	if !options.AllowedFields[sortBy] {
		sortBy = options.DefaultField
	}

	order = strings.ToUpper(order)
	if order != "ASC" && order != "DESC" {
		order = options.DefaultOrder
	}

	return db.Order(sortBy + " " + order)
}
