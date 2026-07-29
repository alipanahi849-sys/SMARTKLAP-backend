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
	revealOTP := config.AppConfig != nil && config.AppConfig.Environment == "development"
	otpSender := service.NewLogOTPSender(revealOTP)

	authService := service.NewAuthServiceWithOTP(userRepo, roleRepo, refreshTokenRepo, otpStore, otpSender)
	authHandler := handler.NewAuthHandler(authService)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", middleware.AuthRateLimit(), authHandler.Register)
		authGroup.POST("/login", middleware.AuthRateLimit(), authHandler.Login)
		authGroup.POST("/verify-otp", middleware.AuthRateLimit(), authHandler.VerifyOTP)
		authGroup.POST("/resend-otp", middleware.AuthRateLimit(), authHandler.ResendOTP)
		authGroup.POST("/refresh", middleware.AuthRateLimit(), authHandler.RefreshToken)
		authGroup.POST("/logout", middleware.Auth(), authHandler.Logout)
		authGroup.POST("/logout-all", middleware.Auth(), authHandler.LogoutAll)
	}

	userGroup := r.Group("/users")
	{
		userGroup.GET("/me", middleware.Auth(), authHandler.GetMe)
		userGroup.PUT("/me", middleware.Auth(), authHandler.UpdateProfile)
	}
}
