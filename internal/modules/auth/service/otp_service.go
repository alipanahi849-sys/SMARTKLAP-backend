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

	"clap/internal/modules/auth/models"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/internal/shared/utils"
)

// OTP behaviour constants.
const (
	otpLength           = 4
	otpTTL              = 5 * time.Minute
	otpResendCooldown   = 30 * time.Second
	otpMaxVerifyAttempt = 5

	otpPurposeRegister = "register"
	otpPurposeLogin    = "login"
)

// OTPResult is returned by register/login OTP operations.
type OTPResult struct {
	UserID            string
	Email             string
	OTPSent           bool
	RetryAfterSeconds int
}

// Register starts passwordless sign-up: stores a pending registration with the
// OTP and emails the code. The user row is created only after VerifyOTP succeeds.
// Returns 409 when the email already belongs to an existing account.
// Call Register again to resend (cooldown applies).
func (s *authService) Register(ctx context.Context, name, email string) (*OTPResult, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	if name == "" {
		return nil, errors.NewBadRequest("Name is required", nil)
	}
	if len(name) > 100 {
		return nil, errors.NewBadRequest("Name must be at most 100 characters", nil)
	}

	if existing, err := s.userRepo.FindByEmail(ctx, email); err == nil && existing != nil {
		return nil, errors.ErrEmailExists
	}

	if err := s.checkResendCooldown(ctx, email); err != nil {
		return nil, err
	}
	if err := s.issueOTP(ctx, email, OTPRecord{
		Purpose: otpPurposeRegister,
		Name:    name,
	}); err != nil {
		return nil, err
	}

	logger.Info().Str("email", email).Msg("otp_register_pending")

	return &OTPResult{Email: email, OTPSent: true}, nil
}

// Login emails a one-time code to an existing account.
// Call again to request a new code (subject to cooldown). Returns 404 if unregistered.
func (s *authService) Login(ctx context.Context, email string) (*OTPResult, error) {
	email = normalizeEmail(email)

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, errors.NewNotFound("Email is not registered", nil)
	}
	if !user.IsActive {
		return nil, errors.NewUnauthorized("User account is inactive", nil)
	}

	if err := s.checkResendCooldown(ctx, email); err != nil {
		return nil, err
	}
	if err := s.issueOTP(ctx, email, OTPRecord{Purpose: otpPurposeLogin}); err != nil {
		return nil, err
	}

	logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", email).
		Msg("otp_login_requested")

	return &OTPResult{UserID: user.ID.String(), Email: email, OTPSent: true}, nil
}

// VerifyOTP checks the code. For register purpose it creates the user; for login
// it authenticates an existing user. On success it issues a JWT pair.
func (s *authService) VerifyOTP(ctx context.Context, email, code, ipAddress, userAgent string) (*models.User, *utils.TokenPair, error) {
	email = normalizeEmail(email)

	rec, err := s.otpStore.Get(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if rec == nil {
		return nil, nil, errors.NewUnauthorized("Invalid or expired code", nil)
	}
	if rec.Attempts >= otpMaxVerifyAttempt {
		_ = s.otpStore.Delete(ctx, email)
		return nil, nil, errors.NewTooManyRequests("Too many attempts. Request a new code.", nil)
	}

	if subtle.ConstantTimeCompare([]byte(hashOTP(code)), []byte(rec.CodeHash)) != 1 {
		attempts, incErr := s.otpStore.IncrementAttempts(ctx, email)
		if incErr == nil && attempts >= otpMaxVerifyAttempt {
			_ = s.otpStore.Delete(ctx, email)
		}
		logger.Warn().Str("email", email).Msg("otp_verify_failed")
		return nil, nil, errors.NewUnauthorized("Invalid or expired code", nil)
	}

	// Code accepted — single use. Keep a copy of pending fields before delete.
	pending := *rec
	if err := s.otpStore.Delete(ctx, email); err != nil {
		return nil, nil, err
	}
	_ = s.otpStore.ClearCooldown(ctx, email)

	var user *models.User
	switch pending.Purpose {
	case otpPurposeRegister:
		user, err = s.createVerifiedUser(ctx, pending.Name, email)
		if err != nil {
			return nil, nil, err
		}
	default: // login (and legacy records without purpose)
		user, err = s.userRepo.FindByEmail(ctx, email)
		if err != nil {
			return nil, nil, errors.ErrUserNotFound
		}
		if !user.IsActive {
			return nil, nil, errors.NewUnauthorized("User account is inactive", nil)
		}
		if !user.IsVerified {
			user.IsVerified = true
			if err := s.userRepo.Update(ctx, user); err != nil {
				return nil, nil, err
			}
		}
	}

	tokenPair, err := s.generateTokenPair(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, nil, err
	}

	logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", email).
		Str("purpose", pending.Purpose).
		Msg("otp_verified")

	return user, tokenPair, nil
}

// ─── internals ────────────────────────────────────────────────────────────────

func (s *authService) createVerifiedUser(ctx context.Context, name, email string) (*models.User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.NewBadRequest("Name is required", nil)
	}

	// Race: another verify may have created the account already.
	if existing, err := s.userRepo.FindByEmail(ctx, email); err == nil && existing != nil {
		return nil, errors.ErrEmailExists
	}

	user := &models.User{
		Email:        email,
		PasswordHash: "",
		FirstName:    name,
		IsActive:     true,
		IsVerified:   true,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	userRole, err := s.roleRepo.FindByName(ctx, models.RoleUser)
	if err != nil {
		return nil, err
	}
	if err := s.userRepo.AddRole(ctx, user.ID, userRole.ID); err != nil {
		return nil, err
	}

	user, err = s.userRepo.FindByID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *authService) issueOTP(ctx context.Context, email string, rec OTPRecord) error {
	code, err := generateOTPCode(otpLength)
	if err != nil {
		return errors.NewInternal("Failed to generate code", err)
	}

	rec.CodeHash = hashOTP(code)
	rec.Attempts = 0
	if err := s.otpStore.Save(ctx, email, rec, otpTTL); err != nil {
		return err
	}
	if err := s.otpStore.SetCooldown(ctx, email, otpResendCooldown); err != nil {
		return err
	}
	if err := s.otpSender.SendOTP(ctx, email, code); err != nil {
		return errors.NewInternal("Failed to send code", err)
	}
	return nil
}

func (s *authService) checkResendCooldown(ctx context.Context, email string) error {
	remaining, err := s.otpStore.CooldownRemaining(ctx, email)
	if err != nil {
		return err
	}
	if remaining > 0 {
		return errors.NewTooManyRequests(
			fmt.Sprintf("Please wait %d seconds before requesting a new code", int(remaining.Seconds())+1), nil,
		)
	}
	return nil
}

func generateOTPCode(length int) (string, error) {
	digits := make([]byte, length)
	for i := range digits {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits[i] = byte('0' + n.Int64())
	}
	return string(digits), nil
}

func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
