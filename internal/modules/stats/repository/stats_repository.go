package repository

import (
	"context"

	"clap/internal/modules/stats/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlayerRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.Player, error)
	// FindByClubIDs loads all active players of the given clubs in one query
	// (used to render both squads of a match without N+1).
	FindByClubIDs(ctx context.Context, clubIDs []uuid.UUID) ([]models.Player, error)
}

type MatchStatsRepository interface {
	StatsByMatch(ctx context.Context, matchID uuid.UUID) ([]models.MatchStat, error)
	TimelineByMatch(ctx context.Context, matchID uuid.UUID) ([]models.MatchTimelineEvent, error)
}

type playerRepository struct {
	db *gorm.DB
}

func NewPlayerRepository(db *gorm.DB) PlayerRepository {
	return &playerRepository{db: db}
}

func (r *playerRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Player, error) {
	var player models.Player
	err := r.db.WithContext(ctx).First(&player, "id = ? AND is_active = ?", id, true).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Player not found", nil)
		}
		return nil, errors.NewInternal("Failed to find player", err)
	}
	return &player, nil
}

func (r *playerRepository) FindByClubIDs(ctx context.Context, clubIDs []uuid.UUID) ([]models.Player, error) {
	if len(clubIDs) == 0 {
		return nil, nil
	}
	var players []models.Player
	if err := r.db.WithContext(ctx).
		Where("club_id IN ? AND is_active = ?", clubIDs, true).
		Order("position ASC, jersey_number ASC").
		Find(&players).Error; err != nil {
		return nil, errors.NewInternal("Failed to load players", err)
	}
	return players, nil
}

type matchStatsRepository struct {
	db *gorm.DB
}

func NewMatchStatsRepository(db *gorm.DB) MatchStatsRepository {
	return &matchStatsRepository{db: db}
}

func (r *matchStatsRepository) StatsByMatch(ctx context.Context, matchID uuid.UUID) ([]models.MatchStat, error) {
	var stats []models.MatchStat
	if err := r.db.WithContext(ctx).
		Where("match_id = ?", matchID).
		Order("sort_order ASC").
		Find(&stats).Error; err != nil {
		return nil, errors.NewInternal("Failed to load match stats", err)
	}
	return stats, nil
}

func (r *matchStatsRepository) TimelineByMatch(ctx context.Context, matchID uuid.UUID) ([]models.MatchTimelineEvent, error) {
	var events []models.MatchTimelineEvent
	if err := r.db.WithContext(ctx).
		Where("match_id = ?", matchID).
		Order("sort_order ASC").
		Find(&events).Error; err != nil {
		return nil, errors.NewInternal("Failed to load match timeline", err)
	}
	return events, nil
}
