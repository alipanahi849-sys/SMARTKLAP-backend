package repository

import (
	"context"

	"clap/internal/modules/season/models"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SeasonRepository interface {
	Create(ctx context.Context, season *models.Season) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Season, error)
	FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.Season, int64, error)
	FindByLeagueID(ctx context.Context, leagueID uuid.UUID, page, pageSize int) ([]models.Season, int64, error)
	FindByLeagueAndProviderSeason(ctx context.Context, leagueID uuid.UUID, providerSeasonID string) (*models.Season, error)
	Update(ctx context.Context, season *models.Season) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type seasonRepository struct {
	db *gorm.DB
}

func NewSeasonRepository(db *gorm.DB) SeasonRepository {
	return &seasonRepository{db: db}
}

func (r *seasonRepository) Create(ctx context.Context, season *models.Season) error {
	if err := r.db.WithContext(ctx).Create(season).Error; err != nil {
		return sharederrors.NewInternal("Failed to create season", err)
	}
	return nil
}

func (r *seasonRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Season, error) {
	var season models.Season
	if err := r.db.WithContext(ctx).Preload("League").Where("id = ?", id).First(&season).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Season not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find season", err)
	}
	return &season, nil
}

func (r *seasonRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.Season, int64, error) {
	var seasons []models.Season
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Season{}).Preload("League")

	// Apply filters
	if leagueID, ok := filters["league_id"]; ok {
		if id, err := uuid.Parse(leagueID); err == nil {
			query = query.Where("league_id = ?", id)
		}
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
		return nil, 0, sharederrors.NewInternal("Failed to count seasons", err)
	}

	// Apply sorting
	allowedSortFields := map[string]bool{
		"name":       true,
		"created_at": true,
		"updated_at": true,
		"start_date": true,
		"end_date":   true,
	}
	if !allowedSortFields[sortBy] {
		sortBy = "created_at"
	}
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}
	query = query.Order(sortBy + " " + sortOrder)

	// Apply pagination
	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Find(&seasons).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find seasons", err)
	}

	return seasons, total, nil
}

func (r *seasonRepository) FindByLeagueID(ctx context.Context, leagueID uuid.UUID, page, pageSize int) ([]models.Season, int64, error) {
	var seasons []models.Season
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Season{}).Where("league_id = ?", leagueID).Preload("League")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count seasons", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Order("start_date DESC").Find(&seasons).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find seasons", err)
	}

	return seasons, total, nil
}

func (r *seasonRepository) FindByLeagueAndProviderSeason(ctx context.Context, leagueID uuid.UUID, providerSeasonID string) (*models.Season, error) {
	var season models.Season
	if err := r.db.WithContext(ctx).Where("league_id = ? AND provider_season_id = ?", leagueID, providerSeasonID).First(&season).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, sharederrors.NewInternal("Failed to find season by provider ID", err)
	}
	return &season, nil
}

func (r *seasonRepository) Update(ctx context.Context, season *models.Season) error {
	if err := r.db.WithContext(ctx).Save(season).Error; err != nil {
		return sharederrors.NewInternal("Failed to update season", err)
	}
	return nil
}

func (r *seasonRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Season{}, id).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete season", err)
	}
	return nil
}
