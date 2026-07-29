package repository

import (
	"context"

	"clap/internal/modules/clubseason/models"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClubSeasonRepository interface {
	Create(ctx context.Context, clubSeason *models.ClubSeason) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.ClubSeason, error)
	FindByClubAndSeason(ctx context.Context, clubID, seasonID uuid.UUID) (*models.ClubSeason, error)
	FindClubsInSeason(ctx context.Context, seasonID uuid.UUID, page, pageSize int) ([]models.ClubSeason, int64, error)
	FindSeasonsForClub(ctx context.Context, clubID uuid.UUID, page, pageSize int) ([]models.ClubSeason, int64, error)
	Update(ctx context.Context, clubSeason *models.ClubSeason) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByClubAndSeason(ctx context.Context, clubID, seasonID uuid.UUID) error
}

type clubSeasonRepository struct {
	db *gorm.DB
}

func NewClubSeasonRepository(db *gorm.DB) ClubSeasonRepository {
	return &clubSeasonRepository{db: db}
}

func (r *clubSeasonRepository) Create(ctx context.Context, clubSeason *models.ClubSeason) error {
	if err := r.db.WithContext(ctx).Create(clubSeason).Error; err != nil {
		return sharederrors.NewInternal("Failed to create club season", err)
	}
	return nil
}

func (r *clubSeasonRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.ClubSeason, error) {
	var clubSeason models.ClubSeason
	if err := r.db.WithContext(ctx).Preload("Club").Preload("Season").Where("id = ?", id).First(&clubSeason).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Club season not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find club season", err)
	}
	return &clubSeason, nil
}

func (r *clubSeasonRepository) FindByClubAndSeason(ctx context.Context, clubID, seasonID uuid.UUID) (*models.ClubSeason, error) {
	var clubSeason models.ClubSeason
	if err := r.db.WithContext(ctx).Where("club_id = ? AND season_id = ?", clubID, seasonID).First(&clubSeason).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, sharederrors.NewInternal("Failed to find club season", err)
	}
	return &clubSeason, nil
}

func (r *clubSeasonRepository) FindClubsInSeason(ctx context.Context, seasonID uuid.UUID, page, pageSize int) ([]models.ClubSeason, int64, error) {
	var clubSeasons []models.ClubSeason
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ClubSeason{}).Where("season_id = ?", seasonID).Preload("Club")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count club seasons", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Order("joined_at ASC").Find(&clubSeasons).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find club seasons", err)
	}

	return clubSeasons, total, nil
}

func (r *clubSeasonRepository) FindSeasonsForClub(ctx context.Context, clubID uuid.UUID, page, pageSize int) ([]models.ClubSeason, int64, error) {
	var clubSeasons []models.ClubSeason
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ClubSeason{}).Where("club_id = ?", clubID).Preload("Season")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count club seasons", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Order("joined_at DESC").Find(&clubSeasons).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find club seasons", err)
	}

	return clubSeasons, total, nil
}

func (r *clubSeasonRepository) Update(ctx context.Context, clubSeason *models.ClubSeason) error {
	if err := r.db.WithContext(ctx).Save(clubSeason).Error; err != nil {
		return sharederrors.NewInternal("Failed to update club season", err)
	}
	return nil
}

func (r *clubSeasonRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.ClubSeason{}, id).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete club season", err)
	}
	return nil
}

func (r *clubSeasonRepository) DeleteByClubAndSeason(ctx context.Context, clubID, seasonID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("club_id = ? AND season_id = ?", clubID, seasonID).Delete(&models.ClubSeason{}).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete club season", err)
	}
	return nil
}
