package handler

import (
	"clap/internal/modules/cart/dto"
	"clap/internal/modules/cart/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CartHandler interface {
	GetCart(c *gin.Context)
	AddItem(c *gin.Context)
	UpdateItem(c *gin.Context)
	RemoveItem(c *gin.Context)
}

type cartHandler struct {
	svc service.CartService
}

func NewCartHandler(svc service.CartService) CartHandler {
	return &cartHandler{svc: svc}
}

// GetCart godoc
//
//	@Summary		Get shopping cart
//	@Tags			cart
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Router			/api/v1/cart [get]
func (h *cartHandler) GetCart(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	result, err := h.svc.GetCart(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// AddItem godoc
//
//	@Summary		Add item to cart
//	@Tags			cart
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	dto.AddCartItemRequest	true	"Cart item"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Router			/api/v1/cart/items [post]
func (h *cartHandler) AddItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	result, err := h.svc.AddItem(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// UpdateItem godoc
//
//	@Summary		Update cart item quantity
//	@Tags			cart
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			item_id	path	string	true	"Cart item ID"
//	@Param			body	body	dto.UpdateCartItemRequest	true	"Quantity"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/cart/items/{item_id} [patch]
func (h *cartHandler) UpdateItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		response.BadRequest(c, "Invalid item_id")
		return
	}

	var req dto.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	result, err := h.svc.UpdateItem(c.Request.Context(), userID, itemID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// RemoveItem godoc
//
//	@Summary		Remove cart item
//	@Tags			cart
//	@Produce		json
//	@Security		BearerAuth
//	@Param			item_id	path	string	true	"Cart item ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/cart/items/{item_id} [delete]
func (h *cartHandler) RemoveItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		response.BadRequest(c, "Invalid item_id")
		return
	}

	result, err := h.svc.RemoveItem(c.Request.Context(), userID, itemID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}
