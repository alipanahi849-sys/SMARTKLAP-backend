package product

import (
	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/product/handler"
	"clap/internal/modules/product/repository"
	"clap/internal/modules/product/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the mobile Store endpoints (Mobile API Contract §7).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	productSvc := service.NewProductService(
		repository.NewProductRepository(db),
		authrepo.NewUserRepository(),
		storageinit.Provider(),
	)
	h := handler.NewProductHandler(productSvc)

	products := r.Group("/products")
	products.Use(middleware.Auth())
	{
		products.GET("", h.List)
	}
}
