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
	LoginWithGoogle(c *gin.Context)
	VerifyOTP(c *gin.Context)
	RequestChangeEmail(c *gin.Context)
	VerifyChangeEmail(c *gin.Context)
	RefreshToken(c *gin.Context)
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

// GoogleLoginRequest is the native Google Sign-In payload: { "id_token" }.
type GoogleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

type VerifyOTPRequest struct {
	Email   string `json:"email" binding:"required,email"`
	OTPCode string `json:"code" binding:"required,len=4,numeric"`
}

// ChangeEmailRequest starts an authenticated email change: OTP goes to the new address.
type ChangeEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// VerifyChangeEmailRequest confirms the OTP sent to the new email.
type VerifyChangeEmailRequest struct {
	Email   string `json:"email" binding:"required,email"`
	OTPCode string `json:"code" binding:"required,len=4,numeric"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
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
		response.BadRequest(c, middleware.ValidationMessage(err))
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
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	if _, err := h.authService.Login(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, response.EmptyObject, "OTP sent successfully")
}

// LoginWithGoogle godoc
//
//	@Summary		Login with Google
//	@Description	Verify a Google ID token from native Sign-In and issue access/refresh tokens. Creates the user if the email is new; links an existing OTP account.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		GoogleLoginRequest	true	"Google ID token"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		503		{object}	response.Response
//	@Router			/api/v1/auth/google [post]
func (h *authHandler) LoginWithGoogle(c *gin.Context) {
	var req GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	_, tokenPair, err := h.authService.LoginWithGoogle(
		c.Request.Context(), req.IDToken, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
	}, "Logged in with Google")
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
		response.BadRequest(c, middleware.ValidationMessage(err))
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

// RequestChangeEmail godoc
//
//	@Summary		Change email (request OTP)
//	@Description	Send a 4-digit OTP to the new email. Email is updated only after verify-change-email. Call again to resend.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		ChangeEmailRequest	true	"New email"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		409		{object}	response.Response
//	@Failure		429		{object}	response.Response
//	@Router			/api/v1/auth/change-email [post]
func (h *authHandler) RequestChangeEmail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req ChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	if _, err := h.authService.RequestChangeEmail(c.Request.Context(), userID, req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, response.EmptyObject, "OTP sent successfully")
}

// VerifyChangeEmail godoc
//
//	@Summary		Change email (verify OTP)
//	@Description	Validate the OTP sent to the new email, update the account email, and issue fresh tokens
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		VerifyChangeEmailRequest	true	"New email + OTP"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response
//	@Failure		401		{object}	response.Response
//	@Failure		409		{object}	response.Response
//	@Router			/api/v1/auth/verify-change-email [post]
func (h *authHandler) VerifyChangeEmail(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	var req VerifyChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	_, tokenPair, err := h.authService.VerifyChangeEmail(
		c.Request.Context(), userID, req.Email, req.OTPCode, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
	}, "Email changed successfully")
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
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, tokenPair, "Tokens refreshed successfully")
}
