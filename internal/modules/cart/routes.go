package cart

import (
	shoprepo "clap/internal/modules/shop/repository"
	"clap/internal/modules/cart/handler"
	"clap/internal/modules/cart/repository"
	"clap/internal/modules/cart/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the mobile Cart endpoints (Mobile API Contract §6.3).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	cartSvc := service.NewCartService(
		repository.NewCartRepository(db),
		shoprepo.NewProductRepository(db),
		storageinit.Provider(),
	)
	h := handler.NewCartHandler(cartSvc)

	cartGroup := r.Group("/cart")
	cartGroup.Use(middleware.Auth())
	{
		cartGroup.GET("", h.GetCart)
		cartGroup.POST("/items", h.AddItem)
		cartGroup.PATCH("/items/:item_id", h.UpdateItem)
		cartGroup.DELETE("/items/:item_id", h.RemoveItem)
	}
}

// NewService exposes the cart service for other modules (e.g. shop, order).
func NewService() service.CartService {
	db := database.GetDB()
	return service.NewCartService(
		repository.NewCartRepository(db),
		shoprepo.NewProductRepository(db),
		storageinit.Provider(),
	)
}
