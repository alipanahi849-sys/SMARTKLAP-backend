package user

import (
	authmodels "clap/internal/modules/auth/models"
	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/user/handler"
	"clap/internal/modules/user/repository"
	"clap/internal/modules/user/service"
	"clap/internal/shared/mediainit"
	"clap/internal/shared/middleware"
	"clap/internal/shared/storageinit"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	profileRepo := repository.NewProfileRepository()
	profileService := service.NewProfileService(profileRepo)

	userRepo := authrepo.NewUserRepository()
	roleRepo := authrepo.NewRoleRepository()

	mobileSvc := service.NewMobileProfileServiceWithOptimizer(
		userRepo,
		profileRepo,
		storageinit.Provider(),
		mediainit.Optimizer(),
	)
	mobileHandler := handler.NewMobileProfileHandler(mobileSvc)
	profileHandler := handler.NewProfileHandler(mobileSvc, profileService)
	adminUsers := handler.NewAdminUserHandler(service.NewAdminUserService(userRepo, roleRepo))

	profileGroup := r.Group("/profile")
	{
		profileGroup.GET("/me", middleware.Auth(), profileHandler.GetProfile)
		profileGroup.PATCH("/me", middleware.Auth(), mobileHandler.UpdateMe)
		profileGroup.DELETE("/me", middleware.Auth(), profileHandler.DeleteProfile)
		profileGroup.POST("/me/avatar", middleware.Auth(), mobileHandler.UploadAvatar)
		profileGroup.GET("/leaderboard", middleware.Auth(), mobileHandler.Leaderboard)
	}

	admin := r.Group("/admin")
	admin.Use(middleware.AdminAuth(), middleware.RequirePermission(authmodels.PanelUsers))
	{
		admin.GET("/users", adminUsers.List)
		admin.POST("/users", adminUsers.Create)
		admin.GET("/users/:id", adminUsers.Get)
		admin.PATCH("/users/:id", adminUsers.Update)
	}
}
