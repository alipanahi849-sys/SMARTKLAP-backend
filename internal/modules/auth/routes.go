package auth

import (
	"clap/internal/modules/auth/handler"
	"clap/internal/modules/auth/repository"
	"clap/internal/modules/auth/service"
	"clap/internal/shared/config"
	"clap/internal/shared/middleware"
	"clap/internal/shared/redis"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	userRepo := repository.NewUserRepository()
	roleRepo := repository.NewRoleRepository()
	refreshTokenRepo := repository.NewRefreshTokenRepository()

	// OTP codes live in Redis when available (shared across instances);
	// otherwise fall back to the process-local store.
	var otpStore service.OTPStore
	if redis.GetClient() != nil {
		otpStore = service.NewRedisOTPStore()
	} else {
		otpStore = service.NewMemoryOTPStore()
	}
	otpSender := service.NewOTPSenderFromConfig(config.AppConfig)

	authService := service.NewAuthServiceWithOTP(userRepo, roleRepo, refreshTokenRepo, otpStore, otpSender)
	authHandler := handler.NewAuthHandler(authService)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", middleware.AuthRateLimit(), authHandler.Register)
		authGroup.POST("/login", middleware.AuthRateLimit(), authHandler.Login)
		authGroup.POST("/verify-otp", middleware.AuthRateLimit(), authHandler.VerifyOTP)
		authGroup.POST("/refresh", middleware.AuthRateLimit(), authHandler.RefreshToken)

		// Authenticated email change (OTP delivered to the new address).
		authGroup.POST("/change-email", middleware.Auth(), middleware.AuthRateLimit(), authHandler.RequestChangeEmail)
		authGroup.POST("/verify-change-email", middleware.Auth(), middleware.AuthRateLimit(), authHandler.VerifyChangeEmail)
	}
}
