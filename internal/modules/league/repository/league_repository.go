package repository

import (
	"context"

	"clap/internal/modules/league/models"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LeagueRepository interface {
	Create(ctx context.Context, league *models.League) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.League, error)
	FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.League, int64, error)
	Update(ctx context.Context, league *models.League) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByProviderLeagueID(ctx context.Context, provider, providerLeagueID string) (*models.League, error)
}

type leagueRepository struct {
	db *gorm.DB
}

func NewLeagueRepository(db *gorm.DB) LeagueRepository {
	return &leagueRepository{db: db}
}

func (r *leagueRepository) Create(ctx context.Context, league *models.League) error {
	if err := r.db.WithContext(ctx).Create(league).Error; err != nil {
		return sharederrors.NewInternal("Failed to create league", err)
	}
	return nil
}

func (r *leagueRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.League, error) {
	var league models.League
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&league).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("League not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find league", err)
	}
	return &league, nil
}

func (r *leagueRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.League, int64, error) {
	var leagues []models.League
	var total int64

	query := r.db.WithContext(ctx).Model(&models.League{})

	// Apply filters
	if active, ok := filters["is_active"]; ok {
		if active == "true" {
			query = query.Where("is_active = ?", true)
		} else if active == "false" {
			query = query.Where("is_active = ?", false)
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count leagues", err)
	}

	// Apply sorting
	allowedSortFields := map[string]bool{
		"name":       true,
		"created_at": true,
		"updated_at": true,
		"country":    true,
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
	if err := query.Offset(offset).Limit(pageSize).Find(&leagues).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find leagues", err)
	}

	return leagues, total, nil
}

func (r *leagueRepository) Update(ctx context.Context, league *models.League) error {
	if err := r.db.WithContext(ctx).Save(league).Error; err != nil {
		return sharederrors.NewInternal("Failed to update league", err)
	}
	return nil
}

func (r *leagueRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.League{}, id).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete league", err)
	}
	return nil
}

func (r *leagueRepository) FindByProviderLeagueID(ctx context.Context, provider, providerLeagueID string) (*models.League, error) {
	var league models.League
	if err := r.db.WithContext(ctx).Where("provider = ? AND provider_league_id = ?", provider, providerLeagueID).First(&league).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, sharederrors.NewInternal("Failed to find league by provider ID", err)
	}
	return &league, nil
}
