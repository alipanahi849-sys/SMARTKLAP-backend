package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	clubmodels "clap/internal/modules/club/models"
	clubrepo "clap/internal/modules/club/repository"
	"clap/internal/modules/news/dto"
	"clap/internal/modules/news/models"
	"clap/internal/modules/news/repository"
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
	SearchNewsClubs(ctx context.Context, query string) ([]dto.NewsClubCandidate, error)
}

type newsService struct {
	feed      newsfeed.Provider
	local     repository.NewsRepository
	settings  settingsrepo.SettingsRepository
	clubs     clubrepo.ClubRepository
	clubsFrom ClubProvisioner
}

func NewNewsService(
	feed newsfeed.Provider,
	local repository.NewsRepository,
	settings settingsrepo.SettingsRepository,
	clubs clubrepo.ClubRepository,
	clubsFrom ClubProvisioner,
) NewsService {
	return &newsService{
		feed:      feed,
		local:     local,
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

	if s.feed == nil || !s.feed.Enabled() {
		return s.listLocal(ctx, filters, limit, empty)
	}

	club, err := s.newsClub(ctx)
	if err != nil {
		return nil, err
	}
	if club == nil || strings.TrimSpace(club.Name) == "" {
		return empty, nil
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

func (s *newsService) listLocal(
	ctx context.Context,
	filters dto.NewsListFilters,
	limit int,
	empty *dto.NewsListResponse,
) (*dto.NewsListResponse, error) {
	if s.local == nil {
		return empty, nil
	}

	var after *repository.NewsCursorAnchor
	if cursor := strings.TrimSpace(filters.Cursor); cursor != "" {
		cursorID, parseErr := uuid.Parse(cursor)
		if parseErr != nil {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		cursorItem, err := s.local.FindByID(ctx, cursorID)
		if err != nil {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		after = &repository.NewsCursorAnchor{
			PublishedAt: cursorItem.PublishedAt,
			ID:          cursorItem.ID,
		}
	}

	items, err := s.local.ListAfter(ctx, limit+1, after)
	if err != nil {
		return nil, err
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	out := make([]dto.NewsItem, 0, len(items))
	for _, item := range items {
		out = append(out, toListItemFromModel(item))
	}

	meta := dto.NewsListMeta{Limit: limit, HasMore: hasMore}
	if hasMore && len(items) > 0 {
		next := items[len(items)-1].ID.String()
		meta.NextCursor = &next
	}

	return &dto.NewsListResponse{Items: out, Meta: meta}, nil
}

func (s *newsService) GetByID(ctx context.Context, id string) (*dto.NewsDetailResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.NewBadRequest("Invalid news ID", nil)
	}

	if parsed, err := uuid.Parse(id); err == nil {
		if s.local == nil {
			return nil, errors.NewNotFound("News article not found", nil)
		}
		item, err := s.local.FindByID(ctx, parsed)
		if err != nil {
			return nil, err
		}
		if !item.IsActive {
			return nil, errors.NewNotFound("News article not found", nil)
		}
		return toDetailFromModel(item), nil
	}

	if s.feed == nil || !s.feed.Enabled() {
		return nil, errors.NewNotFound("News article not found", nil)
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
	case strings.TrimSpace(req.Name) != "":
		club, err = s.ensureClubByName(ctx, strings.TrimSpace(req.Name))
	default:
		return nil, errors.NewBadRequest("club_id, provider_team_id, or name is required", nil)
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

func (s *newsService) SearchNewsClubs(ctx context.Context, query string) ([]dto.NewsClubCandidate, error) {
	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return []dto.NewsClubCandidate{}, nil
	}

	clubs, _, err := s.clubs.Search(ctx, query, 1, 20)
	if err != nil {
		return nil, err
	}

	out := make([]dto.NewsClubCandidate, 0, len(clubs)+1)
	seen := map[string]struct{}{}
	for i := range clubs {
		club := clubs[i]
		key := strings.ToLower(strings.TrimSpace(club.Name))
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
		id := club.ID
		out = append(out, dto.NewsClubCandidate{
			ClubID:         &id,
			Name:           club.Name,
			Country:        club.Country,
			LogoURL:        club.LogoURL,
			ProviderTeamID: club.ProviderTeamID,
			Provider:       club.Provider,
		})
	}

	// Always offer creating/using the exact typed name for Guardian search.
	want := strings.ToLower(query)
	if _, ok := seen[want]; !ok {
		out = append([]dto.NewsClubCandidate{{Name: query}}, out...)
	}

	return out, nil
}

func (s *newsService) ensureClubByName(ctx context.Context, name string) (*clubmodels.Club, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.NewBadRequest("name is required", nil)
	}

	clubs, _, err := s.clubs.Search(ctx, name, 1, 50)
	if err != nil {
		return nil, err
	}
	want := strings.ToLower(name)
	for i := range clubs {
		if strings.ToLower(strings.TrimSpace(clubs[i].Name)) == want {
			return &clubs[i], nil
		}
	}

	club := &clubmodels.Club{
		Name:     name,
		IsActive: true,
	}
	if err := s.clubs.Create(ctx, club); err != nil {
		return nil, err
	}
	return club, nil
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

func toListItemFromModel(item models.News) dto.NewsItem {
	return dto.NewsItem{
		ID:        item.ID.String(),
		Title:     item.Title,
		CreatedAt: formatTime(item.CreatedAt),
		UpdatedAt: formatTime(item.UpdatedAt),
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

func toDetailFromModel(item *models.News) *dto.NewsDetailResponse {
	return &dto.NewsDetailResponse{
		ID:          item.ID.String(),
		ClubID:      item.ClubID,
		Title:       item.Title,
		BodyHTML:    item.BodyHTML,
		ImageURL:    item.ImageURL,
		PublishedAt: formatTime(item.PublishedAt),
		IsActive:    item.IsActive,
		CreatedAt:   formatTime(item.CreatedAt),
		UpdatedAt:   formatTime(item.UpdatedAt),
	}
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
