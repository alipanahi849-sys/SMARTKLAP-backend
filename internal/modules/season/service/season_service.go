package service

import (
	"context"
	"time"

	"clap/internal/modules/season/dto"
	"clap/internal/modules/season/models"
	"clap/internal/modules/season/repository"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type SeasonService interface {
	Create(ctx context.Context, req *dto.CreateSeasonRequest, authCtx *utils.AuthorizationContext) (*dto.SeasonResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.SeasonResponse, error)
	List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.SeasonListResponse, error)
	ListByLeagueID(ctx context.Context, leagueID uuid.UUID, page, pageSize int) (*dto.SeasonListResponse, error)
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateSeasonRequest, authCtx *utils.AuthorizationContext) (*dto.SeasonResponse, error)
	Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error
}

type seasonService struct {
	seasonRepo repository.SeasonRepository
}

func NewSeasonService(seasonRepo repository.SeasonRepository) SeasonService {
	return &seasonService{seasonRepo: seasonRepo}
}

func (s *seasonService) Create(ctx context.Context, req *dto.CreateSeasonRequest, authCtx *utils.AuthorizationContext) (*dto.SeasonResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		return nil, sharederrors.NewBadRequest("Invalid start date format", err)
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		return nil, sharederrors.NewBadRequest("Invalid end date format", err)
	}

	if startDate.After(endDate) {
		return nil, sharederrors.NewBadRequest("Start date must be before end date", nil)
	}

	season := &models.Season{
		LeagueID:  req.LeagueID,
		Name:      req.Name,
		StartDate: startDate,
		EndDate:   endDate,
		IsActive:  req.IsActive,
		CreatedBy: &authCtx.UserID,
		UpdatedBy: &authCtx.UserID,
	}

	if err := s.seasonRepo.Create(ctx, season); err != nil {
		return nil, err
	}

	return s.toResponse(season), nil
}

func (s *seasonService) GetByID(ctx context.Context, id uuid.UUID) (*dto.SeasonResponse, error) {
	season, err := s.seasonRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(season), nil
}

func (s *seasonService) List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.SeasonListResponse, error) {
	seasons, total, err := s.seasonRepo.FindAll(ctx, page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.SeasonResponse, len(seasons))
	for i, season := range seasons {
		responses[i] = *s.toResponse(&season)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.SeasonListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *seasonService) ListByLeagueID(ctx context.Context, leagueID uuid.UUID, page, pageSize int) (*dto.SeasonListResponse, error) {
	seasons, total, err := s.seasonRepo.FindByLeagueID(ctx, leagueID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.SeasonResponse, len(seasons))
	for i, season := range seasons {
		responses[i] = *s.toResponse(&season)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.SeasonListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *seasonService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateSeasonRequest, authCtx *utils.AuthorizationContext) (*dto.SeasonResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	season, err := s.seasonRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	startDate, err := time.Parse(time.RFC3339, req.StartDate)
	if err != nil {
		return nil, sharederrors.NewBadRequest("Invalid start date format", err)
	}

	endDate, err := time.Parse(time.RFC3339, req.EndDate)
	if err != nil {
		return nil, sharederrors.NewBadRequest("Invalid end date format", err)
	}

	if startDate.After(endDate) {
		return nil, sharederrors.NewBadRequest("Start date must be before end date", nil)
	}

	season.Name = req.Name
	season.StartDate = startDate
	season.EndDate = endDate
	if req.IsActive != nil {
		season.IsActive = *req.IsActive
	}
	season.UpdatedBy = &authCtx.UserID

	if err := s.seasonRepo.Update(ctx, season); err != nil {
		return nil, err
	}

	return s.toResponse(season), nil
}

func (s *seasonService) Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error {
	if err := authCtx.RequireAdmin(); err != nil {
		return err
	}

	return s.seasonRepo.Delete(ctx, id)
}

func (s *seasonService) toResponse(season *models.Season) *dto.SeasonResponse {
	return &dto.SeasonResponse{
		ID:        season.ID,
		LeagueID:  season.LeagueID,
		Name:      season.Name,
		StartDate: season.StartDate.Format(time.RFC3339),
		EndDate:   season.EndDate.Format(time.RFC3339),
		IsActive:  season.IsActive,
		CreatedAt: season.CreatedAt.Format(time.RFC3339),
		UpdatedAt: season.UpdatedAt.Format(time.RFC3339),
	}
}
