package user

import (
	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/user/handler"
	"clap/internal/modules/user/repository"
	"clap/internal/modules/user/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/mediainit"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	profileRepo := repository.NewProfileRepository()
	profileService := service.NewProfileService(profileRepo)

	mobileSvc := service.NewMobileProfileServiceWithOptimizer(
		authrepo.NewUserRepository(),
		profileRepo,
		storageinit.Provider(),
		mediainit.Optimizer(),
	)
	mobileHandler := handler.NewMobileProfileHandler(mobileSvc)
	profileHandler := handler.NewProfileHandler(mobileSvc, profileService)

	profileGroup := r.Group("/profile")
	{
		profileGroup.GET("/me", middleware.Auth(), profileHandler.GetProfile)
		profileGroup.PATCH("/me", middleware.Auth(), mobileHandler.UpdateMe)
		profileGroup.DELETE("/me", middleware.Auth(), profileHandler.DeleteProfile)
		profileGroup.POST("/me/avatar", middleware.Auth(), mobileHandler.UploadAvatar)
		profileGroup.GET("/leaderboard", middleware.Auth(), mobileHandler.Leaderboard)
	}
}
