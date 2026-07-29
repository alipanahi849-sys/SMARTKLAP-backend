package service

import (
	"context"
	"time"

	"clap/internal/modules/song/dto"
	"clap/internal/modules/song/models"
	"clap/internal/modules/song/repository"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type SongService interface {
	Create(ctx context.Context, req *dto.CreateSongRequest, authCtx *utils.AuthorizationContext) (*dto.SongResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.SongResponse, error)
	List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.SongListResponse, error)
	Search(ctx context.Context, query string, page, pageSize int) (*dto.SongListResponse, error)
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateSongRequest, authCtx *utils.AuthorizationContext) (*dto.SongResponse, error)
	Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error
}

type songService struct {
	songRepo repository.SongRepository
}

func NewSongService(songRepo repository.SongRepository) SongService {
	return &songService{songRepo: songRepo}
}

func (s *songService) Create(ctx context.Context, req *dto.CreateSongRequest, authCtx *utils.AuthorizationContext) (*dto.SongResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	song := &models.Song{
		Title:     req.Title,
		Artist:    req.Artist,
		Album:     req.Album,
		Duration:  req.Duration,
		AudioURL:  req.AudioURL,
		IsActive:  req.IsActive,
		CreatedBy: &authCtx.UserID,
		UpdatedBy: &authCtx.UserID,
	}

	if err := s.songRepo.Create(ctx, song); err != nil {
		return nil, err
	}

	return s.toResponse(song), nil
}

func (s *songService) GetByID(ctx context.Context, id uuid.UUID) (*dto.SongResponse, error) {
	song, err := s.songRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(song), nil
}

func (s *songService) List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.SongListResponse, error) {
	songs, total, err := s.songRepo.FindAll(ctx, page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.SongResponse, len(songs))
	for i, song := range songs {
		responses[i] = *s.toResponse(&song)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.SongListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *songService) Search(ctx context.Context, query string, page, pageSize int) (*dto.SongListResponse, error) {
	songs, total, err := s.songRepo.Search(ctx, query, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.SongResponse, len(songs))
	for i, song := range songs {
		responses[i] = *s.toResponse(&song)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.SongListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *songService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateSongRequest, authCtx *utils.AuthorizationContext) (*dto.SongResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	song, err := s.songRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	song.Title = req.Title
	song.Artist = req.Artist
	song.Album = req.Album
	song.Duration = req.Duration
	song.AudioURL = req.AudioURL
	if req.IsActive != nil {
		song.IsActive = *req.IsActive
	}
	song.UpdatedBy = &authCtx.UserID

	if err := s.songRepo.Update(ctx, song); err != nil {
		return nil, err
	}

	return s.toResponse(song), nil
}

func (s *songService) Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error {
	if err := authCtx.RequireAdmin(); err != nil {
		return err
	}

	return s.songRepo.Delete(ctx, id)
}

func (s *songService) toResponse(song *models.Song) *dto.SongResponse {
	return &dto.SongResponse{
		ID:          song.ID,
		Title:       song.Title,
		Artist:      song.Artist,
		Album:       song.Album,
		Duration:    song.Duration,
		AudioURL:    song.AudioURL,
		IsActive:    song.IsActive,
		CreatedAt:   song.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   song.UpdatedAt.Format(time.RFC3339),
		MediaFileID: song.MediaFileID,
		StorageKey:  song.StorageKey,
		MimeType:    song.MimeType,
		FileSize:    song.FileSize,
		DurationMs:  song.DurationMs,
		Bitrate:     song.Bitrate,
		SampleRate:  song.SampleRate,
	}
}
