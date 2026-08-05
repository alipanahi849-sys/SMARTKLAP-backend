package cart

import (
	"clap/internal/modules/cart/handler"
	cartrepo "clap/internal/modules/cart/repository"
	cartservice "clap/internal/modules/cart/service"
	shoprepo "clap/internal/modules/shop/repository"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires cart endpoints (Mobile API Contract §6.3).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	svc := cartservice.NewCartService(
		cartrepo.NewCartRepository(db),
		shoprepo.NewProductRepository(db),
	)
	h := handler.NewCartHandler(svc)

	group := r.Group("/cart")
	group.Use(middleware.Auth())
	{
		group.POST("/items", h.AddItem)
		group.POST("/items/decrease", h.DecreaseItem)
	}
}

// NewService exposes a wired CartService for cross-module composition.
func NewService() cartservice.CartService {
	db := database.GetDB()
	return cartservice.NewCartService(
		cartrepo.NewCartRepository(db),
		shoprepo.NewProductRepository(db),
	)
}
