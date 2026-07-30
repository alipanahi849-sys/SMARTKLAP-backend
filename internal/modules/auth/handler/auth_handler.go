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
	ResendOTP(c *gin.Context)
	RefreshToken(c *gin.Context)
	Logout(c *gin.Context)
	LogoutAll(c *gin.Context)
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

// RegisterRequest is the password registration payload:
// { "name", "email", "password" }.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest supports both auth flows:
//   - Mobile OTP (contract §1.3): { "email" } — sends a code
//   - Legacy password: { "email", "password" }
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password"`
}

type VerifyOTPRequest struct {
	Email   string `json:"email" binding:"required,email"`
	OTPCode string `json:"otp_code" binding:"required,len=4,numeric"`
}

type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Password  string `json:"password" binding:"omitempty,min=8"`
}

// Register godoc
//
//	@Summary		Register a new user
//	@Description	Create an account with name, email, and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RegisterRequest	true	"Registration payload"
//	@Success		201		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		409		{object}	response.Response
//	@Router			/api/v1/auth/register [post]
func (h *authHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, tokenPair, err := h.authService.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, gin.H{
		"user":   user,
		"tokens": tokenPair,
	})
}

// Login godoc
//
//	@Summary		Login
//	@Description	Password login returns tokens. OTP login (email only) sends a code.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginRequest	true	"Login payload"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Router			/api/v1/auth/login [post]
func (h *authHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Legacy password login keeps its original behaviour.
	if req.Password != "" {
		ipAddress := c.ClientIP()
		userAgent := c.Request.UserAgent()

		user, tokenPair, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, ipAddress, userAgent)
		if err != nil {
			response.Error(c, err)
			return
		}

		response.Success(c, gin.H{
			"user":   user,
			"tokens": tokenPair,
		})
		return
	}

	// Mobile OTP login (contract §1.3): send a code to the registered email.
	if _, err := h.authService.LoginOTP(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"otp_sent": true})
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

	user, tokenPair, err := h.authService.VerifyOTP(
		c.Request.Context(), req.Email, req.OTPCode, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"user": gin.H{
			"id":     user.ID,
			"name":   user.DisplayName(),
			"email":  user.Email,
			"points": user.Points,
		},
	})
}

// ResendOTP godoc
//
//	@Summary		Resend OTP
//	@Description	Resend OTP with a 30-second cooldown
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		ResendOTPRequest	true	"Resend OTP payload"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		429		{object}	response.Response
//	@Router			/api/v1/auth/resend-otp [post]
func (h *authHandler) ResendOTP(c *gin.Context) {
	var req ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.authService.ResendOTP(c.Request.Context(), req.Email)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"otp_sent":            result.OTPSent,
		"retry_after_seconds": result.RetryAfterSeconds,
	})
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

	response.Success(c, tokenPair)
}

// Logout godoc
//
//	@Summary		Logout
//	@Description	Revoke the given refresh token, or all sessions if body is empty
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		RefreshTokenRequest	false	"Optional refresh token"
//	@Success		200		{object}	response.Response
//	@Success		204		"No Content"
//	@Failure		401		{object}	response.Response
//	@Router			/api/v1/auth/logout [post]
func (h *authHandler) Logout(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	// Mobile flow (contract §1.4): no body → revoke every session, 204.
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		if err := h.authService.LogoutAll(c.Request.Context(), userID); err != nil {
			response.Error(c, err)
			return
		}
		response.NoContent(c)
		return
	}

	// Legacy flow: revoke the specific refresh token.
	if err := h.authService.Logout(c.Request.Context(), userID, req.RefreshToken); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Logged out successfully")
}

// LogoutAll godoc
//
//	@Summary		Logout all devices
//	@Description	Revoke every refresh token for the current user
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Router			/api/v1/auth/logout-all [post]
func (h *authHandler) LogoutAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	if err := h.authService.LogoutAll(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Logged out from all devices")
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

	response.Success(c, user)
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
	if req.Password != "" {
		updates["password"] = req.Password
	}

	user, err := h.authService.UpdateUser(c.Request.Context(), userID, updates)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, user)
}
