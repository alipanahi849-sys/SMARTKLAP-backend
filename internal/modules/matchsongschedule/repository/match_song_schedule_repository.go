package repository

import (
	"context"

	"clap/internal/modules/matchsongschedule/models"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchSongScheduleRepository interface {
	Create(ctx context.Context, schedule *models.MatchSongSchedule) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.MatchSongSchedule, error)
	FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.MatchSongSchedule, int64, error)
	FindByMatchID(ctx context.Context, matchID uuid.UUID, page, pageSize int) ([]models.MatchSongSchedule, int64, error)
	FindBySongID(ctx context.Context, songID uuid.UUID, page, pageSize int) ([]models.MatchSongSchedule, int64, error)
	Update(ctx context.Context, schedule *models.MatchSongSchedule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type matchSongScheduleRepository struct {
	db *gorm.DB
}

func NewMatchSongScheduleRepository(db *gorm.DB) MatchSongScheduleRepository {
	return &matchSongScheduleRepository{db: db}
}

func (r *matchSongScheduleRepository) Create(ctx context.Context, schedule *models.MatchSongSchedule) error {
	if err := r.db.WithContext(ctx).Create(schedule).Error; err != nil {
		return sharederrors.NewInternal("Failed to create match song schedule", err)
	}
	return nil
}

func (r *matchSongScheduleRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.MatchSongSchedule, error) {
	var schedule models.MatchSongSchedule
	if err := r.db.WithContext(ctx).Preload("Match").Preload("Song").Where("id = ?", id).First(&schedule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Match song schedule not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find match song schedule", err)
	}
	return &schedule, nil
}

func (r *matchSongScheduleRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.MatchSongSchedule, int64, error) {
	var schedules []models.MatchSongSchedule
	var total int64

	query := r.db.WithContext(ctx).Model(&models.MatchSongSchedule{}).Preload("Match").Preload("Song")

	// Apply filters
	if matchID, ok := filters["match_id"]; ok {
		if id, err := uuid.Parse(matchID); err == nil {
			query = query.Where("match_id = ?", id)
		}
	}

	if songID, ok := filters["song_id"]; ok {
		if id, err := uuid.Parse(songID); err == nil {
			query = query.Where("song_id = ?", id)
		}
	}

	if eventType, ok := filters["event_type"]; ok && eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	if active, ok := filters["is_active"]; ok {
		if active == "true" {
			query = query.Where("is_active = ?", true)
		} else if active == "false" {
			query = query.Where("is_active = ?", false)
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count match song schedules", err)
	}

	// Apply sorting
	allowedSortFields := map[string]bool{
		"scheduled_time": true,
		"event_type":     true,
		"created_at":     true,
	}
	if !allowedSortFields[sortBy] {
		sortBy = "scheduled_time"
	}
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}
	query = query.Order(sortBy + " " + sortOrder)

	// Apply pagination
	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Find(&schedules).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find match song schedules", err)
	}

	return schedules, total, nil
}

func (r *matchSongScheduleRepository) FindByMatchID(ctx context.Context, matchID uuid.UUID, page, pageSize int) ([]models.MatchSongSchedule, int64, error) {
	var schedules []models.MatchSongSchedule
	var total int64

	query := r.db.WithContext(ctx).Model(&models.MatchSongSchedule{}).Where("match_id = ?", matchID).Preload("Song").Preload("Match")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count match song schedules", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Order("scheduled_time ASC").Find(&schedules).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find match song schedules", err)
	}

	return schedules, total, nil
}

func (r *matchSongScheduleRepository) FindBySongID(ctx context.Context, songID uuid.UUID, page, pageSize int) ([]models.MatchSongSchedule, int64, error) {
	var schedules []models.MatchSongSchedule
	var total int64

	query := r.db.WithContext(ctx).Model(&models.MatchSongSchedule{}).Where("song_id = ?", songID).Preload("Match").Preload("Song")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count match song schedules", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Order("scheduled_time DESC").Find(&schedules).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find match song schedules", err)
	}

	return schedules, total, nil
}

func (r *matchSongScheduleRepository) Update(ctx context.Context, schedule *models.MatchSongSchedule) error {
	if err := r.db.WithContext(ctx).Save(schedule).Error; err != nil {
		return sharederrors.NewInternal("Failed to update match song schedule", err)
	}
	return nil
}

func (r *matchSongScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.MatchSongSchedule{}, id).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete match song schedule", err)
	}
	return nil
}
