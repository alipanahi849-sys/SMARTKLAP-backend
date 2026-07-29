package repository

import (
	"context"

	"clap/internal/modules/club/models"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClubRepository interface {
	Create(ctx context.Context, club *models.Club) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Club, error)
	// FindByIDs batch-loads clubs by primary key (avoids N+1 in aggregates).
	FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]models.Club, error)
	FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.Club, int64, error)
	Search(ctx context.Context, query string, page, pageSize int) ([]models.Club, int64, error)
	Update(ctx context.Context, club *models.Club) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type clubRepository struct {
	db *gorm.DB
}

func NewClubRepository(db *gorm.DB) ClubRepository {
	return &clubRepository{db: db}
}

func (r *clubRepository) Create(ctx context.Context, club *models.Club) error {
	if err := r.db.WithContext(ctx).Create(club).Error; err != nil {
		return sharederrors.NewInternal("Failed to create club", err)
	}
	return nil
}

func (r *clubRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Club, error) {
	var club models.Club
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&club).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Club not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find club", err)
	}
	return &club, nil
}

func (r *clubRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]models.Club, error) {
	result := make(map[uuid.UUID]models.Club, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var clubs []models.Club
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&clubs).Error; err != nil {
		return nil, sharederrors.NewInternal("Failed to load clubs", err)
	}
	for _, club := range clubs {
		result[club.ID] = club
	}
	return result, nil
}

func (r *clubRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.Club, int64, error) {
	var clubs []models.Club
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Club{})

	// Apply filters
	if active, ok := filters["is_active"]; ok {
		if active == "true" {
			query = query.Where("is_active = ?", true)
		} else if active == "false" {
			query = query.Where("is_active = ?", false)
		}
	}

	if country, ok := filters["country"]; ok && country != "" {
		query = query.Where("country = ?", country)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count clubs", err)
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
	if err := query.Offset(offset).Limit(pageSize).Find(&clubs).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find clubs", err)
	}

	return clubs, total, nil
}

func (r *clubRepository) Search(ctx context.Context, query string, page, pageSize int) ([]models.Club, int64, error) {
	var clubs []models.Club
	var total int64

	dbQuery := r.db.WithContext(ctx).Model(&models.Club{}).Where("name ILIKE ? OR short_name ILIKE ?", "%"+query+"%", "%"+query+"%")

	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count clubs", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := dbQuery.Offset(offset).Limit(pageSize).Order("name ASC").Find(&clubs).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to search clubs", err)
	}

	return clubs, total, nil
}

func (r *clubRepository) Update(ctx context.Context, club *models.Club) error {
	if err := r.db.WithContext(ctx).Save(club).Error; err != nil {
		return sharederrors.NewInternal("Failed to update club", err)
	}
	return nil
}

func (r *clubRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Club{}, id).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete club", err)
	}
	return nil
}
