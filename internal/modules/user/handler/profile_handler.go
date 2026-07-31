package handler

import (
	"bytes"
	"io"

	"clap/internal/modules/user/dto"
	"clap/internal/modules/user/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProfileHandler serves /api/v1/profiles/me with the same compact mobile shape
// as /api/v1/profile/me (contract §2): name, email, avatar_url, points, rank.
type ProfileHandler interface {
	GetProfile(c *gin.Context)
	CreateProfile(c *gin.Context)
	UpdateProfile(c *gin.Context)
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
//	@Tags			profiles
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profiles/me [get]
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

// Create profile godoc
//
//	@Summary		Create profile
//	@Description	Ensures profile exists and optionally sets name/email. Returns the compact mobile profile shape.
//	@Tags			profiles
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.UpdateMobileProfileRequest	false	"Optional name/email"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profiles/me [post]
func (h *profileHandler) CreateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	req, ok := bindOptionalProfileUpdate(c)
	if !ok {
		return
	}

	var (
		profile *dto.MobileProfileResponse
		err     error
	)
	if req.Name != nil || req.Email != nil {
		profile, err = h.mobileSvc.UpdateMe(c.Request.Context(), userID, &req)
	} else {
		profile, err = h.mobileSvc.GetMe(c.Request.Context(), userID)
	}
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, profile)
}

// Update profile godoc
//
//	@Summary		Update profile
//	@Tags			profiles
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.UpdateMobileProfileRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profiles/me [put]
func (h *profileHandler) UpdateProfile(c *gin.Context) {
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

	profile, err := h.mobileSvc.UpdateMe(c.Request.Context(), userID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, profile)
}

// Delete profile godoc
//
//	@Summary		Delete profile
//	@Tags			profiles
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/profiles/me [delete]
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

// bindOptionalProfileUpdate accepts missing/empty JSON bodies for create.
// Returns ok=false when a BadRequest response was already written.
func bindOptionalProfileUpdate(c *gin.Context) (dto.UpdateMobileProfileRequest, bool) {
	var req dto.UpdateMobileProfileRequest
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "invalid request body")
		return req, false
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 || bytes.Equal(body, []byte("null")) || bytes.Equal(body, []byte("{}")) {
		return req, true
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return req, false
	}
	return req, true
}
