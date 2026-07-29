package service

import (
	"context"
	"time"

	clubrepo "clap/internal/modules/club/repository"
	"clap/internal/modules/match/dto"
	"clap/internal/modules/match/models"
	"clap/internal/modules/match/repository"
	statsrepo "clap/internal/modules/stats/repository"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

type MatchService interface {
	Create(ctx context.Context, req *dto.CreateMatchRequest, authCtx *utils.AuthorizationContext) (*dto.MatchResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.MatchResponse, error)
	List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.MatchListResponse, error)
	ListBySeason(ctx context.Context, seasonID uuid.UUID, page, pageSize int) (*dto.MatchListResponse, error)
	ListByLeague(ctx context.Context, leagueID uuid.UUID, page, pageSize int) (*dto.MatchListResponse, error)
	ListByClub(ctx context.Context, clubID uuid.UUID, page, pageSize int) (*dto.MatchListResponse, error)
	ListUpcoming(ctx context.Context, page, pageSize int) (*dto.MatchListResponse, error)
	ListLive(ctx context.Context) ([]dto.MatchResponse, error)
	Update(ctx context.Context, id uuid.UUID, req *dto.UpdateMatchRequest, authCtx *utils.AuthorizationContext) (*dto.MatchResponse, error)
	Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error
}

type matchService struct {
	matchRepo repository.MatchRepository
	detail    *detailDeps         // optional; nil disables mobile detail enrichment
	publisher MatchEventPublisher // optional; nil disables realtime delivery
}

func NewMatchService(matchRepo repository.MatchRepository) MatchService {
	return &matchService{matchRepo: matchRepo}
}

// NewMatchServiceWithDetail constructs the service with the mobile detail
// enrichment (contract §9.1) and optional realtime publishing of score updates.
func NewMatchServiceWithDetail(
	matchRepo repository.MatchRepository,
	clubRepo clubrepo.ClubRepository,
	statsRepo statsrepo.MatchStatsRepository,
	playerRepo statsrepo.PlayerRepository,
	publisher MatchEventPublisher,
) MatchService {
	return &matchService{
		matchRepo: matchRepo,
		detail: &detailDeps{
			clubRepo:   clubRepo,
			statsRepo:  statsRepo,
			playerRepo: playerRepo,
		},
		publisher: publisher,
	}
}

func (s *matchService) Create(ctx context.Context, req *dto.CreateMatchRequest, authCtx *utils.AuthorizationContext) (*dto.MatchResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	// Validate home and away clubs are different
	if req.HomeClubID == req.AwayClubID {
		return nil, sharederrors.NewBadRequest("Home and away clubs cannot be the same", nil)
	}

	matchDateTime, err := time.Parse(time.RFC3339, req.MatchDateTime)
	if err != nil {
		return nil, sharederrors.NewBadRequest("Invalid match_datetime format", err)
	}

	match := &models.Match{
		LeagueID:        req.LeagueID,
		SeasonID:        req.SeasonID,
		HomeClubID:      req.HomeClubID,
		AwayClubID:      req.AwayClubID,
		Provider:        req.Provider,
		ProviderMatchID: req.ProviderMatchID,
		MatchDateTime:   matchDateTime,
		StadiumName:     req.StadiumName,
		Status:          req.Status,
		CreatedBy:       &authCtx.UserID,
		UpdatedBy:       &authCtx.UserID,
	}

	if err := s.matchRepo.Create(ctx, match); err != nil {
		return nil, err
	}

	return s.toResponse(match), nil
}

func (s *matchService) GetByID(ctx context.Context, id uuid.UUID) (*dto.MatchResponse, error) {
	match, err := s.matchRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := s.toResponse(match)
	s.enrichDetail(ctx, match, resp)
	return resp, nil
}

