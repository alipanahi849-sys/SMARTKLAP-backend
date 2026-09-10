package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"clap/internal/modules/adminauth/models"
	"clap/internal/modules/adminauth/repository"
	authsvc "clap/internal/modules/auth/service"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

const (
	otpLength           = 4
	otpTTL              = 5 * time.Minute
	otpResendCooldown   = 30 * time.Second
	otpMaxVerifyAttempt = 5

	otpPurposeAdminLogin       = "admin_login"
	otpPurposeAdminChangeEmail = "admin_change_email"
)

type OTPResult struct {
	AdminID           string
	Email             string
	OTPSent           bool
	RetryAfterSeconds int
}

type AdminAccount struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type AdminAuthService interface {
	Login(ctx context.Context, email string) (*OTPResult, error)
	VerifyOTP(ctx context.Context, email, code, ipAddress, userAgent string) (*models.AdminUser, *utils.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*utils.TokenPair, error)
	Me(ctx context.Context, adminID uuid.UUID) (*AdminAccount, error)
	UpdateMe(ctx context.Context, adminID uuid.UUID, name string) (*AdminAccount, error)
	RequestChangeEmail(ctx context.Context, adminID uuid.UUID, newEmail string) (*OTPResult, error)
	VerifyChangeEmail(ctx context.Context, adminID uuid.UUID, newEmail, code, ipAddress, userAgent string) (*models.AdminUser, *utils.TokenPair, error)
}

type adminAuthService struct {
	adminRepo        repository.AdminUserRepository
	refreshTokenRepo repository.AdminRefreshTokenRepository
	otpStore         authsvc.OTPStore
	otpSender        authsvc.OTPSender
}

func NewAdminAuthService(
	adminRepo repository.AdminUserRepository,
	refreshTokenRepo repository.AdminRefreshTokenRepository,
	otpStore authsvc.OTPStore,
	otpSender authsvc.OTPSender,
) AdminAuthService {
	return &adminAuthService{
		adminRepo:        adminRepo,
		refreshTokenRepo: refreshTokenRepo,
		otpStore:         otpStore,
		otpSender:        otpSender,
	}
}

func (s *adminAuthService) Login(ctx context.Context, email string) (*OTPResult, error) {
	email = normalizeEmail(email)

	admin, err := s.adminRepo.FindByEmail(ctx, email)
	if err != nil {
		if err == errors.ErrUserNotFound {
			return nil, errors.NewNotFound("Admin email is not registered", nil)
		}
		return nil, err
	}
	if !admin.IsActive {
		return nil, errors.NewUnauthorized("Admin account is inactive", nil)
	}

	otpKey := adminOTPKey(email)
	if err := s.checkResendCooldown(ctx, otpKey); err != nil {
		return nil, err
	}
	if err := s.issueOTP(ctx, otpKey, email, authsvc.OTPRecord{Purpose: otpPurposeAdminLogin}); err != nil {
		return nil, err
	}

	logger.Info().
		Str("admin_id", admin.ID.String()).
		Str("email", email).
		Msg("admin_otp_login_requested")

	return &OTPResult{AdminID: admin.ID.String(), Email: email, OTPSent: true}, nil
}

func (s *adminAuthService) VerifyOTP(ctx context.Context, email, code, ipAddress, userAgent string) (*models.AdminUser, *utils.TokenPair, error) {
	email = normalizeEmail(email)
	otpKey := adminOTPKey(email)

	pending, err := s.consumeOTP(ctx, otpKey, code)
	if err != nil {
		return nil, nil, err
	}
	if pending.Purpose != otpPurposeAdminLogin {
		return nil, nil, errors.NewUnauthorized("Invalid or expired code", nil)
	}

	admin, err := s.adminRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, errors.ErrUserNotFound
	}
	if !admin.IsActive {
		return nil, nil, errors.NewUnauthorized("Admin account is inactive", nil)
	}

	tokenPair, err := s.generateTokenPair(ctx, admin, ipAddress, userAgent)
	if err != nil {
		return nil, nil, err
	}

	logger.Info().
		Str("admin_id", admin.ID.String()).
		Str("email", email).
		Msg("admin_otp_verified")

	return admin, tokenPair, nil
}

