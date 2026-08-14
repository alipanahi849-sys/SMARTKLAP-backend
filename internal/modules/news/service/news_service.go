package service

import (
	"context"
	"time"

	"clap/internal/modules/news/dto"
	"clap/internal/modules/news/models"
	"clap/internal/modules/news/repository"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
)

type NewsService interface {
	List(ctx context.Context, filters dto.NewsListFilters) (*dto.NewsListResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.NewsDetailResponse, error)
}

type newsService struct {
	newsRepo repository.NewsRepository
}

func NewNewsService(newsRepo repository.NewsRepository) NewsService {
	return &newsService{newsRepo: newsRepo}
}

func (s *newsService) List(ctx context.Context, filters dto.NewsListFilters) (*dto.NewsListResponse, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}

	var after *repository.NewsCursorAnchor
	if filters.Cursor != nil {
		cursorItem, err := s.newsRepo.FindByID(ctx, *filters.Cursor)
		if err != nil {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		after = &repository.NewsCursorAnchor{
			PublishedAt: cursorItem.PublishedAt,
			ID:          cursorItem.ID,
		}
	}

	items, err := s.newsRepo.ListAfter(ctx, limit+1, after)
	if err != nil {
		return nil, err
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	out := make([]dto.NewsItem, 0, len(items))
	for _, item := range items {
		out = append(out, toListItem(item))
	}

	meta := dto.NewsListMeta{Limit: limit, HasMore: hasMore}
	if hasMore && len(items) > 0 {
		lastID := items[len(items)-1].ID
		meta.NextCursor = &lastID
	}

	return &dto.NewsListResponse{Items: out, Meta: meta}, nil
}

func (s *newsService) GetByID(ctx context.Context, id uuid.UUID) (*dto.NewsDetailResponse, error) {
	item, err := s.newsRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !item.IsActive {
		return nil, errors.NewNotFound("News article not found", nil)
	}
	return toDetail(item), nil
}

func toListItem(item models.News) dto.NewsItem {
	return dto.NewsItem{
		ID:        item.ID,
		Title:     item.Title,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
		ImageURL:  item.ImageURL,
	}
}

func toDetail(item *models.News) *dto.NewsDetailResponse {
	return &dto.NewsDetailResponse{
		ID:          item.ID,
		ClubID:      item.ClubID,
		Title:       item.Title,
		BodyHTML:    item.Body,
		ImageURL:    item.ImageURL,
		PublishedAt: item.PublishedAt.UTC().Format(time.RFC3339),
		IsActive:    item.IsActive,
		CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
