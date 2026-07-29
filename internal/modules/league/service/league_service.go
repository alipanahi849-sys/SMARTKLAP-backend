package service

import (
	"context"
	"time"

	"clap/internal/modules/league/dto"
	"clap/internal/modules/league/models"
	"clap/internal/modules/league/repository"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type LeagueService interface {
	Create(ctx context.Context, req *dto.CreateLeagueRequest, authCtx *utils.AuthorizationContext) (*dto.LeagueResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.LeagueResponse, error)
	List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.LeagueListResponse, error)
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateLeagueRequest, authCtx *utils.AuthorizationContext) (*dto.LeagueResponse, error)
	Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error
}

type leagueService struct {
	leagueRepo repository.LeagueRepository
}

func NewLeagueService(leagueRepo repository.LeagueRepository) LeagueService {
	return &leagueService{leagueRepo: leagueRepo}
}

func (s *leagueService) Create(ctx context.Context, req *dto.CreateLeagueRequest, authCtx *utils.AuthorizationContext) (*dto.LeagueResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	// Check for unique constraint on provider + provider_league_id
	if req.Provider != "" && req.ProviderLeagueID != "" {
		existing, err := s.leagueRepo.FindByProviderLeagueID(ctx, req.Provider, req.ProviderLeagueID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, sharederrors.NewConflict("League with this provider ID already exists", nil)
		}
	}

	league := &models.League{
		Name:             req.Name,
		Country:          req.Country,
		Provider:         req.Provider,
		ProviderLeagueID: req.ProviderLeagueID,
		IsActive:         req.IsActive,
		CreatedBy:        &authCtx.UserID,
		UpdatedBy:        &authCtx.UserID,
	}

	if err := s.leagueRepo.Create(ctx, league); err != nil {
		return nil, err
	}

	return s.toResponse(league), nil
}

func (s *leagueService) GetByID(ctx context.Context, id uuid.UUID) (*dto.LeagueResponse, error) {
	league, err := s.leagueRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(league), nil
}

func (s *leagueService) List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.LeagueListResponse, error) {
	leagues, total, err := s.leagueRepo.FindAll(ctx, page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.LeagueResponse, len(leagues))
	for i, league := range leagues {
		responses[i] = *s.toResponse(&league)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.LeagueListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *leagueService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateLeagueRequest, authCtx *utils.AuthorizationContext) (*dto.LeagueResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	league, err := s.leagueRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check for unique constraint on provider + provider_league_id if changed
	if req.Provider != "" && req.ProviderLeagueID != "" {
		if req.Provider != league.Provider || req.ProviderLeagueID != league.ProviderLeagueID {
			existing, err := s.leagueRepo.FindByProviderLeagueID(ctx, req.Provider, req.ProviderLeagueID)
			if err != nil {
				return nil, err
			}
			if existing != nil && existing.ID != id {
				return nil, sharederrors.NewConflict("League with this provider ID already exists", nil)
			}
		}
	}

	league.Name = req.Name
	league.Country = req.Country
	league.Provider = req.Provider
	league.ProviderLeagueID = req.ProviderLeagueID
	if req.IsActive != nil {
		league.IsActive = *req.IsActive
	}
	league.UpdatedBy = &authCtx.UserID

	if err := s.leagueRepo.Update(ctx, league); err != nil {
		return nil, err
	}

	return s.toResponse(league), nil
}

func (s *leagueService) Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error {
	if err := authCtx.RequireAdmin(); err != nil {
		return err
	}

	return s.leagueRepo.Delete(ctx, id)
}

func (s *leagueService) toResponse(league *models.League) *dto.LeagueResponse {
	return &dto.LeagueResponse{
		ID:               league.ID,
		Name:             league.Name,
		Country:          league.Country,
		Provider:         league.Provider,
		ProviderLeagueID: league.ProviderLeagueID,
		IsActive:         league.IsActive,
		CreatedAt:        league.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        league.UpdatedAt.Format(time.RFC3339),
	}
}