func (s *adminAuthService) RefreshToken(ctx context.Context, refreshToken string) (*utils.TokenPair, error) {
	tokenHash := utils.HashRefreshToken(refreshToken)
	token, err := s.refreshTokenRepo.FindByToken(ctx, tokenHash)
	if err != nil {
		return nil, errors.ErrInvalidToken
	}
	if !token.IsValid() {
		return nil, errors.ErrInvalidToken
	}

	admin, err := s.adminRepo.FindByID(ctx, token.AdminUserID)
	if err != nil {
		return nil, errors.ErrInvalidToken
	}
	if !admin.IsActive {
		return nil, errors.NewUnauthorized("Admin account is inactive", nil)
	}

	if err := s.refreshTokenRepo.Revoke(ctx, token.ID); err != nil {
		return nil, err
	}

	return s.generateTokenPair(ctx, admin, token.IPAddress, token.UserAgent)
}

func (s *adminAuthService) Me(ctx context.Context, adminID uuid.UUID) (*AdminAccount, error) {
	admin, err := s.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	return toAdminAccount(admin), nil
}

func (s *adminAuthService) UpdateMe(ctx context.Context, adminID uuid.UUID, name string) (*AdminAccount, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.NewBadRequest("Name is required", nil)
	}
	if len(name) > 100 {
		return nil, errors.NewBadRequest("Name must be at most 100 characters", nil)
	}

	admin, err := s.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, err
	}

	admin.FirstName = name
	admin.LastName = ""
	if err := s.adminRepo.Update(ctx, admin); err != nil {
		return nil, err
	}

	return toAdminAccount(admin), nil
}

func (s *adminAuthService) RequestChangeEmail(ctx context.Context, adminID uuid.UUID, newEmail string) (*OTPResult, error) {
	newEmail = normalizeEmail(newEmail)

	admin, err := s.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if !admin.IsActive {
		return nil, errors.NewUnauthorized("Admin account is inactive", nil)
	}
	if newEmail == admin.Email {
		return nil, errors.NewBadRequest("New email must be different from the current email", nil)
	}
	if existing, findErr := s.adminRepo.FindByEmail(ctx, newEmail); findErr == nil && existing != nil {
		return nil, errors.ErrEmailExists
	}

	otpKey := adminOTPKey(newEmail)
	if err := s.checkResendCooldown(ctx, otpKey); err != nil {
		return nil, err
	}
	if err := s.issueOTP(ctx, otpKey, newEmail, authsvc.OTPRecord{
		Purpose: otpPurposeAdminChangeEmail,
		UserID:  adminID.String(),
	}); err != nil {
		return nil, err
	}

	return &OTPResult{AdminID: adminID.String(), Email: newEmail, OTPSent: true}, nil
}

func (s *adminAuthService) VerifyChangeEmail(ctx context.Context, adminID uuid.UUID, newEmail, code, ipAddress, userAgent string) (*models.AdminUser, *utils.TokenPair, error) {
	newEmail = normalizeEmail(newEmail)
	otpKey := adminOTPKey(newEmail)

	admin, err := s.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, nil, err
	}
	if !admin.IsActive {
		return nil, nil, errors.NewUnauthorized("Admin account is inactive", nil)
	}
	if newEmail == admin.Email {
		return nil, nil, errors.NewBadRequest("New email must be different from the current email", nil)
	}

	rec, err := s.otpStore.Get(ctx, otpKey)
	if err != nil {
		return nil, nil, err
	}
	if rec == nil || rec.Purpose != otpPurposeAdminChangeEmail || rec.UserID != adminID.String() {
		return nil, nil, errors.NewUnauthorized("Invalid or expired code", nil)
	}

	if _, err := s.consumeOTP(ctx, otpKey, code); err != nil {
		return nil, nil, err
	}

	if existing, findErr := s.adminRepo.FindByEmail(ctx, newEmail); findErr == nil && existing != nil && existing.ID != adminID {
		return nil, nil, errors.ErrEmailExists
	}

	admin.Email = newEmail
	if err := s.adminRepo.Update(ctx, admin); err != nil {
		return nil, nil, err
	}

	_ = s.refreshTokenRepo.RevokeAllForAdmin(ctx, adminID)

	admin, err = s.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, nil, err
	}

	tokenPair, err := s.generateTokenPair(ctx, admin, ipAddress, userAgent)
	if err != nil {
		return nil, nil, err
	}

	return admin, tokenPair, nil
}

