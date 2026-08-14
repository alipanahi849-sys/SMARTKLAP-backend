package service

import (
	"context"
	"strings"

	"clap/internal/modules/auth/models"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/internal/shared/utils"
)

func (s *authService) LoginWithGoogle(ctx context.Context, idToken, ipAddress, userAgent string) (*models.User, *utils.TokenPair, error) {
	if s.googleVerifier == nil {
		return nil, nil, errors.NewServiceUnavailable("Google sign-in is not configured", nil)
	}

	identity, err := s.googleVerifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, nil, err
	}
	identity.Email = normalizeEmail(identity.Email)
	if identity.Email == "" || strings.TrimSpace(identity.Subject) == "" {
		return nil, nil, errors.NewUnauthorized("Invalid Google token", nil)
	}

	user, err := s.upsertGoogleUser(ctx, identity)
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
		Msg("google_login")

	return user, tokenPair, nil
}

func (s *authService) upsertGoogleUser(ctx context.Context, identity *GoogleIdentity) (*models.User, error) {
	if user, err := s.userRepo.FindByGoogleID(ctx, identity.Subject); err == nil && user != nil {
		return user, nil
	} else if err != nil && err != errors.ErrUserNotFound {
		return nil, err
	}

	if user, err := s.userRepo.FindByEmail(ctx, identity.Email); err == nil && user != nil {
		if user.GoogleID == nil || *user.GoogleID == "" {
			gid := identity.Subject
			user.GoogleID = &gid
		}
		if !user.IsVerified {
			user.IsVerified = true
		}
		if strings.TrimSpace(user.FirstName) == "" && strings.TrimSpace(user.LastName) == "" {
			user.FirstName, user.LastName = googleDisplayName(identity)
		}
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, err
		}
		return s.userRepo.FindByID(ctx, user.ID)
	} else if err != nil && err != errors.ErrUserNotFound {
		return nil, err
	}

	first, last := googleDisplayName(identity)
	gid := identity.Subject
	user, err := s.createVerifiedUser(ctx, strings.TrimSpace(first+" "+last), identity.Email)
	if err != nil {
		if err == errors.ErrEmailExists {
			return s.upsertGoogleUser(ctx, identity)
		}
		return nil, err
	}
	user.GoogleID = &gid
	user.FirstName = first
	user.LastName = last
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return s.userRepo.FindByID(ctx, user.ID)
}

func googleDisplayName(identity *GoogleIdentity) (first, last string) {
	if identity.Given != "" {
		return identity.Given, identity.Family
	}
	name := strings.TrimSpace(identity.Name)
	if name == "" {
		local, _, _ := strings.Cut(identity.Email, "@")
		name = strings.TrimSpace(local)
	}
	if name == "" {
		name = "Google User"
	}
	return name, ""
}
