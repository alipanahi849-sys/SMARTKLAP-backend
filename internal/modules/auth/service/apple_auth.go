package service

import (
	"context"
	"strings"

	"clap/internal/modules/auth/models"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/internal/shared/utils"
)

func (s *authService) LoginWithApple(ctx context.Context, idToken, givenName, familyName, ipAddress, userAgent string) (*models.User, *utils.TokenPair, error) {
	if s.appleVerifier == nil {
		return nil, nil, errors.NewServiceUnavailable("Apple sign-in is not configured", nil)
	}

	identity, err := s.appleVerifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, nil, err
	}
	identity.Email = normalizeEmail(identity.Email)
	identity.Given = strings.TrimSpace(givenName)
	identity.Family = strings.TrimSpace(familyName)
	if strings.TrimSpace(identity.Subject) == "" {
		return nil, nil, errors.NewUnauthorized("Invalid Apple token", nil)
	}

	user, err := s.upsertAppleUser(ctx, identity)
	if err != nil {
		return nil, nil, err
	}
	if !user.IsActive {
		return nil, nil, errors.NewUnauthorized("User account is inactive", nil)
	}

	tokenPair, err := s.generateTokenPair(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, nil, err
	}

	logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", user.Email).
		Msg("apple_login")

	return user, tokenPair, nil
}

func (s *authService) upsertAppleUser(ctx context.Context, identity *AppleIdentity) (*models.User, error) {
	if user, err := s.userRepo.FindByAppleID(ctx, identity.Subject); err == nil && user != nil {
		return s.applyAppleProfile(ctx, user, identity)
	} else if err != nil && err != errors.ErrUserNotFound {
		return nil, err
	}

	if identity.Email == "" {
		return nil, errors.NewUnauthorized("Apple account email is unavailable. Revoke this app under Settings > Apple ID > Sign in with Apple, then try again.", nil)
	}

	if user, err := s.userRepo.FindByEmail(ctx, identity.Email); err == nil && user != nil {
		if user.AppleID == nil || *user.AppleID == "" {
			aid := identity.Subject
			user.AppleID = &aid
		}
		if !user.IsVerified {
			user.IsVerified = true
		}
		if strings.TrimSpace(user.FirstName) == "" && strings.TrimSpace(user.LastName) == "" {
			user.FirstName, user.LastName = appleDisplayName(identity)
		}
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, err
		}
		return s.userRepo.FindByID(ctx, user.ID)
	} else if err != nil && err != errors.ErrUserNotFound {
		return nil, err
	}

	first, last := appleDisplayName(identity)
	aid := identity.Subject
	user, err := s.createVerifiedUser(ctx, strings.TrimSpace(first+" "+last), identity.Email)
	if err != nil {
		if err == errors.ErrEmailExists {
			return s.upsertAppleUser(ctx, identity)
		}
		return nil, err
	}
	user.AppleID = &aid
	user.FirstName = first
	user.LastName = last
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return s.userRepo.FindByID(ctx, user.ID)
}

func (s *authService) applyAppleProfile(ctx context.Context, user *models.User, identity *AppleIdentity) (*models.User, error) {
	if strings.TrimSpace(user.FirstName) != "" || strings.TrimSpace(user.LastName) != "" {
		return user, nil
	}
	if identity.Given == "" && identity.Family == "" {
		return user, nil
	}
	user.FirstName = identity.Given
	user.LastName = identity.Family
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return s.userRepo.FindByID(ctx, user.ID)
}

func appleDisplayName(identity *AppleIdentity) (first, last string) {
	if identity.Given != "" || identity.Family != "" {
		return identity.Given, identity.Family
	}
	name := ""
	if identity.Email != "" {
		local, _, _ := strings.Cut(identity.Email, "@")
		name = strings.TrimSpace(local)
	}
	if name == "" {
		name = "Apple User"
	}
	return name, ""
}
