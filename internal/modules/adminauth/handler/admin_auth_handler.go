package handler

import (
	"clap/internal/modules/adminauth/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminAuthHandler interface {
	Login(c *gin.Context)
	VerifyOTP(c *gin.Context)
	RefreshToken(c *gin.Context)
	Me(c *gin.Context)
	UpdateMe(c *gin.Context)
	RequestChangeEmail(c *gin.Context)
	VerifyChangeEmail(c *gin.Context)
}

type adminAuthHandler struct {
	svc service.AdminAuthService
}

func NewAdminAuthHandler(svc service.AdminAuthService) AdminAuthHandler {
	return &adminAuthHandler{svc: svc}
}

type loginRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type verifyOTPRequest struct {
	Email   string `json:"email" binding:"required,email"`
	OTPCode string `json:"code" binding:"required,len=4,numeric"`
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type updateMeRequest struct {
	Name string `json:"name" binding:"required,max=100"`
}

type changeEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type verifyChangeEmailRequest struct {
	Email   string `json:"email" binding:"required,email"`
	OTPCode string `json:"code" binding:"required,len=4,numeric"`
}

func (h *adminAuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	if _, err := h.svc.Login(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, response.EmptyObject, "OTP sent successfully")
}

func (h *adminAuthHandler) VerifyOTP(c *gin.Context) {
	var req verifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	_, tokenPair, err := h.svc.VerifyOTP(
		c.Request.Context(), req.Email, req.OTPCode, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
	}, "OTP verified successfully")
}

func (h *adminAuthHandler) RefreshToken(c *gin.Context) {
	var req refreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	tokenPair, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, tokenPair, "Tokens refreshed successfully")
}

func (h *adminAuthHandler) Me(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if adminID == uuid.Nil {
		response.Unauthorized(c, "Invalid admin")
		return
	}

	account, err := h.svc.Me(c.Request.Context(), adminID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, account)
}

func (h *adminAuthHandler) UpdateMe(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if adminID == uuid.Nil {
		response.Unauthorized(c, "Invalid admin")
		return
	}

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	account, err := h.svc.UpdateMe(c.Request.Context(), adminID, req.Name)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, account, "Admin profile updated")
}

func (h *adminAuthHandler) RequestChangeEmail(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if adminID == uuid.Nil {
		response.Unauthorized(c, "Invalid admin")
		return
	}

	var req changeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	if _, err := h.svc.RequestChangeEmail(c.Request.Context(), adminID, req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, response.EmptyObject, "OTP sent successfully")
}

func (h *adminAuthHandler) VerifyChangeEmail(c *gin.Context) {
	adminID := middleware.GetUserID(c)
	if adminID == uuid.Nil {
		response.Unauthorized(c, "Invalid admin")
		return
	}

	var req verifyChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, middleware.ValidationMessage(err))
		return
	}

	_, tokenPair, err := h.svc.VerifyChangeEmail(
		c.Request.Context(), adminID, req.Email, req.OTPCode, c.ClientIP(), c.Request.UserAgent(),
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
	}, "Email changed successfully")
}
