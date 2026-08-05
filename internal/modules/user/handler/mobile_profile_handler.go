package handler

import (
	"strings"

	"clap/internal/modules/user/dto"
	"clap/internal/modules/user/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MobileProfileHandler serves update/avatar/leaderboard under /profile.
type MobileProfileHandler interface {
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

// Update profile godoc
//
//	@Summary		Update profile
//	@Tags			profile
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.UpdateMobileProfileRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profile/me [patch]
func (h *mobileProfileHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req dto.UpdateMobileProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	profile, err := h.svc.UpdateMe(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessWithMessage(c, profile, "Profile updated successfully")
}

// Leaderboard godoc
//
//	@Summary		Leaderboard
//	@Tags			profile
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cursor	query	string	false	"User ID of the last item from the previous page"
//	@Param			limit	query	int		false	"Items per page (default 4, max 50)"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profile/leaderboard [get]
func (h *mobileProfileHandler) Leaderboard(c *gin.Context) {
	limit := utils.GetLeaderboardCursorLimit(c)

	var cursor *uuid.UUID
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "Invalid cursor")
			return
		}
		cursor = &id
	}

	board, err := h.svc.Leaderboard(c.Request.Context(), dto.LeaderboardFilters{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, board)
}

// Upload avatar godoc
//
//	@Summary		Upload avatar
//	@Tags			profile
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			avatar	formData	file	true	"Avatar image"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profile/me/avatar [post]
func (h *mobileProfileHandler) UploadAvatar(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	file := utils.FirstFormFile(c, "avatar", "file", "image", "photo")
	if file == nil {
		response.BadRequest(c, "avatar file is required")
		return
	}

	result, uploadErr := h.svc.UploadAvatar(c.Request.Context(), userID, file)
	if uploadErr != nil {
		response.Error(c, uploadErr)
		return
	}
	response.SuccessWithMessage(c, result, "Avatar uploaded successfully")
}
