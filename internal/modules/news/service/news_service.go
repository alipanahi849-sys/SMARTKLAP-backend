package service

import (
	"context"
	"time"

	"clap/internal/modules/news/dto"
	"clap/internal/modules/news/models"
	"clap/internal/modules/news/repository"
	"clap/internal/shared/utils"
)

type NewsService interface {
	List(ctx context.Context, page, limit int) (*dto.NewsListResponse, error)
	// Preview powers the Home (Club Mode) club_news card (contract §3.2).
	Preview(ctx context.Context, limit int) ([]dto.NewsItem, error)
}

type newsService struct {
	newsRepo repository.NewsRepository
}

func NewNewsService(newsRepo repository.NewsRepository) NewsService {
	return &newsService{newsRepo: newsRepo}
}

func (s *newsService) List(ctx context.Context, page, limit int) (*dto.NewsListResponse, error) {
	items, total, err := s.newsRepo.FindAll(ctx, page, limit)
	if err != nil {
		return nil, err
	}
	return &dto.NewsListResponse{
		Items: toItems(items),
		Meta:  utils.NewListMeta(total, page, limit),
	}, nil
}

func (s *newsService) Preview(ctx context.Context, limit int) ([]dto.NewsItem, error) {
	items, err := s.newsRepo.FindPreview(ctx, limit)
	if err != nil {
		return nil, err
	}
	return toItems(items), nil
}

func toItems(items []models.News) []dto.NewsItem {
	result := make([]dto.NewsItem, len(items))
	for i, n := range items {
		result[i] = dto.NewsItem{
			ID:       n.ID,
			Title:    n.Title,
			Date:     n.PublishedAt.UTC().Format(time.RFC3339),
			ImageURL: n.ImageURL,
		}
	}
	return result
}
