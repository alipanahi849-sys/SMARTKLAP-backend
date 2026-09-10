package repository

import (
	"context"
	"time"

	"clap/internal/modules/adminauth/models"
	"clap/internal/shared/database"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminUserRepository interface {
	Create(ctx context.Context, user *models.AdminUser) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.AdminUser, error)
	FindByEmail(ctx context.Context, email string) (*models.AdminUser, error)
	Update(ctx context.Context, user *models.AdminUser) error
}

type adminUserRepository struct {
	db *gorm.DB
}

func NewAdminUserRepository() AdminUserRepository {
	return &adminUserRepository{db: database.GetDB()}
}

func (r *adminUserRepository) Create(ctx context.Context, user *models.AdminUser) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return errors.NewInternal("Failed to create admin user", err)
	}
	return nil
}

func (r *adminUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.AdminUser, error) {
	var user models.AdminUser
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find admin user", err)
	}
	return &user, nil
}

func (r *adminUserRepository) FindByEmail(ctx context.Context, email string) (*models.AdminUser, error) {
	var user models.AdminUser
	err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find admin user", err)
	}
	return &user, nil
}

func (r *adminUserRepository) Update(ctx context.Context, user *models.AdminUser) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return errors.NewInternal("Failed to update admin user", err)
	}
	return nil
}

type AdminRefreshTokenRepository interface {
	Create(ctx context.Context, token *models.AdminRefreshToken) error
	FindByToken(ctx context.Context, tokenHash string) (*models.AdminRefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForAdmin(ctx context.Context, adminID uuid.UUID) error
}

type adminRefreshTokenRepository struct {
	db *gorm.DB
}

func NewAdminRefreshTokenRepository() AdminRefreshTokenRepository {
	return &adminRefreshTokenRepository{db: database.GetDB()}
}

func (r *adminRefreshTokenRepository) Create(ctx context.Context, token *models.AdminRefreshToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return errors.NewInternal("Failed to create admin refresh token", err)
	}
	return nil
}

func (r *adminRefreshTokenRepository) FindByToken(ctx context.Context, tokenHash string) (*models.AdminRefreshToken, error) {
	var token models.AdminRefreshToken
	err := r.db.WithContext(ctx).First(&token, "token = ?", tokenHash).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find admin refresh token", err)
	}
	return &token, nil
}

func (r *adminRefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&models.AdminRefreshToken{}).Where("id = ?", id).Update("revoked_at", now).Error; err != nil {
		return errors.NewInternal("Failed to revoke admin refresh token", err)
	}
	return nil
}

func (r *adminRefreshTokenRepository) RevokeAllForAdmin(ctx context.Context, adminID uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&models.AdminRefreshToken{}).Where("admin_user_id = ?", adminID).Update("revoked_at", now).Error; err != nil {
		return errors.NewInternal("Failed to revoke admin refresh tokens", err)
	}
	return nil
}
