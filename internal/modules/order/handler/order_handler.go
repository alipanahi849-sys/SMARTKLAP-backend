package handler

import (
	"strings"

	"clap/internal/modules/order/dto"
	"clap/internal/modules/order/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OrderHandler serves checkout and payment endpoints.
type OrderHandler interface {
	List(c *gin.Context)
	GetByID(c *gin.Context)
	Calculate(c *gin.Context)
	Create(c *gin.Context)
	Confirm(c *gin.Context)
	Pay(c *gin.Context)
	StripeWebhook(c *gin.Context)
}

type orderHandler struct {
	svc service.OrderService
}

func NewOrderHandler(svc service.OrderService) OrderHandler {
	return &orderHandler{svc: svc}
}

// List orders godoc
//
//	@Summary		List user orders
//	@Description	Returns the authenticated user's orders with cursor pagination
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cursor	query	string	false	"Order ID of the last item from the previous page"
//	@Param			limit	query	int		false	"Items per page (default 20, max 100)"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/orders [get]
func (h *orderHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	limit := utils.GetMobileCursorLimit(c)

	var cursor *uuid.UUID
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "Invalid cursor")
			return
		}
		cursor = &id
	}

	result, err := h.svc.ListOrders(c.Request.Context(), userID, dto.OrderListFilters{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// GetByID order godoc
//
//	@Summary		Get order detail
//	@Description	Returns one order with all line items, images, and quantities
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			order_id	path	string	true	"Order ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/orders/{order_id} [get]
func (h *orderHandler) GetByID(c *gin.Context) {
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

	result, err := h.svc.GetOrder(c.Request.Context(), userID, orderID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Calculate order godoc
//
//	@Summary		Calculate checkout totals
//	@Description	Preview subtotal, shipping, total, and points required for the current cart
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	dto.CalculateOrderRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		422	{object}	response.Response
//	@Router			/api/v1/orders/calculate [post]
func (h *orderHandler) Calculate(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.CalculateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.CalculateOrder(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Create order godoc
//
//	@Summary		Create checkout order
//	@Description	Creates an order from the user's cart (Mobile API Contract §6.4)
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	dto.CreateOrderRequest	true	"Request body"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		422	{object}	response.Response
//	@Router			/api/v1/orders [post]
func (h *orderHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.CreateOrder(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, result)
}

// Pay order godoc
//
//	@Summary		Pay for an order
//	@Description	Completes payment with points or initiates Stripe card payment
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			order_id	path	string	true	"Order ID"
//	@Param			body		body	dto.PayOrderRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Failure		422	{object}	response.Response
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
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.PayOrder(c.Request.Context(), userID, orderID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Confirm card payment godoc
//
//	@Summary		Confirm browser card payment
//	@Description	Verifies Stripe Checkout Session status and marks the order paid
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			order_id	path	string	true	"Order ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		422	{object}	response.Response
//	@Router			/api/v1/orders/{order_id}/confirm-payment [post]
func (h *orderHandler) Confirm(c *gin.Context) {
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

	result, err := h.svc.ConfirmCardPayment(c.Request.Context(), userID, orderID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// StripeWebhook godoc
//
//	@Summary		Stripe webhook
//	@Description	Receives Stripe payment events and fulfills paid card orders
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/webhooks/stripe [post]
func (h *orderHandler) StripeWebhook(c *gin.Context) {
	payload, err := c.GetRawData()
	if err != nil {
		response.BadRequest(c, "Invalid webhook payload")
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	if err := h.svc.HandleStripeWebhook(c.Request.Context(), payload, signature); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"received": true})
}
