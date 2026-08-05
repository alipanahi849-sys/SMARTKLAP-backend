package handler

import (
	"clap/internal/modules/order/dto"
	"clap/internal/modules/order/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler interface {
	Create(c *gin.Context)
	Pay(c *gin.Context)
	Preview(c *gin.Context)
}

type orderHandler struct {
	svc service.OrderService
}

func NewOrderHandler(svc service.OrderService) OrderHandler {
	return &orderHandler{svc: svc}
}

// Create godoc
//
//	@Summary		Create order from cart
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	dto.CreateOrderRequest	true	"Checkout details"
//	@Success		201	{object}	response.Response
//	@Failure		422	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Router			/api/v1/orders [post]
func (h *orderHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	result, err := h.svc.CreateFromCart(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, result)
}

// Pay godoc
//
//	@Summary		Pay for order
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			order_id	path	string	true	"Order ID"
//	@Param			body	body	dto.PayOrderRequest	true	"Payment method"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/orders/{order_id}/pay [post]
func (h *orderHandler) Pay(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	orderID, err := uuid.Parse(c.Param("order_id"))
	if err != nil {
		response.BadRequest(c, "Invalid order_id")
		return
	}

	var req dto.PayOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	result, err := h.svc.Pay(c.Request.Context(), userID, orderID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// Preview godoc
//
//	@Summary		Preview checkout totals
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			delivery_method	query	string	false	"seat or pickup"
//	@Success		200	{object}	response.Response
//	@Failure		422	{object}	response.Response
//	@Router			/api/v1/orders/preview [get]
func (h *orderHandler) Preview(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	deliveryMethod := c.DefaultQuery("delivery_method", "seat")

	result, err := h.svc.GetCheckoutPreview(c.Request.Context(), userID, deliveryMethod)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}
