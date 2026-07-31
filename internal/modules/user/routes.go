package user

import (
	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/user/handler"
	"clap/internal/modules/user/repository"
	"clap/internal/modules/user/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	profileRepo := repository.NewProfileRepository()
	profileService := service.NewProfileService(profileRepo)

	// Mobile Profile screens (Mobile API Contract §2).
	mobileSvc := service.NewMobileProfileService(
		authrepo.NewUserRepository(),
		profileRepo,
		storageinit.Provider(),
	)
	mobileHandler := handler.NewMobileProfileHandler(mobileSvc)

	// /profiles/me aliases the same compact response for clients using the plural path.
	profileHandler := handler.NewProfileHandler(mobileSvc, profileService)

	profileGroup := r.Group("/profiles")
	{
		profileGroup.GET("/me", middleware.Auth(), profileHandler.GetProfile)
		profileGroup.POST("/me", middleware.Auth(), profileHandler.CreateProfile)
		profileGroup.PUT("/me", middleware.Auth(), profileHandler.UpdateProfile)
		profileGroup.PATCH("/me", middleware.Auth(), profileHandler.UpdateProfile)
		profileGroup.DELETE("/me", middleware.Auth(), profileHandler.DeleteProfile)
	}

	mobileGroup := r.Group("/profile")
	{
		mobileGroup.GET("/me", middleware.Auth(), mobileHandler.GetMe)
		mobileGroup.PATCH("/me", middleware.Auth(), mobileHandler.UpdateMe)
		mobileGroup.POST("/me/avatar", middleware.Auth(), mobileHandler.UploadAvatar)
		mobileGroup.GET("/leaderboard", middleware.Auth(), mobileHandler.Leaderboard)
	}
}
