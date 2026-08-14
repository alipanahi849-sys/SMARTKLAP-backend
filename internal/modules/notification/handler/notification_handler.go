package handler

import (
	"clap/internal/modules/notification/dto"
	"clap/internal/modules/notification/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NotificationHandler interface {
	RegisterDevice(c *gin.Context)
	UnregisterDevice(c *gin.Context)
}

type notificationHandler struct {
	svc service.NotificationService
}

func NewNotificationHandler(svc service.NotificationService) NotificationHandler {
	return &notificationHandler{svc: svc}
}

// RegisterDevice godoc
//
//	@Summary		Register FCM device token
//	@Description	Stores or updates the caller's FCM token so the backend can send push notifications to this device
//	@Tags			notifications
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.RegisterDeviceRequest	true	"FCM token and platform"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Router			/api/v1/notifications/devices [post]
func (h *notificationHandler) RegisterDevice(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.RegisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.RegisterDevice(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessWithMessage(c, result, "Push device registered")
}

// UnregisterDevice godoc
//
//	@Summary		Unregister FCM device token
//	@Description	Removes the FCM token for the caller (used on logout). Idempotent if the token is already gone.
//	@Tags			notifications
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.UnregisterDeviceRequest	true	"FCM token"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Router			/api/v1/notifications/devices [delete]
func (h *notificationHandler) UnregisterDevice(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.UnregisterDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.svc.UnregisterDevice(c.Request.Context(), userID, &req); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessWithMessage(c, response.EmptyObject, "Push device unregistered")
}
