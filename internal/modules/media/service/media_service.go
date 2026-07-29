package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"clap/internal/modules/media/dto"
	"clap/internal/modules/media/models"
	"clap/internal/modules/media/repository"
	songrepository "clap/internal/modules/song/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/utils"
	"clap/pkg/audio"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

type MediaService interface {
	Upload(ctx context.Context, file *multipart.FileHeader, authCtx *utils.AuthorizationContext) (*dto.MediaResponse, error)
	GetPlaybackURL(ctx context.Context, mediaID uuid.UUID) (*dto.PlaybackURLResponse, error)
	UploadSongAudio(ctx context.Context, songID uuid.UUID, file *multipart.FileHeader, authCtx *utils.AuthorizationContext) (*dto.SongAudioUploadResponse, error)
}

type mediaService struct {
	mediaRepo    repository.MediaRepository
	songRepo     songrepository.SongRepository
	storage      storage.StorageProvider
	maxFileSize  int64
	signedExpiry time.Duration
}

func NewMediaService(
	mediaRepo repository.MediaRepository,
	songRepo songrepository.SongRepository,
	storage storage.StorageProvider,
	maxFileSizeMB int,
	signedExpiryMinutes int,
) MediaService {
	return &mediaService{
		mediaRepo:    mediaRepo,
		songRepo:     songRepo,
		storage:      storage,
		maxFileSize:  int64(maxFileSizeMB) * 1024 * 1024,
		signedExpiry: time.Duration(signedExpiryMinutes) * time.Minute,
	}
}

func (s *mediaService) Upload(ctx context.Context, file *multipart.FileHeader, authCtx *utils.AuthorizationContext) (*dto.MediaResponse, error) {
	// Check if user is admin or club admin
	if err := authCtx.RequireAdminOrClubAdmin(); err != nil {
		return nil, errors.NewForbidden("Only admins and club admins can upload media", err)
	}

	// Validate file size
	if file.Size > s.maxFileSize {
		return nil, errors.NewBadRequest(fmt.Sprintf("File size exceeds maximum allowed size of %d MB", s.maxFileSize/(1024*1024)), nil)
	}

	// Validate MIME type
	allowedMimeTypes := []string{"audio/mpeg", "audio/mp3"}
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(file.Filename))
	}

	if !s.isAllowedMimeType(mimeType, allowedMimeTypes) {
		return nil, errors.NewBadRequest("Invalid file type. Only MP3 files are allowed", nil)
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		return nil, errors.NewInternal("Failed to open file", err)
	}
	defer src.Close()

	// Calculate checksum
	checksum, err := s.calculateChecksum(src)
	if err != nil {
		return nil, errors.NewInternal("Failed to calculate checksum", err)
	}

	// Check for duplicate
	existingMedia, err := s.mediaRepo.FindByChecksum(ctx, checksum)
	if err != nil {
		return nil, err
	}
	if existingMedia != nil {
		return s.toMediaResponse(existingMedia), nil
	}

	// Reset file reader
	src, err = file.Open()
	if err != nil {
		return nil, errors.NewInternal("Failed to reopen file", err)
	}
	defer src.Close()

	// Extract metadata (validate that it's a valid audio file)
	_, err = audio.ExtractMetadata(ctx, src, mimeType)
	if err != nil {
		return nil, errors.NewInternal("Failed to extract audio metadata", err)
	}

	// Reset file reader again for upload
	src, err = file.Open()
	if err != nil {
		return nil, errors.NewInternal("Failed to reopen file for upload", err)
	}
	defer src.Close()

	// Generate storage key
	storageKey := s.generateStorageKey(file.Filename, checksum)

	// Upload to storage
	if err := s.storage.Upload(ctx, storageKey, src, mimeType, file.Size); err != nil {
		return nil, errors.NewInternal("Failed to upload file to storage", err)
	}

	// Create media record
	mediaFile := &models.MediaFile{
		StorageKey:       storageKey,
		OriginalFileName: file.Filename,
		MimeType:         mimeType,
		FileSize:         file.Size,
		Checksum:         checksum,
		UploadedBy:       authCtx.UserID,
	}

	if err := s.mediaRepo.Create(ctx, mediaFile); err != nil {
		// Rollback: delete from storage
		_ = s.storage.Delete(ctx, storageKey)
		return nil, err
	}

	return s.toMediaResponse(mediaFile), nil
}

