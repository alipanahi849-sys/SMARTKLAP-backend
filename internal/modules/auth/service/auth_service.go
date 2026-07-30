package service

import (
	"clap/internal/modules/auth/models"
	"clap/internal/modules/auth/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/utils"
	"context"
	"strings"

	"github.com/google/uuid"
)

type AuthService interface {
	Register(ctx context.Context, name, email, password string) (*models.User, *utils.TokenPair, error)
	Login(ctx context.Context, email, password, ipAddress, userAgent string) (*models.User, *utils.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*utils.TokenPair, error)
	Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateUser(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) (*models.User, error)

	// Passwordless OTP flow (Mobile API Contract §1).
	RegisterOTP(ctx context.Context, name, email string) (*OTPResult, error)
	LoginOTP(ctx context.Context, email string) (*OTPResult, error)
	ResendOTP(ctx context.Context, email string) (*OTPResult, error)
	VerifyOTP(ctx context.Context, email, code, ipAddress, userAgent string) (*models.User, *utils.TokenPair, error)
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

func (s *authService) Register(ctx context.Context, name, email, password string) (*models.User, *utils.TokenPair, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil, errors.NewBadRequest("Name is required", nil)
	}
	if len(name) > 100 {
		return nil, nil, errors.NewBadRequest("Name must be at most 100 characters", nil)
	}

	if err := utils.ValidatePassword(password); err != nil {
		return nil, nil, errors.NewBadRequest("Invalid password", err)
	}

	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, nil, errors.ErrEmailExists
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, nil, errors.NewInternal("Failed to hash password", err)
	}

	user := &models.User{
		Email:        email,
		PasswordHash: passwordHash,
		FirstName:    name,
		IsActive:     true,
		IsVerified:   false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, err
	}

	userRole, err := s.roleRepo.FindByName(ctx, models.RoleUser)
	if err != nil {
		return nil, nil, err
	}

	if err := s.userRepo.AddRole(ctx, user.ID, userRole.ID); err != nil {
		return nil, nil, err
	}

	user, err = s.userRepo.FindByID(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	tokenPair, err := s.generateTokenPair(ctx, user, "", "")
	if err != nil {
		return nil, nil, err
	}

	return user, tokenPair, nil
}

func (s *authService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*models.User, *utils.TokenPair, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, errors.ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, nil, errors.NewUnauthorized("User account is inactive", nil)
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return nil, nil, errors.ErrInvalidCredentials
	}

	tokenPair, err := s.generateTokenPair(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, nil, err
	}

	return user, tokenPair, nil
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

func (s *authService) Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	tokenHash := utils.HashRefreshToken(refreshToken)
	token, err := s.refreshTokenRepo.FindByToken(ctx, tokenHash)
	if err != nil {
		return errors.ErrInvalidToken
	}

	if token.UserID != userID {
		return errors.ErrPermissionDenied
	}

	return s.refreshTokenRepo.Revoke(ctx, token.ID)
}

func (s *authService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.refreshTokenRepo.RevokeAllForUser(ctx, userID)
}

func (s *authService) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

func (s *authService) UpdateUser(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if password, ok := updates["password"]; ok {
		if passwordStr, ok := password.(string); ok {
			if err := utils.ValidatePassword(passwordStr); err != nil {
				return nil, errors.NewBadRequest("Invalid password", err)
			}
			passwordHash, err := utils.HashPassword(passwordStr)
			if err != nil {
				return nil, errors.NewInternal("Failed to hash password", err)
			}
			updates["password_hash"] = passwordHash
			delete(updates, "password")
		}
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return s.userRepo.FindByID(ctx, userID)
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
