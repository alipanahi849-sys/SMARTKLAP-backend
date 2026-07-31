package handler

import (
	"clap/internal/modules/auth/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
	VerifyOTP(c *gin.Context)
	RefreshToken(c *gin.Context)
	GetMe(c *gin.Context)
	UpdateProfile(c *gin.Context)
}

type authHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) AuthHandler {
	return &authHandler{
		authService: authService,
	}
}

// RegisterRequest is the passwordless sign-up payload: { "name", "email" }.
type RegisterRequest struct {
	Name  string `json:"name" binding:"required,max=100"`
	Email string `json:"email" binding:"required,email"`
}

// LoginRequest requests an OTP for an existing account: { "email" }.
type LoginRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type VerifyOTPRequest struct {
	Email   string `json:"email" binding:"required,email"`
	OTPCode string `json:"code" binding:"required,len=4,numeric"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}

// Register godoc
//
//	@Summary		Register (OTP email)
//	@Description	Start sign-up with name+email and send a 4-digit OTP. User is saved only after verify-otp. Call register again to resend.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RegisterRequest	true	"Registration payload"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		409		{object}	response.Response
//	@Router			/api/v1/auth/register [post]
func (h *authHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if _, err := h.authService.Register(c.Request.Context(), req.Name, req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, response.EmptyObject, "OTP sent successfully")
}

// Login godoc
//
//	@Summary		Login (OTP email)
//	@Description	Send a 4-digit OTP to a registered email. Call again to resend (cooldown applies).
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginRequest	true	"Login payload"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		404		{object}	response.Response
//	@Failure		429		{object}	response.Response
//	@Router			/api/v1/auth/login [post]
func (h *authHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if _, err := h.authService.Login(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, response.EmptyObject, "OTP sent successfully")
}

// VerifyOTP godoc
//
//	@Summary		Verify OTP
//	@Description	Validate the 4-digit OTP and issue access/refresh tokens
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		VerifyOTPRequest	true	"OTP verification payload"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Router			/api/v1/auth/verify-otp [post]
func (h *authHandler) VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	_, tokenPair, err := h.authService.VerifyOTP(
		c.Request.Context(), req.Email, req.OTPCode, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
	}, "OTP verified successfully")
}

// RefreshToken godoc
//
//	@Summary		Refresh tokens
//	@Description	Exchange a refresh token for a new access/refresh pair
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RefreshTokenRequest	true	"Refresh token payload"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Router			/api/v1/auth/refresh [post]
func (h *authHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, tokenPair, "Tokens refreshed successfully")
}

// GetMe godoc
//
//	@Summary		Current user
//	@Description	Return the authenticated user profile
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Router			/api/v1/users/me [get]
func (h *authHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	user, err := h.authService.GetUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, user, "User fetched successfully")
}

// UpdateProfile godoc
//
//	@Summary		Update profile
//	@Description	Update the authenticated user profile fields
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		UpdateProfileRequest	true	"Profile fields"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Router			/api/v1/users/me [put]
func (h *authHandler) UpdateProfile(c *gin.Context) {
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
	if req.FirstName != "" {
		updates["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		updates["last_name"] = req.LastName
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}

	user, err := h.authService.UpdateUser(c.Request.Context(), userID, updates)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, user, "Profile updated successfully")
}
