package handler

import (
	"mime/multipart"
	"strconv"

	"clap/internal/modules/user/dto"
	"clap/internal/modules/user/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

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
//	@Tags			profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profile/leaderboard [get]
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

	file := firstFormFile(c, "avatar", "file", "image", "photo")
	if file == nil {
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

// firstFormFile returns the first multipart file matching any of the given
// field names, or the first uploaded file in the form as a fallback.
func firstFormFile(c *gin.Context, names ...string) *multipart.FileHeader {
	for _, name := range names {
		if file, err := c.FormFile(name); err == nil && file != nil {
			return file
		}
	}

	form, err := c.MultipartForm()
	if err != nil || form == nil {
		return nil
	}
	for _, files := range form.File {
		if len(files) > 0 && files[0] != nil {
			return files[0]
		}
	}
	return nil
}
