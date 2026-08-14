package repository

import (
	"context"
	"time"

	"clap/internal/modules/notification/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PushDeviceRepository interface {
	Upsert(ctx context.Context, device *models.PushDevice) (*models.PushDevice, error)
	DeleteByUserAndToken(ctx context.Context, userID uuid.UUID, token string) error
	ListTokens(ctx context.Context) ([]string, error)
	DeleteByTokens(ctx context.Context, tokens []string) error
}

type pushDeviceRepository struct {
	db *gorm.DB
}

func NewPushDeviceRepository(db *gorm.DB) PushDeviceRepository {
	return &pushDeviceRepository{db: db}
}

func (r *pushDeviceRepository) Upsert(ctx context.Context, device *models.PushDevice) (*models.PushDevice, error) {
	now := time.Now().UTC()
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}
	device.UpdatedAt = now

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "fcm_token"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"user_id":    device.UserID,
				"platform":   device.Platform,
				"updated_at": now,
			}),
		}).
		Create(device).Error
	if err != nil {
		return nil, errors.NewInternal("Failed to save push device", err)
	}

	var stored models.PushDevice
	if err := r.db.WithContext(ctx).
		Where("fcm_token = ?", device.FCMToken).
		First(&stored).Error; err != nil {
		return nil, errors.NewInternal("Failed to load push device", err)
	}
	return &stored, nil
}

func (r *pushDeviceRepository) DeleteByUserAndToken(ctx context.Context, userID uuid.UUID, token string) error {
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND fcm_token = ?", userID, token).
		Delete(&models.PushDevice{}).Error
	if err != nil {
		return errors.NewInternal("Failed to delete push device", err)
	}
	return nil
}

func (r *pushDeviceRepository) ListTokens(ctx context.Context) ([]string, error) {
	var tokens []string
	err := r.db.WithContext(ctx).
		Model(&models.PushDevice{}).
		Pluck("fcm_token", &tokens).Error
	if err != nil {
		return nil, errors.NewInternal("Failed to list push devices", err)
	}
	return tokens, nil
}

func (r *pushDeviceRepository) DeleteByTokens(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	err := r.db.WithContext(ctx).
		Where("fcm_token IN ?", tokens).
		Delete(&models.PushDevice{}).Error
	if err != nil {
		return errors.NewInternal("Failed to delete stale push devices", err)
	}
	return nil
}
