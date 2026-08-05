package handler

import (
	"clap/internal/modules/shop/dto"
	"clap/internal/modules/shop/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CartHandler serves shop cart add/decrease endpoints.
type CartHandler interface {
	AddItem(c *gin.Context)
	DecreaseItem(c *gin.Context)
}

type cartHandler struct {
	svc service.CartService
}

func NewCartHandler(svc service.CartService) CartHandler {
	return &cartHandler{svc: svc}
}

// Add cart item godoc
//
//	@Summary		Add item to shop cart
//	@Description	Increments cart quantity without changing product stock
//	@Tags			shop
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	dto.AddCartItemRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Failure		422	{object}	response.Response
//	@Router			/api/v1/shop/cart/items [post]
func (h *cartHandler) AddItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.AddItem(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Decrease cart item godoc
//
//	@Summary		Decrease item in shop cart
//	@Description	Decrements cart quantity without changing product stock
//	@Tags			shop
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	dto.DecreaseCartItemRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/shop/cart/items/decrease [post]
func (h *cartHandler) DecreaseItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.DecreaseCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.DecreaseItem(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
