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

	mobileSvc := service.NewMobileProfileService(
		authrepo.NewUserRepository(),
		profileRepo,
		storageinit.Provider(),
	)
	mobileHandler := handler.NewMobileProfileHandler(mobileSvc)
	profileHandler := handler.NewProfileHandler(mobileSvc, profileService)

	// Basic profile: get + delete only.
	profileGroup := r.Group("/profile")
	{
		profileGroup.GET("/me", middleware.Auth(), profileHandler.GetProfile)
		profileGroup.DELETE("/me", middleware.Auth(), profileHandler.DeleteProfile)

		// Mobile subset nested under /profile/mobile.
		mobileGroup := profileGroup.Group("/mobile")
		{
			mobileGroup.GET("/me", middleware.Auth(), mobileHandler.GetMe)
			mobileGroup.PATCH("/me", middleware.Auth(), mobileHandler.UpdateMe)
			mobileGroup.POST("/me/avatar", middleware.Auth(), mobileHandler.UploadAvatar)
			mobileGroup.GET("/leaderboard", middleware.Auth(), mobileHandler.Leaderboard)
		}
	}
}