func (s *adminAuthService) generateTokenPair(ctx context.Context, admin *models.AdminUser, ipAddress, userAgent string) (*utils.TokenPair, error) {
	roles := []string{string(utils.RoleAdmin)}

	accessToken, expiresIn, err := utils.GenerateAdminAccessToken(admin.ID, admin.Email, roles)
	if err != nil {
		return nil, err
	}

	refreshTokenString, expiresAt, err := utils.GenerateRefreshToken(admin.ID)
	if err != nil {
		return nil, err
	}

	refreshToken := &models.AdminRefreshToken{
		AdminUserID: admin.ID,
		Token:       utils.HashRefreshToken(refreshTokenString),
		ExpiresAt:   expiresAt,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
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

func (s *adminAuthService) issueOTP(ctx context.Context, otpKey, deliveryEmail string, rec authsvc.OTPRecord) error {
	code, err := generateOTPCode(otpLength)
	if err != nil {
		return errors.NewInternal("Failed to generate code", err)
	}

	rec.CodeHash = hashOTP(code)
	rec.Attempts = 0
	if err := s.otpStore.Save(ctx, otpKey, rec, otpTTL); err != nil {
		return err
	}
	if err := s.otpStore.SetCooldown(ctx, otpKey, otpResendCooldown); err != nil {
		return err
	}
	if err := s.otpSender.SendOTP(ctx, deliveryEmail, code); err != nil {
		_ = s.otpStore.Delete(ctx, otpKey)
		_ = s.otpStore.ClearCooldown(ctx, otpKey)
		return errors.NewInternal("Failed to send OTP", err)
	}
	return nil
}

func (s *adminAuthService) consumeOTP(ctx context.Context, otpKey, code string) (*authsvc.OTPRecord, error) {
	rec, err := s.otpStore.Get(ctx, otpKey)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, errors.NewUnauthorized("Invalid or expired code", nil)
	}
	if rec.Attempts >= otpMaxVerifyAttempt {
		_ = s.otpStore.Delete(ctx, otpKey)
		return nil, errors.NewTooManyRequests("Too many attempts. Request a new code.", nil)
	}

	if subtle.ConstantTimeCompare([]byte(hashOTP(code)), []byte(rec.CodeHash)) != 1 {
		attempts, incErr := s.otpStore.IncrementAttempts(ctx, otpKey)
		if incErr == nil && attempts >= otpMaxVerifyAttempt {
			_ = s.otpStore.Delete(ctx, otpKey)
		}
		return nil, errors.NewUnauthorized("Invalid or expired code", nil)
	}

	pending := *rec
	if err := s.otpStore.Delete(ctx, otpKey); err != nil {
		return nil, err
	}
	_ = s.otpStore.ClearCooldown(ctx, otpKey)
	return &pending, nil
}

func (s *adminAuthService) checkResendCooldown(ctx context.Context, otpKey string) error {
	remaining, err := s.otpStore.CooldownRemaining(ctx, otpKey)
	if err != nil {
		return err
	}
	if remaining > 0 {
		secs := int(remaining.Seconds())
		if secs < 1 {
			secs = 1
		}
		return errors.NewTooManyRequests(fmt.Sprintf("Please wait %d seconds before requesting another code", secs), nil)
	}
	return nil
}

func toAdminAccount(admin *models.AdminUser) *AdminAccount {
	return &AdminAccount{
		ID:    admin.ID.String(),
		Email: admin.Email,
		Name:  admin.DisplayName(),
	}
}

func adminOTPKey(email string) string {
	return "admin:" + email
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func generateOTPCode(length int) (string, error) {
	max := big.NewInt(int64(pow10(length)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", length, n.Int64()), nil
}

func pow10(n int) int {
	v := 1
	for i := 0; i < n; i++ {
		v *= 10
	}
	return v
}
