package repository

import (
	"context"

	"clap/internal/modules/songlyric/models"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SongLyricRepository interface {
	Create(ctx context.Context, lyric *models.SongLyric) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.SongLyric, error)
	FindBySongID(ctx context.Context, songID uuid.UUID, language string) (*models.SongLyric, error)
	FindAllBySongID(ctx context.Context, songID uuid.UUID, page, pageSize int) ([]models.SongLyric, int64, error)
	Update(ctx context.Context, lyric *models.SongLyric) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBySongID(ctx context.Context, songID uuid.UUID) error
	ReplaceLyrics(ctx context.Context, songID uuid.UUID, lyric *models.SongLyric) error
}

type songLyricRepository struct {
	db *gorm.DB
}

func NewSongLyricRepository(db *gorm.DB) SongLyricRepository {
	return &songLyricRepository{db: db}
}

func (r *songLyricRepository) Create(ctx context.Context, lyric *models.SongLyric) error {
	if err := r.db.WithContext(ctx).Create(lyric).Error; err != nil {
		return sharederrors.NewInternal("Failed to create song lyric", err)
	}
	return nil
}

func (r *songLyricRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.SongLyric, error) {
	var lyric models.SongLyric
	if err := r.db.WithContext(ctx).Preload("Song").Where("id = ?", id).First(&lyric).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Song lyric not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find song lyric", err)
	}
	return &lyric, nil
}

func (r *songLyricRepository) FindBySongID(ctx context.Context, songID uuid.UUID, language string) (*models.SongLyric, error) {
	var lyric models.SongLyric
	if err := r.db.WithContext(ctx).Preload("Song").Where("song_id = ? AND language = ?", songID, language).First(&lyric).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, sharederrors.NewInternal("Failed to find song lyric", err)
	}
	return &lyric, nil
}

func (r *songLyricRepository) FindAllBySongID(ctx context.Context, songID uuid.UUID, page, pageSize int) ([]models.SongLyric, int64, error) {
	var lyrics []models.SongLyric
	var total int64

	query := r.db.WithContext(ctx).Model(&models.SongLyric{}).Where("song_id = ?", songID).Preload("Song")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count song lyrics", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Order("language ASC").Find(&lyrics).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find song lyrics", err)
	}

	return lyrics, total, nil
}

func (r *songLyricRepository) Update(ctx context.Context, lyric *models.SongLyric) error {
	if err := r.db.WithContext(ctx).Save(lyric).Error; err != nil {
		return sharederrors.NewInternal("Failed to update song lyric", err)
	}
	return nil
}

func (r *songLyricRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.SongLyric{}, id).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete song lyric", err)
	}
	return nil
}

func (r *songLyricRepository) DeleteBySongID(ctx context.Context, songID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("song_id = ?", songID).Delete(&models.SongLyric{}).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete song lyrics by song ID", err)
	}
	return nil
}

func (r *songLyricRepository) ReplaceLyrics(ctx context.Context, songID uuid.UUID, lyric *models.SongLyric) error {
	// Use database transaction to ensure atomicity
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing lyrics for this song
		if err := tx.Where("song_id = ?", songID).Delete(&models.SongLyric{}).Error; err != nil {
			return sharederrors.NewInternal("Failed to delete existing lyrics", err)
		}

		// Create new lyric record
		if err := tx.Create(lyric).Error; err != nil {
			return sharederrors.NewInternal("Failed to create new lyrics", err)
		}

		return nil
	})
}
