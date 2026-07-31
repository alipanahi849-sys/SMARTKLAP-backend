package handler

import (
	"clap/internal/modules/user/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProfileHandler serves basic /api/v1/profile/me read + delete.
type ProfileHandler interface {
	GetProfile(c *gin.Context)
	DeleteProfile(c *gin.Context)
}

type profileHandler struct {
	mobileSvc  service.MobileProfileService
	profileSvc service.ProfileService
}

func NewProfileHandler(mobileSvc service.MobileProfileService, profileSvc service.ProfileService) ProfileHandler {
	return &profileHandler{
		mobileSvc:  mobileSvc,
		profileSvc: profileSvc,
	}
}

// Get profile godoc
//
//	@Summary		Get profile
//	@Tags			profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profile/me [get]
func (h *profileHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	profile, err := h.mobileSvc.GetMe(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, profile)
}

// Delete profile godoc
//
//	@Summary		Delete profile
//	@Tags			profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profile/me [delete]
func (h *profileHandler) DeleteProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	if err := h.profileSvc.DeleteProfile(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Profile deleted successfully")
}