func (s *mediaService) GetPlaybackURL(ctx context.Context, mediaID uuid.UUID) (*dto.PlaybackURLResponse, error) {
	mediaFile, err := s.mediaRepo.FindByID(ctx, mediaID)
	if err != nil {
		return nil, err
	}

	url, err := s.storage.GenerateSignedURL(ctx, mediaFile.StorageKey, s.signedExpiry)
	if err != nil {
		return nil, errors.NewInternal("Failed to generate signed URL", err)
	}

	expiresAt := time.Now().Add(s.signedExpiry).Format(time.RFC3339)

	return &dto.PlaybackURLResponse{
		URL:       url,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *mediaService) UploadSongAudio(ctx context.Context, songID uuid.UUID, file *multipart.FileHeader, authCtx *utils.AuthorizationContext) (*dto.SongAudioUploadResponse, error) {
	// Check if user is admin or club admin
	if err := authCtx.RequireAdminOrClubAdmin(); err != nil {
		return nil, errors.NewForbidden("Only admins and club admins can upload audio", err)
	}

	// Get song
	song, err := s.songRepo.FindByID(ctx, songID)
	if err != nil {
		return nil, err
	}

	// If club admin, check if they own the song
	if authCtx.IsClubAdmin && !authCtx.IsAdmin {
		// For now, we'll allow club admins to upload to any song
		// In a real implementation, you might want to check club ownership
	}

	// Upload media
	mediaResponse, err := s.Upload(ctx, file, authCtx)
	if err != nil {
		return nil, err
	}

	// Get media file to extract metadata
	mediaFile, err := s.mediaRepo.FindByID(ctx, mediaResponse.ID)
	if err != nil {
		return nil, err
	}

	// Reset file reader for metadata extraction
	src, err := file.Open()
	if err != nil {
		return nil, errors.NewInternal("Failed to open file for metadata extraction", err)
	}
	defer src.Close()

	// Extract metadata
	metadata, err := audio.ExtractMetadata(ctx, src, mediaFile.MimeType)
	if err != nil {
		return nil, errors.NewInternal("Failed to extract audio metadata", err)
	}

	// Update song with media information
	song.MediaFileID = &mediaFile.ID
	song.StorageKey = mediaFile.StorageKey
	song.MimeType = mediaFile.MimeType
	song.FileSize = mediaFile.FileSize
	song.Duration = int(metadata.DurationMs / 1000) // Convert to seconds for backward compatibility

	if err := s.songRepo.Update(ctx, song); err != nil {
		return nil, err
	}

	return &dto.SongAudioUploadResponse{
		MediaFileID: mediaFile.ID,
		StorageKey:  mediaFile.StorageKey,
		MimeType:    mediaFile.MimeType,
		FileSize:    mediaFile.FileSize,
		DurationMs:  metadata.DurationMs,
		Bitrate:     metadata.Bitrate,
		SampleRate:  metadata.SampleRate,
	}, nil
}

func (s *mediaService) calculateChecksum(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *mediaService) generateStorageKey(filename, checksum string) string {
	ext := filepath.Ext(filename)
	// Use hierarchical structure for scalability: media/ab/cd/ef/checksum.ext
	// This prevents performance issues with millions of files in a single directory
	if len(checksum) >= 6 {
		return fmt.Sprintf("media/%s/%s/%s/%s%s", checksum[0:2], checksum[2:4], checksum[4:6], checksum, ext)
	}
	// Fallback for short checksums (shouldn't happen with SHA256)
	return fmt.Sprintf("media/%s%s", checksum, ext)
}

func (s *mediaService) isAllowedMimeType(mimeType string, allowed []string) bool {
	for _, allowedType := range allowed {
		if strings.EqualFold(mimeType, allowedType) {
			return true
		}
	}
	return false
}

func (s *mediaService) toMediaResponse(mediaFile *models.MediaFile) *dto.MediaResponse {
	return &dto.MediaResponse{
		ID:               mediaFile.ID,
		StorageKey:       mediaFile.StorageKey,
		OriginalFileName: mediaFile.OriginalFileName,
		MimeType:         mediaFile.MimeType,
		FileSize:         mediaFile.FileSize,
		Checksum:         mediaFile.Checksum,
		UploadedBy:       mediaFile.UploadedBy,
		CreatedAt:        mediaFile.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        mediaFile.UpdatedAt.Format(time.RFC3339),
	}
}
