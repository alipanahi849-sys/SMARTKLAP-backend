package repository

import (
	"context"

	"clap/internal/modules/song/models"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SongRepository interface {
	Create(ctx context.Context, song *models.Song) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Song, error)
	FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.Song, int64, error)
	Search(ctx context.Context, query string, page, pageSize int) ([]models.Song, int64, error)
	Update(ctx context.Context, song *models.Song) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type songRepository struct {
	db *gorm.DB
}

func NewSongRepository(db *gorm.DB) SongRepository {
	return &songRepository{db: db}
}

func (r *songRepository) Create(ctx context.Context, song *models.Song) error {
	if err := r.db.WithContext(ctx).Create(song).Error; err != nil {
		return sharederrors.NewInternal("Failed to create song", err)
	}
	return nil
}

func (r *songRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Song, error) {
	var song models.Song
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&song).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Song not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find song", err)
	}
	return &song, nil
}

func (r *songRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.Song, int64, error) {
	var songs []models.Song
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Song{})

	// Apply filters
	if active, ok := filters["is_active"]; ok {
		if active == "true" {
			query = query.Where("is_active = ?", true)
		} else if active == "false" {
			query = query.Where("is_active = ?", false)
		}
	}

	if artist, ok := filters["artist"]; ok && artist != "" {
		query = query.Where("artist ILIKE ?", "%"+artist+"%")
	}

	if album, ok := filters["album"]; ok && album != "" {
		query = query.Where("album ILIKE ?", "%"+album+"%")
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count songs", err)
	}

	// Apply sorting
	allowedSortFields := map[string]bool{
		"title":      true,
		"artist":     true,
		"created_at": true,
		"updated_at": true,
		"duration":   true,
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
	if err := query.Offset(offset).Limit(pageSize).Find(&songs).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find songs", err)
	}

	return songs, total, nil
}

func (r *songRepository) Search(ctx context.Context, query string, page, pageSize int) ([]models.Song, int64, error) {
	var songs []models.Song
	var total int64

	dbQuery := r.db.WithContext(ctx).Model(&models.Song{}).Where("title ILIKE ? OR artist ILIKE ?", "%"+query+"%", "%"+query+"%")

	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count songs", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := dbQuery.Offset(offset).Limit(pageSize).Order("title ASC").Find(&songs).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to search songs", err)
	}

	return songs, total, nil
}

func (r *songRepository) Update(ctx context.Context, song *models.Song) error {
	if err := r.db.WithContext(ctx).Save(song).Error; err != nil {
		return sharederrors.NewInternal("Failed to update song", err)
	}
	return nil
}

func (r *songRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Song{}, id).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete song", err)
	}
	return nil
}
