package repository

import (
	"context"
	"time"

	"clap/internal/modules/eventscheduler/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchedulerEventRepository interface {
	Create(ctx context.Context, event *models.SchedulerEvent) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.SchedulerEvent, error)
	FindPendingUpTo(ctx context.Context, upTo time.Time) ([]*models.SchedulerEvent, error)
	FindAllPending(ctx context.Context) ([]*models.SchedulerEvent, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.SchedulerEventStatus) error
	UpdateExecuteAt(ctx context.Context, id uuid.UUID, executeAt time.Time) error
	// ClaimForProcessing atomically transitions an event from pending → processing
	// using a database transaction with FOR UPDATE SKIP LOCKED.
	// Returns NotFound if the event is absent, already claimed, or not pending.
	ClaimForProcessing(ctx context.Context, id uuid.UUID) (*models.SchedulerEvent, error)
	// MarkExecuted transitions the event from processing → executed.
	MarkExecuted(ctx context.Context, id uuid.UUID) error
	// MarkFailed transitions the event from processing → failed and stores the reason.
	MarkFailed(ctx context.Context, id uuid.UUID, reason string) error
	// ResetStaleProcessing transitions events stuck in processing whose
	// updated_at is older than the cutoff back to pending so they can be
	// re-dispatched after a crash. Returns the number of rows reset.
	ResetStaleProcessing(ctx context.Context, olderThan time.Time) (int64, error)
	// DeleteTerminalOlderThan deletes executed/failed/cancelled events whose
	// updated_at is older than the cutoff. Returns the number of rows deleted.
	DeleteTerminalOlderThan(ctx context.Context, olderThan time.Time) (int64, error)
}

type schedulerEventRepository struct {
	db *gorm.DB
}

func NewSchedulerEventRepository(db *gorm.DB) SchedulerEventRepository {
	return &schedulerEventRepository{db: db}
}

func (r *schedulerEventRepository) Create(ctx context.Context, event *models.SchedulerEvent) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return sharederrors.NewInternal("Failed to create scheduler event", err)
	}
	return nil
}

func (r *schedulerEventRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.SchedulerEvent, error) {
	var ev models.SchedulerEvent
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&ev).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Scheduler event not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find scheduler event", err)
	}
	return &ev, nil
}

func (r *schedulerEventRepository) FindPendingUpTo(ctx context.Context, upTo time.Time) ([]*models.SchedulerEvent, error) {
	var events []*models.SchedulerEvent
	err := r.db.WithContext(ctx).
		Where("status = ? AND execute_at <= ?", models.SchedulerEventPending, upTo).
		Order("execute_at ASC").
		Find(&events).Error
	if err != nil {
		return nil, sharederrors.NewInternal("Failed to find pending scheduler events", err)
	}
	return events, nil
}

func (r *schedulerEventRepository) FindAllPending(ctx context.Context) ([]*models.SchedulerEvent, error) {
	var events []*models.SchedulerEvent
	err := r.db.WithContext(ctx).
		Where("status = ?", models.SchedulerEventPending).
		Order("execute_at ASC").
		Find(&events).Error
	if err != nil {
		return nil, sharederrors.NewInternal("Failed to find all pending scheduler events", err)
	}
	return events, nil
}

func (r *schedulerEventRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.SchedulerEventStatus) error {
	result := r.db.WithContext(ctx).
		Model(&models.SchedulerEvent{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return sharederrors.NewInternal("Failed to update event status", result.Error)
	}
	if result.RowsAffected == 0 {
		return sharederrors.NewNotFound("Scheduler event not found", nil)
	}
	return nil
}

func (r *schedulerEventRepository) UpdateExecuteAt(ctx context.Context, id uuid.UUID, executeAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&models.SchedulerEvent{}).
		Where("id = ? AND status = ?", id, models.SchedulerEventPending).
		Update("execute_at", executeAt)
	if result.Error != nil {
		return sharederrors.NewInternal("Failed to reschedule event", result.Error)
	}
	if result.RowsAffected == 0 {
		return sharederrors.NewNotFound("Pending scheduler event not found", nil)
	}
	return nil
}

