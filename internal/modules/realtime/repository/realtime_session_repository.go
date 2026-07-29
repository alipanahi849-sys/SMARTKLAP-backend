package repository

import (
	"context"

	"clap/internal/modules/realtime/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RealtimeSessionRepository interface {
	Create(ctx context.Context, session *models.RealtimeSession) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.RealtimeSession, error)
	FindByMatchID(ctx context.Context, matchID uuid.UUID) (*models.RealtimeSession, error)
	FindActiveByMatchID(ctx context.Context, matchID uuid.UUID) (*models.RealtimeSession, error)
	// Update uses optimistic locking via session.Version; returns 409 Conflict on mismatch.
	Update(ctx context.Context, session *models.RealtimeSession) error
}

type realtimeSessionRepository struct {
	db *gorm.DB
}

func NewRealtimeSessionRepository(db *gorm.DB) RealtimeSessionRepository {
	return &realtimeSessionRepository{db: db}
}

func (r *realtimeSessionRepository) Create(ctx context.Context, session *models.RealtimeSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return sharederrors.NewInternal("Failed to create realtime session", err)
	}
	return nil
}

func (r *realtimeSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.RealtimeSession, error) {
	var s models.RealtimeSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Realtime session not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find realtime session", err)
	}
	return &s, nil
}

func (r *realtimeSessionRepository) FindByMatchID(ctx context.Context, matchID uuid.UUID) (*models.RealtimeSession, error) {
	var s models.RealtimeSession
	err := r.db.WithContext(ctx).
		Where("match_id = ?", matchID).
		Order("created_at DESC").
		First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Realtime session not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find realtime session", err)
	}
	return &s, nil
}

func (r *realtimeSessionRepository) FindActiveByMatchID(ctx context.Context, matchID uuid.UUID) (*models.RealtimeSession, error) {
	var s models.RealtimeSession
	err := r.db.WithContext(ctx).
		Where("match_id = ? AND status IN ('pending','running','paused')", matchID).
		Order("created_at DESC").
		First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("No active realtime session found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find realtime session", err)
	}
	return &s, nil
}

// Update applies optimistic locking via the version column.
// If no rows are affected the row was concurrently modified; a Conflict error is returned.
func (r *realtimeSessionRepository) Update(ctx context.Context, session *models.RealtimeSession) error {
	currentVersion := session.Version

	result := r.db.WithContext(ctx).
		Model(&models.RealtimeSession{}).
		Where("id = ? AND version = ?", session.ID, currentVersion).
		Updates(map[string]interface{}{
			"status":     session.Status,
			"started_at": session.StartedAt,
			"version":    gorm.Expr("version + 1"),
		})

	if result.Error != nil {
		return sharederrors.NewInternal("Failed to update realtime session", result.Error)
	}
	if result.RowsAffected == 0 {
		return sharederrors.NewConflict(
			"Concurrent modification detected on realtime session — please retry", nil,
		)
	}

	session.Version = currentVersion + 1
	return nil
}
