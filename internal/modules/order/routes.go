package order

import (
	"clap/internal/modules/cart"
	cartservice "clap/internal/modules/cart/service"
	"clap/internal/modules/order/handler"
	"clap/internal/modules/order/repository"
	"clap/internal/modules/order/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the mobile Order endpoints (Mobile API Contract §6.4).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()
	cartSvc := cart.NewService()

	orderSvc := service.NewOrderService(
		repository.NewOrderRepository(db),
		cartSvc,
		storageinit.Provider(),
	)
	h := handler.NewOrderHandler(orderSvc)

	orders := r.Group("/orders")
	orders.Use(middleware.Auth())
	{
		orders.GET("/preview", h.Preview)
		orders.POST("", h.Create)
		orders.POST("/:order_id/pay", h.Pay)
	}
}

// NewService exposes the order service for testing or other modules.
func NewService(cartSvc cartservice.CartService) service.OrderService {
	db := database.GetDB()
	return service.NewOrderService(
		repository.NewOrderRepository(db),
		cartSvc,
		storageinit.Provider(),
	)
}
