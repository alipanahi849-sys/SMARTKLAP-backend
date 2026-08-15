package order

import (
	"context"

	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/order/handler"
	orderrepo "clap/internal/modules/order/repository"
	ordersvc "clap/internal/modules/order/service"
	shoprepo "clap/internal/modules/shop/repository"
	"clap/internal/shared/config"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/paymentinit"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires checkout and payment endpoints (Mobile API Contract §6.4).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	orderRepo := orderrepo.NewOrderRepository(db)
	cartRepo := shoprepo.NewCartRepository(db)
	productRepo := shoprepo.NewProductRepository(db)
	sizeStockRepo := shoprepo.NewProductSizeStockRepository(db)
	userRepo := authrepo.NewUserRepository()

	appURLScheme := "smartklap"
	if config.AppConfig != nil && config.AppConfig.Stripe.AppURLScheme != "" {
		appURLScheme = config.AppConfig.Stripe.AppURLScheme
	}

	svc := ordersvc.NewOrderService(
		orderRepo,
		cartRepo,
		productRepo,
		sizeStockRepo,
		userRepo,
		storageinit.Provider(),
		paymentinit.Provider(),
		appURLScheme,
	)
	h := handler.NewOrderHandler(svc)
	go svc.RunPendingPaymentSweeper(context.Background())

	r.POST("/webhooks/stripe", h.StripeWebhook)

	orders := r.Group("/orders")
	orders.Use(middleware.Auth())
	{
		orders.GET("", h.List)
		orders.POST("/calculate", h.Calculate)
		orders.POST("", h.Create)
		orders.GET("/:order_id", h.GetByID)
		orders.PATCH("/:order_id", h.Update)
		orders.POST("/:order_id/pay", h.Pay)
		orders.POST("/:order_id/confirm-payment", h.Confirm)
	}
}
