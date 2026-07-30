package handler

import (
	"clap/internal/modules/user/models"
	"clap/internal/modules/user/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProfileHandler interface {
	GetProfile(c *gin.Context)
	CreateProfile(c *gin.Context)
	UpdateProfile(c *gin.Context)
	DeleteProfile(c *gin.Context)
}

type profileHandler struct {
	profileService service.ProfileService
}

func NewProfileHandler(profileService service.ProfileService) ProfileHandler {
	return &profileHandler{
		profileService: profileService,
	}
}

type CreateProfileRequest struct {
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	DateOfBirth string `json:"date_of_birth"`
	Country     string `json:"country"`
	City        string `json:"city"`
}

type UpdateProfileRequest struct {
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	DateOfBirth string `json:"date_of_birth"`
	Country     string `json:"country"`
	City        string `json:"city"`
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

	profile, err := h.profileService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, profile)
}

// Create profile godoc
//
//	@Summary		Create profile
//	@Tags			profiles
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateProfileRequest	true	"Request body"
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

	var req CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	profile := &models.Profile{
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
		DateOfBirth: &req.DateOfBirth,
		Country:     req.Country,
		City:        req.City,
	}

	createdProfile, err := h.profileService.CreateProfile(c.Request.Context(), userID, profile)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, createdProfile)
}

// Update profile godoc
//
//	@Summary		Update profile
//	@Tags			profiles
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		UpdateProfileRequest	true	"Request body"
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

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updates := make(map[string]interface{})
	if req.Bio != "" {
		updates["bio"] = req.Bio
	}
	if req.AvatarURL != "" {
		updates["avatar_url"] = req.AvatarURL
	}
	if req.DateOfBirth != "" {
		updates["date_of_birth"] = req.DateOfBirth
	}
	if req.Country != "" {
		updates["country"] = req.Country
	}
	if req.City != "" {
		updates["city"] = req.City
	}

	profile, err := h.profileService.UpdateProfile(c.Request.Context(), userID, updates)
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

	if err := h.profileService.DeleteProfile(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Profile deleted successfully")
}
