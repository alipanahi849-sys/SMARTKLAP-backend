package newsfeed

import (
	"context"
	"time"
)

const ProviderGuardian = "guardian"

// Article is one news item from the upstream publisher.
type Article struct {
	ProviderID  string
	Title       string
	BodyHTML    string
	ImageURL    string
	PublishedAt time.Time
}

// SearchResult is one page of club news from the publisher.
type SearchResult struct {
	Items      []Article
	Page       int
	TotalPages int
}

// Provider fetches real football news from an upstream publisher API.
type Provider interface {
	Name() string
	Enabled() bool
	Search(ctx context.Context, clubName string, page, pageSize int) (*SearchResult, error)
	Get(ctx context.Context, providerID string) (*Article, error)
}
