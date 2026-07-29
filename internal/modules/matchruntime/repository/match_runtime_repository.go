package repository

import (
	"context"

	"clap/internal/modules/matchruntime/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchRuntimeRepository interface {
	Create(ctx context.Context, state *models.MatchRuntimeState) error
	FindByMatchID(ctx context.Context, matchID uuid.UUID) (*models.MatchRuntimeState, error)
	// Update uses optimistic locking: it checks state.Version before writing and
	// returns a 409 Conflict error if the row was modified concurrently.
	// On success, state.Version is incremented in place.
	Update(ctx context.Context, state *models.MatchRuntimeState) error
}

type matchRuntimeRepository struct {
	db *gorm.DB
}

func NewMatchRuntimeRepository(db *gorm.DB) MatchRuntimeRepository {
	return &matchRuntimeRepository{db: db}
}

func (r *matchRuntimeRepository) Create(ctx context.Context, state *models.MatchRuntimeState) error {
	if err := r.db.WithContext(ctx).Create(state).Error; err != nil {
		return sharederrors.NewInternal("Failed to create match runtime state", err)
	}
	return nil
}

func (r *matchRuntimeRepository) FindByMatchID(ctx context.Context, matchID uuid.UUID) (*models.MatchRuntimeState, error) {
	var state models.MatchRuntimeState
	err := r.db.WithContext(ctx).Where("match_id = ?", matchID).First(&state).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Match runtime state not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find match runtime state", err)
	}
	return &state, nil
}

// Update applies optimistic locking via the version column.
// The UPDATE is gated on `version = state.Version`. If no rows are affected
// the record was modified by another writer; a Conflict error is returned and
// the caller should re-read and retry.
func (r *matchRuntimeRepository) Update(ctx context.Context, state *models.MatchRuntimeState) error {
	currentVersion := state.Version

	result := r.db.WithContext(ctx).
		Model(&models.MatchRuntimeState{}).
		Where("id = ? AND version = ?", state.ID, currentVersion).
		Updates(map[string]interface{}{
			"status":          state.Status,
			"started_at":      state.StartedAt,
			"paused_at":       state.PausedAt,
			"ended_at":        state.EndedAt,
			"total_paused_ms": state.TotalPausedMs,
			"updated_by":      state.UpdatedBy,
			"version":         gorm.Expr("version + 1"),
		})

	if result.Error != nil {
		return sharederrors.NewInternal("Failed to update match runtime state", result.Error)
	}
	if result.RowsAffected == 0 {
		return sharederrors.NewConflict(
			"Concurrent modification detected on match runtime state — please retry", nil,
		)
	}

	state.Version = currentVersion + 1
	return nil
}
