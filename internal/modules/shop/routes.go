package shop

import (
	"clap/internal/modules/shop/handler"
	"clap/internal/modules/shop/repository"
	"clap/internal/modules/shop/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires Snacks, Store, Cart and Orders (Mobile API Contract
// §6 and §7).
func RegisterRoutes(r *gin.RouterGroup) {
	h := handler.NewShopHandler(NewService())

	snacks := r.Group("/snacks")
	snacks.Use(middleware.Auth())
	{
		snacks.GET("", h.ListSnacks)
		snacks.GET("/:snack_id", h.SnackDetail)
	}

	products := r.Group("/products")
	products.Use(middleware.Auth())
	{
		products.GET("", h.ListProducts)
		products.GET("/:product_id", h.ProductDetail)
	}

	cart := r.Group("/cart")
	cart.Use(middleware.Auth())
	{
		cart.GET("", h.GetCart)
		cart.POST("/items", h.AddCartItem)
		cart.PATCH("/items/:item_id", h.UpdateCartItem)
		cart.DELETE("/items/:item_id", h.RemoveCartItem)
	}

	orders := r.Group("/orders")
	orders.Use(middleware.Auth())
	{
		orders.POST("", h.Checkout)
		orders.POST("/:order_id/pay", h.Pay)
	}
}

// NewService exposes a fully wired ShopService for cross-module composition
// (the mobile Home aggregate reuses it).
func NewService() service.ShopService {
	db := database.GetDB()
	return service.NewShopService(
		repository.NewSnackRepository(db),
		repository.NewProductRepository(db),
		repository.NewCartRepository(db),
		repository.NewOrderRepository(db),
	)
}
