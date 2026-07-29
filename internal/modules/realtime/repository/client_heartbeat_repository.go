package repository

import (
	"context"
	"time"

	"clap/internal/modules/realtime/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClientHeartbeatRepository interface {
	Record(ctx context.Context, hb *models.ClientHeartbeat) error
	FindBySessionID(ctx context.Context, sessionID uuid.UUID, limit int) ([]*models.ClientHeartbeat, error)
	FindLatestByUser(ctx context.Context, sessionID, userID uuid.UUID) (*models.ClientHeartbeat, error)
	AverageDriftBySession(ctx context.Context, sessionID uuid.UUID) (float64, error)
	// DeleteOlderThan removes heartbeats whose created_at is before cutoff.
	// Returns the number of rows deleted.
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

type clientHeartbeatRepository struct {
	db *gorm.DB
}

func NewClientHeartbeatRepository(db *gorm.DB) ClientHeartbeatRepository {
	return &clientHeartbeatRepository{db: db}
}

func (r *clientHeartbeatRepository) Record(ctx context.Context, hb *models.ClientHeartbeat) error {
	if err := r.db.WithContext(ctx).Create(hb).Error; err != nil {
		return sharederrors.NewInternal("Failed to record heartbeat", err)
	}
	return nil
}

func (r *clientHeartbeatRepository) FindBySessionID(ctx context.Context, sessionID uuid.UUID, limit int) ([]*models.ClientHeartbeat, error) {
	var hbs []*models.ClientHeartbeat
	q := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&hbs).Error; err != nil {
		return nil, sharederrors.NewInternal("Failed to find heartbeats", err)
	}
	return hbs, nil
}

func (r *clientHeartbeatRepository) FindLatestByUser(ctx context.Context, sessionID, userID uuid.UUID) (*models.ClientHeartbeat, error) {
	var hb models.ClientHeartbeat
	err := r.db.WithContext(ctx).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Order("created_at DESC").
		First(&hb).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Heartbeat not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find heartbeat", err)
	}
	return &hb, nil
}

func (r *clientHeartbeatRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&models.ClientHeartbeat{})
	if result.Error != nil {
		return 0, sharederrors.NewInternal("Failed to delete old heartbeats", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *clientHeartbeatRepository) AverageDriftBySession(ctx context.Context, sessionID uuid.UUID) (float64, error) {
	var result struct{ Avg float64 }
	err := r.db.WithContext(ctx).
		Model(&models.ClientHeartbeat{}).
		Select("COALESCE(AVG(drift_ms), 0) AS avg").
		Where("session_id = ?", sessionID).
		Scan(&result).Error
	if err != nil {
		return 0, sharederrors.NewInternal("Failed to calculate average drift", err)
	}
	return result.Avg, nil
}
