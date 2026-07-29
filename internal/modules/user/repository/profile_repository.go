package repository

import (
	"clap/internal/modules/user/models"
	"clap/internal/shared/database"
	"clap/internal/shared/errors"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProfileRepository interface {
	Create(ctx context.Context, profile *models.Profile) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Profile, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*models.Profile, error)
	// FindByUserIDs batch-loads profiles keyed by user ID (avoids N+1 in
	// leaderboard rendering). Missing profiles are simply absent from the map.
	FindByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Profile, error)
	Update(ctx context.Context, profile *models.Profile) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository() ProfileRepository {
	return &profileRepository{
		db: database.GetDB(),
	}
}

func (r *profileRepository) Create(ctx context.Context, profile *models.Profile) error {
	if err := r.db.WithContext(ctx).Create(profile).Error; err != nil {
		return errors.NewInternal("Failed to create profile", err)
	}
	return nil
}

func (r *profileRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Profile, error) {
	var profile models.Profile
	err := r.db.WithContext(ctx).Preload("User").First(&profile, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find profile", err)
	}
	return &profile, nil
}

func (r *profileRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	var profile models.Profile
	err := r.db.WithContext(ctx).Preload("User").First(&profile, "user_id = ?", userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find profile", err)
	}
	return &profile, nil
}

func (r *profileRepository) FindByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*models.Profile, error) {
	result := make(map[uuid.UUID]*models.Profile, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	var profiles []models.Profile
	if err := r.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&profiles).Error; err != nil {
		return nil, errors.NewInternal("Failed to load profiles", err)
	}
	for i := range profiles {
		result[profiles[i].UserID] = &profiles[i]
	}
	return result, nil
}

func (r *profileRepository) Update(ctx context.Context, profile *models.Profile) error {
	if err := r.db.WithContext(ctx).Save(profile).Error; err != nil {
		return errors.NewInternal("Failed to update profile", err)
	}
	return nil
}

func (r *profileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Profile{}, "id = ?", id).Error; err != nil {
		return errors.NewInternal("Failed to delete profile", err)
	}
	return nil
}
