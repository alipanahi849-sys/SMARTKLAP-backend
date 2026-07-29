package repository

import (
	"context"
	"time"

	"clap/internal/modules/realtime/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RealtimeEventRepository interface {
	Create(ctx context.Context, event *models.RealtimeEvent) error
	CreateBatch(ctx context.Context, events []*models.RealtimeEvent) error
	FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*models.RealtimeEvent, error)
	FindBySessionIDOrdered(ctx context.Context, sessionID uuid.UUID) ([]*models.RealtimeEvent, error)
	DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error
	// DeleteOlderThan removes realtime_events whose created_at is before the
	// cutoff. Returns the number of rows deleted. Used by the retention job.
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type realtimeEventRepository struct {
	db *gorm.DB
}

func NewRealtimeEventRepository(db *gorm.DB) RealtimeEventRepository {
	return &realtimeEventRepository{db: db}
}

func (r *realtimeEventRepository) Create(ctx context.Context, event *models.RealtimeEvent) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return sharederrors.NewInternal("Failed to create realtime event", err)
	}
	return nil
}

func (r *realtimeEventRepository) CreateBatch(ctx context.Context, events []*models.RealtimeEvent) error {
	if len(events) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).CreateInBatches(events, 100).Error; err != nil {
		return sharederrors.NewInternal("Failed to batch-create realtime events", err)
	}
	return nil
}

func (r *realtimeEventRepository) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*models.RealtimeEvent, error) {
	var events []*models.RealtimeEvent
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Find(&events).Error; err != nil {
		return nil, sharederrors.NewInternal("Failed to find realtime events", err)
	}
	return events, nil
}

func (r *realtimeEventRepository) FindBySessionIDOrdered(ctx context.Context, sessionID uuid.UUID) ([]*models.RealtimeEvent, error) {
	var events []*models.RealtimeEvent
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("execute_at_ms ASC").
		Find(&events).Error
	if err != nil {
		return nil, sharederrors.NewInternal("Failed to find ordered realtime events", err)
	}
	return events, nil
}

func (r *realtimeEventRepository) DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&models.RealtimeEvent{}).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete realtime events", err)
	}
	return nil
}

func (r *realtimeEventRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&models.RealtimeEvent{})
	if result.Error != nil {
		return 0, sharederrors.NewInternal("Failed to delete old realtime events", result.Error)
	}
	return result.RowsAffected, nil
}
