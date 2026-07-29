package utils

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrUnauthorized  = errors.New("unauthorized access")
	ErrForbidden     = errors.New("forbidden access")
	ErrClubAdminOnly = errors.New("only club admin can perform this action")
	ErrAdminOnly     = errors.New("only admin can perform this action")
	ErrInvalidClub   = errors.New("invalid club access")
)

type UserRole string

const (
	RoleAdmin     UserRole = "admin"
	RoleClubAdmin UserRole = "club_admin"
	RoleUser      UserRole = "user"
)

type AuthorizationContext struct {
	UserID      uuid.UUID
	Roles       []string
	ClubID      *uuid.UUID
	IsAdmin     bool
	IsClubAdmin bool
}

func NewAuthorizationContext(userID uuid.UUID, roles []string, clubID *uuid.UUID) *AuthorizationContext {
	ctx := &AuthorizationContext{
		UserID:      userID,
		Roles:       roles,
		ClubID:      clubID,
		IsAdmin:     false,
		IsClubAdmin: false,
	}

	for _, role := range roles {
		if role == string(RoleAdmin) {
			ctx.IsAdmin = true
		}
		if role == string(RoleClubAdmin) {
			ctx.IsClubAdmin = true
		}
	}

	return ctx
}

func (a *AuthorizationContext) CanAccessClub(clubID uuid.UUID) bool {
	if a.IsAdmin {
		return true
	}

	if a.IsClubAdmin && a.ClubID != nil && *a.ClubID == clubID {
		return true
	}

	return false
}

func (a *AuthorizationContext) CanManageClub(clubID uuid.UUID) error {
	if a.IsAdmin {
		return nil
	}

	if a.IsClubAdmin && a.ClubID != nil && *a.ClubID == clubID {
		return nil
	}

	return ErrClubAdminOnly
}

func (a *AuthorizationContext) IsAdminOrClubAdmin() bool {
	return a.IsAdmin || a.IsClubAdmin
}

func (a *AuthorizationContext) RequireAdmin() error {
	if !a.IsAdmin {
		return ErrAdminOnly
	}
	return nil
}

func (a *AuthorizationContext) RequireAdminOrClubAdmin() error {
	if !a.IsAdmin && !a.IsClubAdmin {
		return ErrUnauthorized
	}
	return nil
}
