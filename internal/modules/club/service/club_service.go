package service

import (
	"context"
	"time"

	"clap/internal/modules/club/dto"
	"clap/internal/modules/club/models"
	"clap/internal/modules/club/repository"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type ClubService interface {
	Create(ctx context.Context, req *dto.CreateClubRequest, authCtx *utils.AuthorizationContext) (*dto.ClubResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.ClubResponse, error)
	List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.ClubListResponse, error)
	Search(ctx context.Context, query string, page, pageSize int) (*dto.ClubListResponse, error)
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateClubRequest, authCtx *utils.AuthorizationContext) (*dto.ClubResponse, error)
	Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error
}

type clubService struct {
	clubRepo repository.ClubRepository
}

func NewClubService(clubRepo repository.ClubRepository) ClubService {
	return &clubService{clubRepo: clubRepo}
}

func (s *clubService) Create(ctx context.Context, req *dto.CreateClubRequest, authCtx *utils.AuthorizationContext) (*dto.ClubResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	club := &models.Club{
		Name:        req.Name,
		ShortName:   req.ShortName,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		Country:     req.Country,
		IsActive:    req.IsActive,
		CreatedBy:   &authCtx.UserID,
		UpdatedBy:   &authCtx.UserID,
	}

	if err := s.clubRepo.Create(ctx, club); err != nil {
		return nil, err
	}

	return s.toResponse(club), nil
}

func (s *clubService) GetByID(ctx context.Context, id uuid.UUID) (*dto.ClubResponse, error) {
	club, err := s.clubRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(club), nil
}

func (s *clubService) List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.ClubListResponse, error) {
	clubs, total, err := s.clubRepo.FindAll(ctx, page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ClubResponse, len(clubs))
	for i, club := range clubs {
		responses[i] = *s.toResponse(&club)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.ClubListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *clubService) Search(ctx context.Context, query string, page, pageSize int) (*dto.ClubListResponse, error) {
	clubs, total, err := s.clubRepo.Search(ctx, query, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ClubResponse, len(clubs))
	for i, club := range clubs {
		responses[i] = *s.toResponse(&club)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.ClubListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *clubService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateClubRequest, authCtx *utils.AuthorizationContext) (*dto.ClubResponse, error) {
	club, err := s.clubRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check authorization - admin can update any club, club admin can only update their own club
	if !authCtx.IsAdmin {
		if authCtx.ClubID == nil || *authCtx.ClubID != id {
			return nil, sharederrors.NewForbidden("You can only update your own club", nil)
		}
	}

	club.Name = req.Name
	club.ShortName = req.ShortName
	club.Description = req.Description
	club.LogoURL = req.LogoURL
	club.Country = req.Country
	if req.IsActive != nil {
		club.IsActive = *req.IsActive
	}
	club.UpdatedBy = &authCtx.UserID

	if err := s.clubRepo.Update(ctx, club); err != nil {
		return nil, err
	}

	return s.toResponse(club), nil
}

func (s *clubService) Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error {
	if err := authCtx.RequireAdmin(); err != nil {
		return err
	}

	return s.clubRepo.Delete(ctx, id)
}

func (s *clubService) toResponse(club *models.Club) *dto.ClubResponse {
	return &dto.ClubResponse{
		ID:          club.ID,
		Name:        club.Name,
		ShortName:   club.ShortName,
		Description: club.Description,
		LogoURL:     club.LogoURL,
		Country:     club.Country,
		IsActive:    club.IsActive,
		CreatedAt:   club.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   club.UpdatedAt.Format(time.RFC3339),
	}
}
