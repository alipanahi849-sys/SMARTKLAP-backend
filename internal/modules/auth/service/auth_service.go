package service

import (
	"clap/internal/modules/auth/models"
	"clap/internal/modules/auth/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/utils"
	"context"
)

type AuthService interface {
	// Passwordless OTP auth: register/login send a code; verify-otp issues tokens.
	Register(ctx context.Context, name, email string) (*OTPResult, error)
	Login(ctx context.Context, email string) (*OTPResult, error)
	VerifyOTP(ctx context.Context, email, code, ipAddress, userAgent string) (*models.User, *utils.TokenPair, error)

	RefreshToken(ctx context.Context, refreshToken string) (*utils.TokenPair, error)
}

type authService struct {
	userRepo         repository.UserRepository
	roleRepo         repository.RoleRepository
	refreshTokenRepo repository.RefreshTokenRepository
	otpStore         OTPStore
	otpSender        OTPSender
}

func NewAuthService(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
) AuthService {
	return NewAuthServiceWithOTP(userRepo, roleRepo, refreshTokenRepo, NewMemoryOTPStore(), NewLogOTPSender(false))
}

// NewAuthServiceWithOTP wires an explicit OTP store and sender (Redis-backed
// in production, in-memory in tests).
func NewAuthServiceWithOTP(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	otpStore OTPStore,
	otpSender OTPSender,
) AuthService {
	return &authService{
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		refreshTokenRepo: refreshTokenRepo,
		otpStore:         otpStore,
		otpSender:        otpSender,
	}
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*utils.TokenPair, error) {
	tokenHash := utils.HashRefreshToken(refreshToken)
	token, err := s.refreshTokenRepo.FindByToken(ctx, tokenHash)
	if err != nil {
		return nil, errors.ErrInvalidToken
	}

	if !token.IsValid() {
		return nil, errors.ErrInvalidToken
	}

	user, err := s.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, errors.ErrInvalidToken
	}

	if !user.IsActive {
		return nil, errors.NewUnauthorized("User account is inactive", nil)
	}

	if err := s.refreshTokenRepo.Revoke(ctx, token.ID); err != nil {
		return nil, err
	}

	newTokenPair, err := s.generateTokenPair(ctx, user, token.IPAddress, token.UserAgent)
	if err != nil {
		return nil, err
	}

	return newTokenPair, nil
}

func (s *authService) generateTokenPair(ctx context.Context, user *models.User, ipAddress, userAgent string) (*utils.TokenPair, error) {
	roles := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = role.Name
	}

	accessToken, expiresIn, err := utils.GenerateAccessToken(user.ID, user.Email, roles)
	if err != nil {
		return nil, err
	}

	refreshTokenString, expiresAt, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		Token:     utils.HashRefreshToken(refreshTokenString),
		ExpiresAt: expiresAt,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &utils.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		ExpiresIn:    expiresIn,
	}, nil
}
