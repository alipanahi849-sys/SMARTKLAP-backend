package notification

import (
	"clap/internal/modules/notification/handler"
	"clap/internal/modules/notification/repository"
	"clap/internal/modules/notification/service"
	"clap/internal/shared/database"
	"clap/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	h := handler.NewNotificationHandler(
		service.NewNotificationService(
			repository.NewPushDeviceRepository(database.GetDB()),
			nil,
		),
	)

	devices := r.Group("/notifications/devices")
	devices.Use(middleware.Auth())
	{
		devices.POST("", h.RegisterDevice)
		devices.DELETE("", h.UnregisterDevice)
	}
}
