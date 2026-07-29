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
