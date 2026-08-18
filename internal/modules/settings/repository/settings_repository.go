package repository

import (
	"context"

	"clap/internal/modules/settings/models"
	sharederrors "clap/internal/shared/errors"

	"gorm.io/gorm"
)

type SettingsRepository interface {
	Get(ctx context.Context) (*models.AppSettings, error)
	Save(ctx context.Context, settings *models.AppSettings) error
}

type settingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) Get(ctx context.Context) (*models.AppSettings, error) {
	var settings models.AppSettings
	if err := r.db.WithContext(ctx).Preload("FeaturedClub").Preload("NewsClub").First(&settings, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			settings = models.AppSettings{ID: 1}
			if createErr := r.db.WithContext(ctx).Create(&settings).Error; createErr != nil {
				return nil, sharederrors.NewInternal("Failed to initialize app settings", createErr)
			}
			return &settings, nil
		}
		return nil, sharederrors.NewInternal("Failed to load app settings", err)
	}
	return &settings, nil
}

func (r *settingsRepository) Save(ctx context.Context, settings *models.AppSettings) error {
	settings.ID = 1
	if err := r.db.WithContext(ctx).Save(settings).Error; err != nil {
		return sharederrors.NewInternal("Failed to save app settings", err)
	}
	return nil
}
