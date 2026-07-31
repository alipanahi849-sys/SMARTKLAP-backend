package handler

import (
	"strconv"

	"clap/internal/modules/user/dto"
	"clap/internal/modules/user/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MobileProfileHandler serves mobile Profile screens nested under /profile/mobile.
type MobileProfileHandler interface {
	GetMe(c *gin.Context)
	UpdateMe(c *gin.Context)
	Leaderboard(c *gin.Context)
	UploadAvatar(c *gin.Context)
}

type mobileProfileHandler struct {
	svc service.MobileProfileService
}

func NewMobileProfileHandler(svc service.MobileProfileService) MobileProfileHandler {
	return &mobileProfileHandler{svc: svc}
}

// Mobile profile me godoc
//
//	@Summary		Get mobile profile
//	@Tags			profile/mobile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profile/mobile/me [get]
func (h *mobileProfileHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	profile, err := h.svc.GetMe(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, profile)
}

// Update mobile profile godoc
//
//	@Summary		Update mobile profile
//	@Tags			profile/mobile
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.UpdateMobileProfileRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profile/mobile/me [patch]
func (h *mobileProfileHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.UpdateMobileProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	profile, err := h.svc.UpdateMe(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, profile)
}

// Leaderboard godoc
//
//	@Summary		Leaderboard
//	@Tags			profile/mobile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profile/mobile/leaderboard [get]
func (h *mobileProfileHandler) Leaderboard(c *gin.Context) {
	limit := 4
	if raw, ok := c.GetQuery("limit"); ok {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			response.BadRequest(c, "limit must be a positive integer")
			return
		}
		limit = parsed
	}

	board, err := h.svc.Leaderboard(c.Request.Context(), limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, board)
}

// Upload avatar godoc
//
//	@Summary		Upload avatar
//	@Tags			profile/mobile
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"Avatar image"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profile/mobile/me/avatar [post]
func (h *mobileProfileHandler) UploadAvatar(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		response.BadRequest(c, "avatar file is required")
		return
	}

	result, uploadErr := h.svc.UploadAvatar(c.Request.Context(), userID, file)
	if uploadErr != nil {
		response.Error(c, uploadErr)
		return
	}
	response.Success(c, result)
}
