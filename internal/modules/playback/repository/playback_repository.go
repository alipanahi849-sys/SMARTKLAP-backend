package repository

import (
	"context"
	"time"

	"clap/internal/modules/playback/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlaybackRepository interface {
	Create(ctx context.Context, schedule *models.PlaybackSchedule) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.PlaybackSchedule, error)
	FindByMatchID(ctx context.Context, matchID uuid.UUID) ([]*models.PlaybackSchedule, error)
	FindUpcoming(ctx context.Context, matchID uuid.UUID, after time.Time) ([]*models.PlaybackSchedule, error)
	// HasOverlap returns true if the proposed [start, start+durationMs) window
	// overlaps with any non-cancelled schedule for the same match.
	// Schedules with durationMs == 0 are skipped in the overlap check on the DB side.
	HasOverlap(ctx context.Context, matchID uuid.UUID, start time.Time, durationMs int64, excludeID *uuid.UUID) (bool, error)
	// UpdateStatus uses optimistic locking via schedule.Version.
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.PlaybackStatus) error
	// Update applies optimistic locking; returns 409 Conflict on version mismatch.
	Update(ctx context.Context, schedule *models.PlaybackSchedule) error
}

type playbackRepository struct {
	db *gorm.DB
}

func NewPlaybackRepository(db *gorm.DB) PlaybackRepository {
	return &playbackRepository{db: db}
}

func (r *playbackRepository) Create(ctx context.Context, s *models.PlaybackSchedule) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return sharederrors.NewInternal("Failed to create playback schedule", err)
	}
	return nil
}

func (r *playbackRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.PlaybackSchedule, error) {
	var s models.PlaybackSchedule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Playback schedule not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find playback schedule", err)
	}
	return &s, nil
}

func (r *playbackRepository) FindByMatchID(ctx context.Context, matchID uuid.UUID) ([]*models.PlaybackSchedule, error) {
	var schedules []*models.PlaybackSchedule
	err := r.db.WithContext(ctx).
		Where("match_id = ? AND status NOT IN ('cancelled')", matchID).
		Order("scheduled_at ASC").
		Find(&schedules).Error
	if err != nil {
		return nil, sharederrors.NewInternal("Failed to find playback schedules", err)
	}
	return schedules, nil
}

func (r *playbackRepository) FindUpcoming(ctx context.Context, matchID uuid.UUID, after time.Time) ([]*models.PlaybackSchedule, error) {
	var schedules []*models.PlaybackSchedule
	err := r.db.WithContext(ctx).
		Where("match_id = ? AND status = ? AND scheduled_at > ?", matchID, models.PlaybackStatusPending, after).
		Order("scheduled_at ASC").
		Find(&schedules).Error
	if err != nil {
		return nil, sharederrors.NewInternal("Failed to find upcoming playback schedules", err)
	}
	return schedules, nil
}

// HasOverlap checks whether the interval [start, start+durationMs) conflicts with
// any existing active schedule for the same match. The check is skipped for
// records with duration_ms = 0 (unknown duration).
//
// Overlap condition (Allen's interval logic):
//
//	existing.scheduled_at < new_end  AND  existing.ends_at > new_start
//
// where ends_at = scheduled_at + duration_ms milliseconds.
func (r *playbackRepository) HasOverlap(ctx context.Context, matchID uuid.UUID, start time.Time, durationMs int64, excludeID *uuid.UUID) (bool, error) {
	if durationMs <= 0 {
		return false, nil
	}

	endMs := start.UnixMilli() + durationMs

	query := r.db.WithContext(ctx).
		Model(&models.PlaybackSchedule{}).
		Where(`match_id = ?
			AND status NOT IN ('cancelled')
			AND deleted_at IS NULL
			AND duration_ms > 0
			AND EXTRACT(EPOCH FROM scheduled_at) * 1000 < ?
			AND EXTRACT(EPOCH FROM scheduled_at) * 1000 + duration_ms > ?`,
			matchID,
			endMs,
			start.UnixMilli(),
		)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, sharederrors.NewInternal("Failed to check playback overlap", err)
	}
	return count > 0, nil
}

func (r *playbackRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.PlaybackStatus) error {
	result := r.db.WithContext(ctx).
		Model(&models.PlaybackSchedule{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return sharederrors.NewInternal("Failed to update playback schedule status", result.Error)
	}
	if result.RowsAffected == 0 {
		return sharederrors.NewNotFound("Playback schedule not found", nil)
	}
	return nil
}

// Update applies optimistic locking via the version column.
func (r *playbackRepository) Update(ctx context.Context, schedule *models.PlaybackSchedule) error {
	currentVersion := schedule.Version

	result := r.db.WithContext(ctx).
		Model(&models.PlaybackSchedule{}).
		Where("id = ? AND version = ?", schedule.ID, currentVersion).
		Updates(map[string]interface{}{
			"status":  schedule.Status,
			"version": gorm.Expr("version + 1"),
		})

	if result.Error != nil {
		return sharederrors.NewInternal("Failed to update playback schedule", result.Error)
	}
	if result.RowsAffected == 0 {
		return sharederrors.NewConflict(
			"Concurrent modification detected on playback schedule — please retry", nil,
		)
	}

	schedule.Version = currentVersion + 1
	return nil
}
