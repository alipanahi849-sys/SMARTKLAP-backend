package service

import (
	"context"
	"time"

	"clap/internal/modules/matchsongschedule/dto"
	"clap/internal/modules/matchsongschedule/models"
	"clap/internal/modules/matchsongschedule/repository"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type MatchSongScheduleService interface {
	Create(ctx context.Context, req *dto.CreateMatchSongScheduleRequest, authCtx *utils.AuthorizationContext) (*dto.MatchSongScheduleResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.MatchSongScheduleResponse, error)
	List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.MatchSongScheduleListResponse, error)
	ListByMatchID(ctx context.Context, matchID uuid.UUID, page, pageSize int) (*dto.MatchSongScheduleListResponse, error)
	ListBySongID(ctx context.Context, songID uuid.UUID, page, pageSize int) (*dto.MatchSongScheduleListResponse, error)
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateMatchSongScheduleRequest, authCtx *utils.AuthorizationContext) (*dto.MatchSongScheduleResponse, error)
	Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error
}

type matchSongScheduleService struct {
	scheduleRepo repository.MatchSongScheduleRepository
}

func NewMatchSongScheduleService(scheduleRepo repository.MatchSongScheduleRepository) MatchSongScheduleService {
	return &matchSongScheduleService{scheduleRepo: scheduleRepo}
}

func (s *matchSongScheduleService) Create(ctx context.Context, req *dto.CreateMatchSongScheduleRequest, authCtx *utils.AuthorizationContext) (*dto.MatchSongScheduleResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	scheduledTime, err := time.Parse(time.RFC3339, req.ScheduledTime)
	if err != nil {
		return nil, sharederrors.NewBadRequest("Invalid scheduled_time format", err)
	}

	schedule := &models.MatchSongSchedule{
		MatchID:       req.MatchID,
		SongID:        req.SongID,
		ScheduledTime: scheduledTime,
		EventType:     req.EventType,
		IsActive:      req.IsActive,
		CreatedBy:     &authCtx.UserID,
		UpdatedBy:     &authCtx.UserID,
	}

	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return nil, err
	}

	return s.toResponse(schedule), nil
}

func (s *matchSongScheduleService) GetByID(ctx context.Context, id uuid.UUID) (*dto.MatchSongScheduleResponse, error) {
	schedule, err := s.scheduleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.toResponse(schedule), nil
}

func (s *matchSongScheduleService) List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.MatchSongScheduleListResponse, error) {
	schedules, total, err := s.scheduleRepo.FindAll(ctx, page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.MatchSongScheduleResponse, len(schedules))
	for i, schedule := range schedules {
		responses[i] = *s.toResponse(&schedule)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.MatchSongScheduleListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *matchSongScheduleService) ListByMatchID(ctx context.Context, matchID uuid.UUID, page, pageSize int) (*dto.MatchSongScheduleListResponse, error) {
	schedules, total, err := s.scheduleRepo.FindByMatchID(ctx, matchID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.MatchSongScheduleResponse, len(schedules))
	for i, schedule := range schedules {
		responses[i] = *s.toResponse(&schedule)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.MatchSongScheduleListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *matchSongScheduleService) ListBySongID(ctx context.Context, songID uuid.UUID, page, pageSize int) (*dto.MatchSongScheduleListResponse, error) {
	schedules, total, err := s.scheduleRepo.FindBySongID(ctx, songID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.MatchSongScheduleResponse, len(schedules))
	for i, schedule := range schedules {
		responses[i] = *s.toResponse(&schedule)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.MatchSongScheduleListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *matchSongScheduleService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateMatchSongScheduleRequest, authCtx *utils.AuthorizationContext) (*dto.MatchSongScheduleResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	schedule, err := s.scheduleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	scheduledTime, err := time.Parse(time.RFC3339, req.ScheduledTime)
	if err != nil {
		return nil, sharederrors.NewBadRequest("Invalid scheduled_time format", err)
	}

	schedule.ScheduledTime = scheduledTime
	schedule.EventType = req.EventType
	if req.IsActive != nil {
		schedule.IsActive = *req.IsActive
	}
	schedule.UpdatedBy = &authCtx.UserID

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, err
	}

	return s.toResponse(schedule), nil
}

func (s *matchSongScheduleService) Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error {
	if err := authCtx.RequireAdmin(); err != nil {
		return err
	}

	return s.scheduleRepo.Delete(ctx, id)
}

func (s *matchSongScheduleService) toResponse(schedule *models.MatchSongSchedule) *dto.MatchSongScheduleResponse {
	return &dto.MatchSongScheduleResponse{
		ID:            schedule.ID,
		MatchID:       schedule.MatchID,
		SongID:        schedule.SongID,
		ScheduledTime: schedule.ScheduledTime.Format(time.RFC3339),
		EventType:     schedule.EventType,
		IsActive:      schedule.IsActive,
		CreatedAt:     schedule.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     schedule.UpdatedAt.Format(time.RFC3339),
	}
}
