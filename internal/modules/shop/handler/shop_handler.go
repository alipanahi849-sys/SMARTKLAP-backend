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

// ShopHandler serves Snacks, Store, Cart and Orders (contract §6–§7).
type ShopHandler interface {
	ListSnacks(c *gin.Context)
	SnackDetail(c *gin.Context)
	ListProducts(c *gin.Context)
	ProductDetail(c *gin.Context)
	GetCart(c *gin.Context)
	AddCartItem(c *gin.Context)
	UpdateCartItem(c *gin.Context)
	RemoveCartItem(c *gin.Context)
	Checkout(c *gin.Context)
	Pay(c *gin.Context)
}

type shopHandler struct {
	svc service.ShopService
}

func NewShopHandler(svc service.ShopService) ShopHandler {
	return &shopHandler{svc: svc}
}

// List snacks godoc
//
//	@Summary		List snacks
//	@Tags			shop
//	@Produce		json
//	@Security		BearerAuth
//	@Param			search	query	string	false	"Search"
//	@Param			category	query	string	false	"Category"
//	@Param			currency	query	string	false	"Currency"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/snacks [get]
func (h *shopHandler) ListSnacks(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	page, limit := utils.GetMobilePagination(c)
	result, err := h.svc.ListSnacks(
		c.Request.Context(), userID,
		c.Query("search"), c.Query("category"), c.Query("currency"),
		page, limit,
	)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Snack detail godoc
//
//	@Summary		Snack detail
//	@Tags			shop
//	@Produce		json
//	@Security		BearerAuth
//	@Param			snack_id	path	string	true	"Snack ID"
//	@Param			currency	query	string	false	"Currency"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/snacks/{snack_id} [get]
func (h *shopHandler) SnackDetail(c *gin.Context) {
	snackID, err := uuid.Parse(c.Param("snack_id"))
	if err != nil {
		response.BadRequest(c, "Invalid snack ID")
		return
	}

	result, svcErr := h.svc.SnackDetail(c.Request.Context(), snackID, c.Query("currency"))
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}

// List products godoc
//
//	@Summary		List products
//	@Tags			shop
//	@Produce		json
//	@Security		BearerAuth
//	@Param			search	query	string	false	"Search"
//	@Param			category	query	string	false	"Category"
//	@Param			currency	query	string	false	"Currency"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/products [get]
func (h *shopHandler) ListProducts(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	page, limit := utils.GetMobilePagination(c)
	result, err := h.svc.ListProducts(
		c.Request.Context(), userID,
		c.Query("search"), c.Query("category"), c.Query("currency"),
		page, limit,
	)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Product detail godoc
//
//	@Summary		Product detail
//	@Tags			shop
//	@Produce		json
//	@Security		BearerAuth
//	@Param			product_id	path	string	true	"Product ID"
//	@Param			currency	query	string	false	"Currency"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/products/{product_id} [get]
func (h *shopHandler) ProductDetail(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		response.BadRequest(c, "Invalid product ID")
		return
	}

	result, svcErr := h.svc.ProductDetail(c.Request.Context(), productID, c.Query("currency"))
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}

// Get cart godoc
//
//	@Summary		Get cart
//	@Tags			shop
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/cart [get]
func (h *shopHandler) GetCart(c *gin.Context) {
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

// Add cart item godoc
//
//	@Summary		Add cart item
//	@Tags			shop
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.AddCartItemRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/cart/items [post]
func (h *shopHandler) AddCartItem(c *gin.Context) {
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

	result, err := h.svc.AddCartItem(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, result)
}

// Update cart item godoc
//
//	@Summary		Update cart item
//	@Tags			shop
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			item_id	path	string	true	"Cart item ID"
//	@Param			body	body		dto.UpdateCartItemRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/cart/items/{item_id} [patch]
func (h *shopHandler) UpdateCartItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		response.BadRequest(c, "Invalid cart item ID")
		return
	}

	var req dto.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, svcErr := h.svc.UpdateCartItem(c.Request.Context(), userID, itemID, req.Quantity)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}

// Remove cart item godoc
//
//	@Summary		Remove cart item
//	@Tags			shop
//	@Produce		json
//	@Security		BearerAuth
//	@Param			item_id	path	string	true	"Cart item ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/cart/items/{item_id} [delete]
func (h *shopHandler) RemoveCartItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	itemID, err := uuid.Parse(c.Param("item_id"))
	if err != nil {
		response.BadRequest(c, "Invalid cart item ID")
		return
	}

	if svcErr := h.svc.RemoveCartItem(c.Request.Context(), userID, itemID); svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.NoContent(c)
}

// Checkout cart godoc
//
//	@Summary		Checkout cart
//	@Tags			shop
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CheckoutRequest	true	"Request body"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/orders [post]
func (h *shopHandler) Checkout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.Checkout(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, result)
}

// Pay order godoc
//
//	@Summary		Pay order
//	@Tags			shop
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			order_id	path	string	true	"Order ID"
//	@Param			body	body		dto.PayOrderRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/orders/{order_id}/pay [post]
func (h *shopHandler) Pay(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	var req dto.PayOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, svcErr := h.svc.Pay(c.Request.Context(), userID, orderID, req.PaymentMethod)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}
