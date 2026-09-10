package adminauth

import (
	"clap/internal/modules/adminauth/handler"
	"clap/internal/modules/adminauth/repository"
	"clap/internal/modules/adminauth/service"
	authsvc "clap/internal/modules/auth/service"
	"clap/internal/shared/config"
	"clap/internal/shared/middleware"
	"clap/internal/shared/redis"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	adminRepo := repository.NewAdminUserRepository()
	refreshRepo := repository.NewAdminRefreshTokenRepository()

	var otpStore authsvc.OTPStore
	if redis.GetClient() != nil {
		otpStore = authsvc.NewRedisOTPStore()
	} else {
		otpStore = authsvc.NewMemoryOTPStore()
	}
	otpSender := authsvc.NewOTPSenderFromConfig(config.AppConfig)

	svc := service.NewAdminAuthService(adminRepo, refreshRepo, otpStore, otpSender)
	h := handler.NewAdminAuthHandler(svc)

	authGroup := r.Group("/admin/auth")
	{
		authGroup.POST("/login", middleware.AuthRateLimit(), h.Login)
		authGroup.POST("/verify-otp", middleware.AuthRateLimit(), h.VerifyOTP)
		authGroup.POST("/refresh", middleware.AuthRateLimit(), h.RefreshToken)

		authed := authGroup.Group("")
		authed.Use(middleware.AdminAuth())
		{
			authed.GET("/me", h.Me)
			authed.PATCH("/me", h.UpdateMe)
			authed.POST("/change-email", middleware.AuthRateLimit(), h.RequestChangeEmail)
			authed.POST("/verify-change-email", middleware.AuthRateLimit(), h.VerifyChangeEmail)
		}
	}
}
