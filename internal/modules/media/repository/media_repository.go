package repository

import (
	"context"

	"clap/internal/modules/media/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MediaRepository interface {
	Create(ctx context.Context, mediaFile *models.MediaFile) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.MediaFile, error)
	FindByChecksum(ctx context.Context, checksum string) (*models.MediaFile, error)
	FindByStorageKey(ctx context.Context, storageKey string) (*models.MediaFile, error)
	Update(ctx context.Context, mediaFile *models.MediaFile) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type mediaRepository struct {
	db *gorm.DB
}

func NewMediaRepository(db *gorm.DB) MediaRepository {
	return &mediaRepository{db: db}
}

func (r *mediaRepository) Create(ctx context.Context, mediaFile *models.MediaFile) error {
	if err := r.db.WithContext(ctx).Create(mediaFile).Error; err != nil {
		return sharederrors.NewInternal("Failed to create media file", err)
	}
	return nil
}

func (r *mediaRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.MediaFile, error) {
	var mediaFile models.MediaFile
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&mediaFile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Media file not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find media file", err)
	}
	return &mediaFile, nil
}

func (r *mediaRepository) FindByChecksum(ctx context.Context, checksum string) (*models.MediaFile, error) {
	var mediaFile models.MediaFile
	if err := r.db.WithContext(ctx).Where("checksum = ?", checksum).First(&mediaFile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, sharederrors.NewInternal("Failed to find media file by checksum", err)
	}
	return &mediaFile, nil
}

func (r *mediaRepository) FindByStorageKey(ctx context.Context, storageKey string) (*models.MediaFile, error) {
	var mediaFile models.MediaFile
	if err := r.db.WithContext(ctx).Where("storage_key = ?", storageKey).First(&mediaFile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Media file not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find media file by storage key", err)
	}
	return &mediaFile, nil
}

func (r *mediaRepository) Update(ctx context.Context, mediaFile *models.MediaFile) error {
	if err := r.db.WithContext(ctx).Save(mediaFile).Error; err != nil {
		return sharederrors.NewInternal("Failed to update media file", err)
	}
	return nil
}

func (r *mediaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.MediaFile{}, id).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete media file", err)
	}
	return nil
}
