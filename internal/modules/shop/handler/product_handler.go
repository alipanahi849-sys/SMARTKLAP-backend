package handler

import (
	"clap/internal/modules/shop/dto"
	"clap/internal/modules/shop/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProductHandler serves the mobile Store screens (contract §7).
type ProductHandler interface {
	List(c *gin.Context)
}

type productHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) ProductHandler {
	return &productHandler{svc: svc}
}

// List shop products godoc
//
//	@Summary		List shop products
//	@Tags			shop
//	@Produce		json
//	@Security		BearerAuth
//	@Param			search		query	string	false	"Search by name, subname or description"
//	@Param			category	query	string	false	"Category filter (t-shirts, balls, stickers, sport-suits)"
//	@Param			currency	query	string	false	"Price display currency (EUR or POINT)"
//	@Param			page		query	int		false	"Page number"
//	@Param			limit		query	int		false	"Items per page"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/shop [get]
func (h *productHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	page, limit := utils.GetMobilePagination(c)
	filters := dto.ProductListFilters{
		Search:   c.Query("search"),
		Category: c.Query("category"),
		Currency: c.DefaultQuery("currency", dto.CurrencyEUR),
	}

	result, err := h.svc.List(c.Request.Context(), userID, page, limit, filters)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
