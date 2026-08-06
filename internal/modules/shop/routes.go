package shop

import (
	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/shop/handler"
	"clap/internal/modules/shop/repository"
	"clap/internal/modules/shop/service"
	"clap/internal/shared/database"
	"clap/internal/shared/mediainit"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the mobile Shop endpoints (Mobile API Contract §6–§7).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	productRepo := repository.NewProductRepository(db)
	sizeStockRepo := repository.NewProductSizeStockRepository(db)
	cartRepo := repository.NewCartRepository(db)
	cartSvc := service.NewCartService(cartRepo, productRepo, sizeStockRepo, storageinit.Provider())

	shopSvc := service.NewProductServiceWithOptimizer(
		productRepo,
		sizeStockRepo,
		authrepo.NewUserRepository(),
		storageinit.Provider(),
		cartSvc,
		mediainit.Optimizer(),
	)

	productH := handler.NewProductHandler(shopSvc)
	cartH := handler.NewCartHandler(cartSvc)

	shop := r.Group("/shop")
	shop.Use(middleware.Auth())
	{
		shop.GET("", productH.List)
		shop.POST("", productH.Create)

		shop.POST("/cart/items", cartH.AddItem)
		shop.POST("/cart/items/decrease", cartH.DecreaseItem)
		shop.GET("/cart", cartH.GetBasket)

		shop.GET("/:id", productH.GetByID)
		shop.PUT("/:id", productH.Update)
		shop.DELETE("/:id", productH.Delete)
		shop.POST("/:id/image", productH.UploadProductImage)
	}
}