// ClaimForProcessing uses a database transaction with FOR UPDATE SKIP LOCKED
// to atomically claim a pending event for exclusive processing.
//
// If two workers race on the same event, only one succeeds; the other gets a
// NotFound error (the row was skipped/locked). This prevents double-execution.
func (r *schedulerEventRepository) ClaimForProcessing(ctx context.Context, id uuid.UUID) (*models.SchedulerEvent, error) {
	var claimed *models.SchedulerEvent

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ev models.SchedulerEvent

		// Lock the specific row; SKIP LOCKED means concurrent callers get nothing
		// rather than blocking.
		if err := tx.Raw(`
			SELECT * FROM scheduler_events
			WHERE id = ? AND status = 'pending'
			FOR UPDATE SKIP LOCKED`,
			id,
		).Scan(&ev).Error; err != nil {
			return sharederrors.NewInternal("Failed to claim scheduler event", err)
		}

		if ev.ID == uuid.Nil {
			// Row not found or already locked by another worker.
			return sharederrors.NewNotFound("Event not available for processing", nil)
		}

		// Transition pending → processing.
		if err := tx.Model(&ev).Update("status", models.SchedulerEventProcessing).Error; err != nil {
			return sharederrors.NewInternal("Failed to mark event as processing", err)
		}

		ev.Status = models.SchedulerEventProcessing
		claimed = &ev
		return nil
	})

	if err != nil {
		return nil, err
	}
	return claimed, nil
}

func (r *schedulerEventRepository) MarkExecuted(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&models.SchedulerEvent{}).
		Where("id = ? AND status = ?", id, models.SchedulerEventProcessing).
		Update("status", models.SchedulerEventExecuted)
	if result.Error != nil {
		return sharederrors.NewInternal("Failed to mark event as executed", result.Error)
	}
	if result.RowsAffected == 0 {
		return sharederrors.NewNotFound("Processing scheduler event not found", nil)
	}
	return nil
}

func (r *schedulerEventRepository) MarkFailed(ctx context.Context, id uuid.UUID, reason string) error {
	result := r.db.WithContext(ctx).
		Model(&models.SchedulerEvent{}).
		Where("id = ? AND status = ?", id, models.SchedulerEventProcessing).
		Updates(map[string]interface{}{
			"status":      models.SchedulerEventFailed,
			"fail_reason": reason,
		})
	if result.Error != nil {
		return sharederrors.NewInternal("Failed to mark event as failed", result.Error)
	}
	if result.RowsAffected == 0 {
		return sharederrors.NewNotFound("Processing scheduler event not found", nil)
	}
	return nil
}

// ResetStaleProcessing reclaims events orphaned in the processing state by a
// crashed dispatcher. The partial index idx_scheduler_events_processing on
// (status, updated_at) WHERE status = 'processing' serves this query.
func (r *schedulerEventRepository) ResetStaleProcessing(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&models.SchedulerEvent{}).
		Where("status = ? AND updated_at < ?", models.SchedulerEventProcessing, olderThan).
		Updates(map[string]interface{}{
			"status":      models.SchedulerEventPending,
			"fail_reason": "",
		})
	if result.Error != nil {
		return 0, sharederrors.NewInternal("Failed to reset stale processing events", result.Error)
	}
	return result.RowsAffected, nil
}

// DeleteTerminalOlderThan permanently removes scheduler events in a terminal
// state (executed/failed/cancelled) older than the cutoff. Used by the
// retention job. The idx_scheduler_events_status index assists this query.
func (r *schedulerEventRepository) DeleteTerminalOlderThan(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Unscoped().
		Where("status IN ? AND updated_at < ?",
			[]models.SchedulerEventStatus{
				models.SchedulerEventExecuted,
				models.SchedulerEventFailed,
				models.SchedulerEventCancelled,
			},
			olderThan,
		).
		Delete(&models.SchedulerEvent{})
	if result.Error != nil {
		return 0, sharederrors.NewInternal("Failed to delete terminal scheduler events", result.Error)
	}
	return result.RowsAffected, nil
}
