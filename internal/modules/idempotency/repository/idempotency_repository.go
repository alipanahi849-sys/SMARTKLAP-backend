package repository

import (
	"context"
	"time"

	"clap/internal/modules/idempotency/models"
	sharederrors "clap/internal/shared/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IdempotencyRepository interface {
	// FindByKeyAndEndpoint looks up a non-expired record.
	// Returns NotFound if absent or expired.
	FindByKeyAndEndpoint(ctx context.Context, key, endpoint string) (*models.IdempotencyKey, error)
	// CreateOrIgnore inserts the record. If a duplicate (key + endpoint) already
	// exists the INSERT is silently ignored (ON CONFLICT DO NOTHING), so the
	// caller can immediately call FindByKeyAndEndpoint to retrieve the winner.
	CreateOrIgnore(ctx context.Context, record *models.IdempotencyKey) error
	// DeleteExpired removes all records past their expires_at timestamp.
	// Returns the number of rows deleted.
	DeleteExpired(ctx context.Context) (int64, error)
}

type idempotencyRepository struct {
	db *gorm.DB
}

func NewIdempotencyRepository(db *gorm.DB) IdempotencyRepository {
	return &idempotencyRepository{db: db}
}

func (r *idempotencyRepository) FindByKeyAndEndpoint(ctx context.Context, key, endpoint string) (*models.IdempotencyKey, error) {
	var rec models.IdempotencyKey
	err := r.db.WithContext(ctx).
		Where("key = ? AND endpoint = ? AND expires_at > ?", key, endpoint, time.Now().UTC()).
		First(&rec).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Idempotency key not found or expired", nil)
		}
		return nil, sharederrors.NewInternal("Failed to look up idempotency key", err)
	}
	return &rec, nil
}

func (r *idempotencyRepository) CreateOrIgnore(ctx context.Context, record *models.IdempotencyKey) error {
	if record.ID.String() == "00000000-0000-0000-0000-000000000000" {
		record.BeforeCreate(nil) //nolint:errcheck
	}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(record).Error
	if err != nil {
		return sharederrors.NewInternal("Failed to store idempotency key", err)
	}
	return nil
}

func (r *idempotencyRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now().UTC()).
		Delete(&models.IdempotencyKey{})
	if result.Error != nil {
		return 0, sharederrors.NewInternal("Failed to delete expired idempotency keys", result.Error)
	}
	return result.RowsAffected, nil
}