func (s *matchService) List(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) (*dto.MatchListResponse, error) {
	matches, total, err := s.matchRepo.FindAll(ctx, page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.MatchResponse, len(matches))
	for i, match := range matches {
		responses[i] = *s.toResponse(&match)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.MatchListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *matchService) ListBySeason(ctx context.Context, seasonID uuid.UUID, page, pageSize int) (*dto.MatchListResponse, error) {
	matches, total, err := s.matchRepo.FindBySeason(ctx, seasonID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.MatchResponse, len(matches))
	for i, match := range matches {
		responses[i] = *s.toResponse(&match)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.MatchListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *matchService) ListByLeague(ctx context.Context, leagueID uuid.UUID, page, pageSize int) (*dto.MatchListResponse, error) {
	matches, total, err := s.matchRepo.FindByLeague(ctx, leagueID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.MatchResponse, len(matches))
	for i, match := range matches {
		responses[i] = *s.toResponse(&match)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.MatchListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *matchService) ListByClub(ctx context.Context, clubID uuid.UUID, page, pageSize int) (*dto.MatchListResponse, error) {
	matches, total, err := s.matchRepo.FindByClub(ctx, clubID, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.MatchResponse, len(matches))
	for i, match := range matches {
		responses[i] = *s.toResponse(&match)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.MatchListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *matchService) ListUpcoming(ctx context.Context, page, pageSize int) (*dto.MatchListResponse, error) {
	matches, total, err := s.matchRepo.FindUpcoming(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.MatchResponse, len(matches))
	for i, match := range matches {
		responses[i] = *s.toResponse(&match)
	}

	pagination := utils.CalculatePagination(total, page, pageSize)

	return &dto.MatchListResponse{
		Data:       responses,
		Pagination: pagination,
	}, nil
}

func (s *matchService) ListLive(ctx context.Context) ([]dto.MatchResponse, error) {
	matches, err := s.matchRepo.FindLive(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.MatchResponse, len(matches))
	for i, match := range matches {
		responses[i] = *s.toResponse(&match)
	}

	return responses, nil
}

func (s *matchService) Update(ctx context.Context, id uuid.UUID, req *dto.UpdateMatchRequest, authCtx *utils.AuthorizationContext) (*dto.MatchResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, err
	}

	match, err := s.matchRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	matchDateTime, err := time.Parse(time.RFC3339, req.MatchDateTime)
	if err != nil {
		return nil, sharederrors.NewBadRequest("Invalid match_datetime format", err)
	}

	scoreChanged := false
	statusChanged := match.Status != req.Status

	match.Provider = req.Provider
	match.ProviderMatchID = req.ProviderMatchID
	match.MatchDateTime = matchDateTime
	match.StadiumName = req.StadiumName
	match.Status = req.Status
	if req.HomeScore != nil {
		scoreChanged = scoreChanged || match.HomeScore == nil || *match.HomeScore != *req.HomeScore
		match.HomeScore = req.HomeScore
	}
	if req.AwayScore != nil {
		scoreChanged = scoreChanged || match.AwayScore == nil || *match.AwayScore != *req.AwayScore
		match.AwayScore = req.AwayScore
	}
	if req.CurrentMinute != nil {
		match.CurrentMinute = *req.CurrentMinute
	}
	match.UpdatedBy = &authCtx.UserID

	if err := s.matchRepo.Update(ctx, match); err != nil {
		return nil, err
	}

	if s.publisher != nil && (scoreChanged || statusChanged) {
		payload := map[string]any{
			"match_id":   match.ID.String(),
			"status":     match.Status,
			"home_score": match.HomeScore,
			"away_score": match.AwayScore,
			"minute":     match.CurrentMinute,
		}
		if pubErr := s.publisher.PublishMatchEvent(ctx, match.ID, "match.score.updated", payload); pubErr != nil {
			// Realtime delivery failure must not abort the mutation, but it
			// must be observable.
			logger.Error().
				Str("match_id", match.ID.String()).
				Str("event_type", "match.score.updated").
				Err(pubErr).
				Msg("realtime publish failed")
		}
	}

	logger.Info().
		Str("match_id", match.ID.String()).
		Str("user_id", authCtx.UserID.String()).
		Str("status", match.Status).
		Msg("match_updated")

	return s.toResponse(match), nil
}

func (s *matchService) Delete(ctx context.Context, id uuid.UUID, authCtx *utils.AuthorizationContext) error {
	if err := authCtx.RequireAdmin(); err != nil {
		return err
	}

	return s.matchRepo.Delete(ctx, id)
}

func (s *matchService) toResponse(match *models.Match) *dto.MatchResponse {
	return &dto.MatchResponse{
		ID:              match.ID,
		LeagueID:        match.LeagueID,
		SeasonID:        match.SeasonID,
		HomeClubID:      match.HomeClubID,
		AwayClubID:      match.AwayClubID,
		Provider:        match.Provider,
		ProviderMatchID: match.ProviderMatchID,
		MatchDateTime:   match.MatchDateTime.Format(time.RFC3339),
		StadiumName:     match.StadiumName,
		Status:          match.Status,
		CreatedAt:       match.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       match.UpdatedAt.Format(time.RFC3339),
	}
}
