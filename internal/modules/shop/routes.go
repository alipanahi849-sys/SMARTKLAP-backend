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
	{
		// Catalog reads: app fans and admin dashboard.
		shop.GET("", middleware.AnyAuth(), productH.List)

		// Cart is fan-only (app users). Register before /:id.
		cart := shop.Group("/cart")
		cart.Use(middleware.Auth())
		{
			cart.POST("/items", cartH.AddItem)
			cart.POST("/items/decrease", cartH.DecreaseItem)
			cart.GET("", cartH.GetBasket)
		}

		// Product mutations: admin staff only.
		shop.POST("", middleware.AdminAuth(), productH.Create)
		shop.GET("/:id", middleware.AnyAuth(), productH.GetByID)
		shop.PUT("/:id", middleware.AdminAuth(), productH.Update)
		shop.DELETE("/:id", middleware.AdminAuth(), productH.Delete)
		shop.POST("/:id/image", middleware.AdminAuth(), productH.UploadProductImage)
	}
}
