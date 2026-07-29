package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"clap/internal/modules/songlyric/dto"
	"clap/internal/modules/songlyric/models"
	"clap/internal/modules/songlyric/repository"
	"clap/internal/shared/config"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"
	"clap/pkg/lyrics"

	"github.com/google/uuid"
)

type SongLyricService interface {
	Create(ctx context.Context, req *dto.CreateSongLyricRequest, authCtx *utils.AuthorizationContext) (*dto.SongLyricResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.SongLyricResponse, error)
	GetBySongID(ctx context.Context, songID uuid.UUID, language string) (*dto.SongLyricResponse, error)
	ListBySongID(ctx context.Context, songID uuid.UUID, page, pageSize int) (*dto.SongLyricListResponse, error)
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateSongLyricRequest, authCtx *utils.AuthorizationContext) (*dto.SongLyricResponse, error)
	Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error
	ImportLyrics(ctx context.Context, songID uuid.UUID, req *dto.ImportLyricsRequest, authCtx *utils.AuthorizationContext) (*dto.LyricsImportResponse, error)
}

type songLyricService struct {
	lyricRepo repository.SongLyricRepository
}

func NewSongLyricService(lyricRepo repository.SongLyricRepository) SongLyricService {
	return &songLyricService{lyricRepo: lyricRepo}
}

func (s *songLyricService) Create(ctx context.Context, req *dto.CreateSongLyricRequest, authCtx *utils.AuthorizationContext) (*dto.SongLyricResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	// Check if lyric for this song and language already exists
	existing, err := s.lyricRepo.FindBySongID(ctx, req.SongID, req.Language)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, sharederrors.NewConflict("Lyrics for this song and language already exist", nil)
	}

	lyric := &models.SongLyric{
		SongID:    req.SongID,
		Language:  req.Language,
		Lyrics:    req.Lyrics,
		CreatedBy: &authCtx.UserID,
		UpdatedBy: &authCtx.UserID,
	}

	if err := s.lyricRepo.Create(ctx, lyric); err != nil {
		return nil, err
	}

	return s.toResponse(lyric), nil
}

func (s *songLyricService) GetByID(ctx context.Context, id uuid.UUID) (*dto.SongLyricResponse, error) {
	lyric, err := s.lyricRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(lyric), nil
}

func (s *songLyricService) GetBySongID(ctx context.Context, songID uuid.UUID, language string) (*dto.SongLyricResponse, error) {
	lyric, err := s.lyricRepo.FindBySongID(ctx, songID, language)
	if err != nil {
		return nil, err
	}
	if lyric == nil {
		return nil, sharederrors.NewNotFound("Lyrics not found for this song and language", nil)
	}

	return s.toResponse(lyric), nil
}

func (s *songLyricService) ListBySongID(ctx context.Context, songID uuid.UUID, page, pageSize int) (*dto.SongLyricListResponse, error) {
	lyrics, total, err := s.lyricRepo.FindAllBySongID(ctx, songID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.SongLyricResponse, len(lyrics))
	for i, lyric := range lyrics {
		responses[i] = *s.toResponse(&lyric)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.SongLyricListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *songLyricService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateSongLyricRequest, authCtx *utils.AuthorizationContext) (*dto.SongLyricResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	lyric, err := s.lyricRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	lyric.Language = req.Language
	lyric.Lyrics = req.Lyrics
	lyric.UpdatedBy = &authCtx.UserID

	if err := s.lyricRepo.Update(ctx, lyric); err != nil {
		return nil, err
	}

	return s.toResponse(lyric), nil
}

func (s *songLyricService) Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error {
	if err := authCtx.RequireAdmin(); err != nil {
		return err
	}

	return s.lyricRepo.Delete(ctx, id)
}

func (s *songLyricService) toResponse(lyric *models.SongLyric) *dto.SongLyricResponse {
	return &dto.SongLyricResponse{
		ID:        lyric.ID,
		SongID:    lyric.SongID,
		Language:  lyric.Language,
		Lyrics:    lyric.Lyrics,
		CreatedAt: lyric.CreatedAt.Format(time.RFC3339),
		UpdatedAt: lyric.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *songLyricService) ImportLyrics(ctx context.Context, songID uuid.UUID, req *dto.ImportLyricsRequest, authCtx *utils.AuthorizationContext) (*dto.LyricsImportResponse, error) {
	// Check if user is admin or club admin
	if err := authCtx.RequireAdminOrClubAdmin(); err != nil {
		return nil, sharederrors.NewForbidden("Only admins and club admins can import lyrics", err)
	}

	// Validate content
	if strings.TrimSpace(req.Content) == "" {
		return nil, sharederrors.NewBadRequest("Lyrics content cannot be empty", nil)
	}

	// Validate file size
	contentSize := len(req.Content)
	maxSizeKB := config.AppConfig.Storage.MaxLyricFileSizeKB
	if contentSize > maxSizeKB*1024 {
		return nil, sharederrors.NewBadRequest(fmt.Sprintf("Lyrics content exceeds maximum size of %d KB", maxSizeKB), nil)
	}

	// Detect format
	format := lyrics.DetectFormat(req.Content)

	// Parse lyrics
	var lyricLines []lyrics.LyricLine
	var err error

	if format == "lrc" {
		lyricLines, err = lyrics.ParseLRC(req.Content)
		if err != nil {
			return nil, sharederrors.NewBadRequest("Invalid LRC format", err)
		}
	} else {
		lyricLines = lyrics.ParsePlainText(req.Content)
	}

	// Validate number of lines
	maxLines := config.AppConfig.Storage.MaxLyricLines
	if len(lyricLines) > maxLines {
		return nil, sharederrors.NewBadRequest(fmt.Sprintf("Lyrics exceed maximum number of lines (%d)", maxLines), nil)
	}

	// Create lyric record
	// For now, we'll store all lyrics as a single text block
	// In the future, we might want to store individual lines with timestamps
	lyricText := req.Content

	lyric := &models.SongLyric{
		SongID:    songID,
		Language:  "en", // Default to English for now
		Lyrics:    lyricText,
		CreatedBy: &authCtx.UserID,
		UpdatedBy: &authCtx.UserID,
	}

	// If replace_existing is true, use transactional replace
	if req.ReplaceExisting {
		if err := s.lyricRepo.ReplaceLyrics(ctx, songID, lyric); err != nil {
			return nil, err
		}
	} else {
		// Otherwise, just create new lyric
		if err := s.lyricRepo.Create(ctx, lyric); err != nil {
			return nil, err
		}
	}

	return &dto.LyricsImportResponse{
		Message: "Lyrics imported successfully",
		Count:   len(lyricLines),
	}, nil
}
