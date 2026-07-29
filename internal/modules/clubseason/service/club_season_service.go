package service

import (
	"context"
	"time"

	"clap/internal/modules/clubseason/dto"
	"clap/internal/modules/clubseason/models"
	"clap/internal/modules/clubseason/repository"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type ClubSeasonService interface {
	AddClubToSeason(ctx context.Context, req *dto.CreateClubSeasonRequest, authCtx *utils.AuthorizationContext) (*dto.ClubSeasonResponse, error)
	RemoveClubFromSeason(ctx context.Context, clubID, seasonID uuid.UUID, authCtx *utils.AuthorizationContext) error
	ListClubsInSeason(ctx context.Context, seasonID uuid.UUID, page, pageSize int) (*dto.ClubSeasonListResponse, error)
	ListSeasonsForClub(ctx context.Context, clubID uuid.UUID, page, pageSize int) (*dto.ClubSeasonListResponse, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, req *dto.UpdateClubSeasonRequest, authCtx *utils.AuthorizationContext) (*dto.ClubSeasonResponse, error)
}

type clubSeasonService struct {
	clubSeasonRepo repository.ClubSeasonRepository
}

func NewClubSeasonService(clubSeasonRepo repository.ClubSeasonRepository) ClubSeasonService {
	return &clubSeasonService{clubSeasonRepo: clubSeasonRepo}
}

func (s *clubSeasonService) AddClubToSeason(ctx context.Context, req *dto.CreateClubSeasonRequest, authCtx *utils.AuthorizationContext) (*dto.ClubSeasonResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	// Check if club season already exists
	existing, err := s.clubSeasonRepo.FindByClubAndSeason(ctx, req.ClubID, req.SeasonID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, sharederrors.NewConflict("Club already added to this season", nil)
	}

	joinedAt, err := time.Parse(time.RFC3339, req.JoinedAt)
	if err != nil {
		return nil, sharederrors.NewBadRequest("Invalid joined_at date format", err)
	}

	clubSeason := &models.ClubSeason{
		ClubID:    req.ClubID,
		SeasonID:  req.SeasonID,
		JoinedAt:  joinedAt,
		Status:    req.Status,
		CreatedBy: &authCtx.UserID,
		UpdatedBy: &authCtx.UserID,
	}

	if err := s.clubSeasonRepo.Create(ctx, clubSeason); err != nil {
		return nil, err
	}

	return s.toResponse(clubSeason), nil
}

func (s *clubSeasonService) RemoveClubFromSeason(ctx context.Context, clubID, seasonID uuid.UUID, authCtx *utils.AuthorizationContext) error {
	if err := authCtx.RequireAdmin(); err != nil {
		return err
	}

	return s.clubSeasonRepo.DeleteByClubAndSeason(ctx, clubID, seasonID)
}

func (s *clubSeasonService) ListClubsInSeason(ctx context.Context, seasonID uuid.UUID, page, pageSize int) (*dto.ClubSeasonListResponse, error) {
	clubSeasons, total, err := s.clubSeasonRepo.FindClubsInSeason(ctx, seasonID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ClubSeasonResponse, len(clubSeasons))
	for i, cs := range clubSeasons {
		responses[i] = *s.toResponse(&cs)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.ClubSeasonListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *clubSeasonService) ListSeasonsForClub(ctx context.Context, clubID uuid.UUID, page, pageSize int) (*dto.ClubSeasonListResponse, error) {
	clubSeasons, total, err := s.clubSeasonRepo.FindSeasonsForClub(ctx, clubID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.ClubSeasonResponse, len(clubSeasons))
	for i, cs := range clubSeasons {
		responses[i] = *s.toResponse(&cs)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.ClubSeasonListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *clubSeasonService) UpdateStatus(ctx context.Context, id uuid.UUID, req *dto.UpdateClubSeasonRequest, authCtx *utils.AuthorizationContext) (*dto.ClubSeasonResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	clubSeason, err := s.clubSeasonRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	clubSeason.Status = req.Status
	clubSeason.UpdatedBy = &authCtx.UserID

	if err := s.clubSeasonRepo.Update(ctx, clubSeason); err != nil {
		return nil, err
	}

	return s.toResponse(clubSeason), nil
}

func (s *clubSeasonService) toResponse(clubSeason *models.ClubSeason) *dto.ClubSeasonResponse {
	return &dto.ClubSeasonResponse{
		ID:        clubSeason.ID,
		ClubID:    clubSeason.ClubID,
		SeasonID:  clubSeason.SeasonID,
		JoinedAt:  clubSeason.JoinedAt.Format(time.RFC3339),
		Status:    clubSeason.Status,
		CreatedAt: clubSeason.CreatedAt.Format(time.RFC3339),
		UpdatedAt: clubSeason.UpdatedAt.Format(time.RFC3339),
	}
}
