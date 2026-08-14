package repository

import (
	"context"

	"clap/internal/modules/match/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchDetailsRepository interface {
	ReplaceStats(ctx context.Context, matchID uuid.UUID, stats []models.MatchStat) error
	ListStats(ctx context.Context, matchID uuid.UUID) ([]models.MatchStat, error)
	ReplaceTimeline(ctx context.Context, matchID uuid.UUID, events []models.MatchTimelineEvent) error
	ListTimeline(ctx context.Context, matchID uuid.UUID) ([]models.MatchTimelineEvent, error)
	ReplaceLineup(ctx context.Context, matchID uuid.UUID, players []models.MatchLineupPlayer) error
	ListLineup(ctx context.Context, matchID uuid.UUID) ([]models.MatchLineupPlayer, error)
}

type matchDetailsRepository struct {
	db *gorm.DB
}

func NewMatchDetailsRepository(db *gorm.DB) MatchDetailsRepository {
	return &matchDetailsRepository{db: db}
}

func (r *matchDetailsRepository) ReplaceStats(ctx context.Context, matchID uuid.UUID, stats []models.MatchStat) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("match_id = ?", matchID).Delete(&models.MatchStat{}).Error; err != nil {
			return sharederrors.NewInternal("Failed to replace match stats", err)
		}
		if len(stats) == 0 {
			return nil
		}
		if err := tx.Create(&stats).Error; err != nil {
			return sharederrors.NewInternal("Failed to save match stats", err)
		}
		return nil
	})
}

func (r *matchDetailsRepository) ListStats(ctx context.Context, matchID uuid.UUID) ([]models.MatchStat, error) {
	var stats []models.MatchStat
	if err := r.db.WithContext(ctx).Where("match_id = ?", matchID).Order("sort_order ASC").Find(&stats).Error; err != nil {
		return nil, sharederrors.NewInternal("Failed to load match stats", err)
	}
	return stats, nil
}

func (r *matchDetailsRepository) ReplaceTimeline(ctx context.Context, matchID uuid.UUID, events []models.MatchTimelineEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("match_id = ?", matchID).Delete(&models.MatchTimelineEvent{}).Error; err != nil {
			return sharederrors.NewInternal("Failed to replace match timeline", err)
		}
		if len(events) == 0 {
			return nil
		}
		if err := tx.Create(&events).Error; err != nil {
			return sharederrors.NewInternal("Failed to save match timeline", err)
		}
		return nil
	})
}

func (r *matchDetailsRepository) ListTimeline(ctx context.Context, matchID uuid.UUID) ([]models.MatchTimelineEvent, error) {
	var events []models.MatchTimelineEvent
	if err := r.db.WithContext(ctx).Where("match_id = ?", matchID).Order("sort_order ASC").Find(&events).Error; err != nil {
		return nil, sharederrors.NewInternal("Failed to load match timeline", err)
	}
	return events, nil
}

func (r *matchDetailsRepository) ReplaceLineup(ctx context.Context, matchID uuid.UUID, players []models.MatchLineupPlayer) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("match_id = ?", matchID).Delete(&models.MatchLineupPlayer{}).Error; err != nil {
			return sharederrors.NewInternal("Failed to replace match lineup", err)
		}
		if len(players) == 0 {
			return nil
		}
		if err := tx.Create(&players).Error; err != nil {
			return sharederrors.NewInternal("Failed to save match lineup", err)
		}
		return nil
	})
}

func (r *matchDetailsRepository) ListLineup(ctx context.Context, matchID uuid.UUID) ([]models.MatchLineupPlayer, error) {
	var players []models.MatchLineupPlayer
	if err := r.db.WithContext(ctx).Where("match_id = ?", matchID).Order("sort_order ASC").Find(&players).Error; err != nil {
		return nil, sharederrors.NewInternal("Failed to load match lineup", err)
	}
	return players, nil
}
