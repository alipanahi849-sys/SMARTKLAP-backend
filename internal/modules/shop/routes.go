package shop

import (
	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/shop/handler"
	"clap/internal/modules/shop/repository"
	"clap/internal/modules/shop/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the mobile Shop endpoints (Mobile API Contract §7).
func RegisterRoutes(r *gin.RouterGroup) {
	db := database.GetDB()

	shopSvc := service.NewProductService(
		repository.NewProductRepository(db),
		authrepo.NewUserRepository(),
		storageinit.Provider(),
	)
	h := handler.NewProductHandler(shopSvc)

	shop := r.Group("/shop")
	shop.Use(middleware.Auth())
	{
		shop.GET("", h.List)
		shop.POST("", h.Create)
		shop.GET("/:id", h.GetByID)
		shop.PUT("/:id", h.Update)
		shop.DELETE("/:id", h.Delete)
		shop.POST("/:id/image", h.UploadProductImage)
	}
}
