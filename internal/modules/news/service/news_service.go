package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	clubmodels "clap/internal/modules/club/models"
	clubrepo "clap/internal/modules/club/repository"
	"clap/internal/modules/news/dto"
	settingsrepo "clap/internal/modules/settings/repository"
	"clap/internal/shared/errors"
	"clap/pkg/newsfeed"

	"github.com/google/uuid"
)

type ClubProvisioner interface {
	EnsureClubFromProvider(ctx context.Context, providerTeamID string) (*clubmodels.Club, error)
}

type NewsService interface {
	List(ctx context.Context, filters dto.NewsListFilters) (*dto.NewsListResponse, error)
	GetByID(ctx context.Context, id string) (*dto.NewsDetailResponse, error)
	GetNewsClub(ctx context.Context) (*dto.NewsClubResponse, error)
	SetNewsClub(ctx context.Context, req dto.SetNewsClubRequest) (*dto.NewsClubResponse, error)
}

type newsService struct {
	feed      newsfeed.Provider
	settings  settingsrepo.SettingsRepository
	clubs     clubrepo.ClubRepository
	clubsFrom ClubProvisioner
}

func NewNewsService(
	feed newsfeed.Provider,
	settings settingsrepo.SettingsRepository,
	clubs clubrepo.ClubRepository,
	clubsFrom ClubProvisioner,
) NewsService {
	return &newsService{
		feed:      feed,
		settings:  settings,
		clubs:     clubs,
		clubsFrom: clubsFrom,
	}
}

func (s *newsService) List(ctx context.Context, filters dto.NewsListFilters) (*dto.NewsListResponse, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}

	empty := &dto.NewsListResponse{
		Items: []dto.NewsItem{},
		Meta:  dto.NewsListMeta{Limit: limit},
	}

	club, err := s.newsClub(ctx)
	if err != nil {
		return nil, err
	}
	if club == nil || strings.TrimSpace(club.Name) == "" {
		return empty, nil
	}
	if s.feed == nil || !s.feed.Enabled() {
		return nil, errors.NewServiceUnavailable("News provider is not configured", nil)
	}

	page := 1
	if cursor := strings.TrimSpace(filters.Cursor); cursor != "" {
		parsed, parseErr := strconv.Atoi(cursor)
		if parseErr != nil || parsed < 1 {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		page = parsed
	}

	result, err := s.feed.Search(ctx, club.Name, page, limit)
	if err != nil {
		return nil, err
	}

	out := make([]dto.NewsItem, 0, len(result.Items))
	for _, item := range result.Items {
		out = append(out, toListItem(item))
	}

	meta := dto.NewsListMeta{Limit: limit, HasMore: result.Page < result.TotalPages}
	if meta.HasMore {
		next := strconv.Itoa(result.Page + 1)
		meta.NextCursor = &next
	}

	return &dto.NewsListResponse{Items: out, Meta: meta}, nil
}

func (s *newsService) GetByID(ctx context.Context, id string) (*dto.NewsDetailResponse, error) {
	if s.feed == nil || !s.feed.Enabled() {
		return nil, errors.NewServiceUnavailable("News provider is not configured", nil)
	}
	providerID, err := newsfeed.DecodeID(id)
	if err != nil {
		return nil, err
	}
	article, err := s.feed.Get(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, errors.NewNotFound("News article not found", nil)
	}
	club, clubErr := s.newsClub(ctx)
	if clubErr != nil {
		return nil, clubErr
	}
	return toDetail(article, club), nil
}

func (s *newsService) GetNewsClub(ctx context.Context) (*dto.NewsClubResponse, error) {
	club, err := s.newsClub(ctx)
	if err != nil {
		return nil, err
	}
	return toNewsClubResponse(club), nil
}

func (s *newsService) SetNewsClub(ctx context.Context, req dto.SetNewsClubRequest) (*dto.NewsClubResponse, error) {
	var club *clubmodels.Club
	var err error
	switch {
	case req.ClubID != nil && *req.ClubID != uuid.Nil:
		club, err = s.clubs.FindByID(ctx, *req.ClubID)
	case strings.TrimSpace(req.ProviderTeamID) != "":
		if s.clubsFrom == nil {
			return nil, errors.NewInternal("Football club lookup is not configured", nil)
		}
		club, err = s.clubsFrom.EnsureClubFromProvider(ctx, strings.TrimSpace(req.ProviderTeamID))
	default:
		return nil, errors.NewBadRequest("club_id or provider_team_id is required", nil)
	}
	if err != nil {
		return nil, err
	}

	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	settings.NewsClubID = &club.ID
	settings.NewsClub = club
	if err := s.settings.Save(ctx, settings); err != nil {
		return nil, err
	}
	return toNewsClubResponse(club), nil
}

func (s *newsService) newsClub(ctx context.Context) (*clubmodels.Club, error) {
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if settings.NewsClubID != nil {
		if settings.NewsClub != nil {
			return settings.NewsClub, nil
		}
		return s.clubs.FindByID(ctx, *settings.NewsClubID)
	}
	if settings.FeaturedClubID == nil {
		return nil, nil
	}
	if settings.FeaturedClub != nil {
		return settings.FeaturedClub, nil
	}
	return s.clubs.FindByID(ctx, *settings.FeaturedClubID)
}

func toListItem(item newsfeed.Article) dto.NewsItem {
	stamp := formatTime(item.PublishedAt)
	return dto.NewsItem{
		ID:        newsfeed.EncodeID(item.ProviderID),
		Title:     item.Title,
		CreatedAt: stamp,
		UpdatedAt: stamp,
		ImageURL:  item.ImageURL,
	}
}

func toDetail(item *newsfeed.Article, club *clubmodels.Club) *dto.NewsDetailResponse {
	stamp := formatTime(item.PublishedAt)
	resp := &dto.NewsDetailResponse{
		ID:          newsfeed.EncodeID(item.ProviderID),
		Title:       item.Title,
		BodyHTML:    item.BodyHTML,
		ImageURL:    item.ImageURL,
		PublishedAt: stamp,
		IsActive:    true,
		CreatedAt:   stamp,
		UpdatedAt:   stamp,
	}
	if club != nil {
		resp.ClubID = &club.ID
	}
	return resp
}

func toNewsClubResponse(club *clubmodels.Club) *dto.NewsClubResponse {
	if club == nil {
		return &dto.NewsClubResponse{}
	}
	return &dto.NewsClubResponse{
		ClubID:         &club.ID,
		Name:           club.Name,
		LogoURL:        club.LogoURL,
		ProviderTeamID: club.ProviderTeamID,
		Provider:       club.Provider,
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
