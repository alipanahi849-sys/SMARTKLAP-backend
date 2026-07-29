package repository

import (
	"clap/internal/modules/auth/models"
	"clap/internal/shared/database"
	"clap/internal/shared/errors"
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error)
	FindByToken(ctx context.Context, token string) (*models.RefreshToken, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.RefreshToken, error)
	Update(ctx context.Context, token *models.RefreshToken) error
	Delete(ctx context.Context, id uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

type refreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository() RefreshTokenRepository {
	return &refreshTokenRepository{
		db: database.GetDB(),
	}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return errors.NewInternal("Failed to create refresh token", err)
	}
	return nil
}

func (r *refreshTokenRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := r.db.WithContext(ctx).First(&token, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find refresh token", err)
	}
	return &token, nil
}

func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken
	err := r.db.WithContext(ctx).Preload("User").Preload("User.Roles").First(&refreshToken, "token = ?", token).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find refresh token", err)
	}
	return &refreshToken, nil
}

func (r *refreshTokenRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.RefreshToken, error) {
	var tokens []models.RefreshToken
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&tokens).Error
	if err != nil {
		return nil, errors.NewInternal("Failed to find refresh tokens", err)
	}
	return tokens, nil
}

func (r *refreshTokenRepository) Update(ctx context.Context, token *models.RefreshToken) error {
	if err := r.db.WithContext(ctx).Save(token).Error; err != nil {
		return errors.NewInternal("Failed to update refresh token", err)
	}
	return nil
}

func (r *refreshTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.RefreshToken{}, "id = ?", id).Error; err != nil {
		return errors.NewInternal("Failed to delete refresh token", err)
	}
	return nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&models.RefreshToken{}).Where("id = ?", id).Update("revoked_at", now).Error; err != nil {
		return errors.NewInternal("Failed to revoke refresh token", err)
	}
	return nil
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).Model(&models.RefreshToken{}).Where("user_id = ?", userID).Update("revoked_at", now).Error; err != nil {
		return errors.NewInternal("Failed to revoke refresh tokens", err)
	}
	return nil
}

func (r *refreshTokenRepository) DeleteExpired(ctx context.Context) error {
	if err := r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{}).Error; err != nil {
		return errors.NewInternal("Failed to delete expired refresh tokens", err)
	}
	return nil
}
