package service

import (
	"context"
	"time"

	"clap/internal/modules/idempotency/models"
	"clap/internal/modules/idempotency/repository"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

const defaultTTL = 24 * time.Hour

// IdempotencyService stores and retrieves cached responses for idempotent
// mutating requests. It is used by the middleware layer.
type IdempotencyService interface {
	// FindExisting returns a cached response if key+endpoint match and are not expired.
	// Returns NotFound when no valid cached response exists.
	FindExisting(ctx context.Context, key, endpoint string) (*models.IdempotencyKey, error)
	// Store persists the response so future duplicate requests can be short-circuited.
	// On a race (two concurrent identical requests) only one writer wins; the loser
	// can immediately read back the winner's result via FindExisting.
	Store(ctx context.Context, key, endpoint, requestHash, responsePayload string, statusCode int) error
	// ValidateRequestHash ensures a repeated request has the same body as the original.
	// If the hash differs the client is sending a different payload with the same key,
	// which is a client error.
	ValidateRequestHash(existing *models.IdempotencyKey, incomingHash string) error
	// PurgeExpired deletes expired records; intended for manual operational use.
	PurgeExpired(ctx context.Context) (int64, error)
}

type idempotencyService struct {
	repo repository.IdempotencyRepository
	ttl  time.Duration
}

func NewIdempotencyService(repo repository.IdempotencyRepository) IdempotencyService {
	return &idempotencyService{repo: repo, ttl: defaultTTL}
}

func NewIdempotencyServiceWithTTL(repo repository.IdempotencyRepository, ttl time.Duration) IdempotencyService {
	return &idempotencyService{repo: repo, ttl: ttl}
}

func (s *idempotencyService) FindExisting(ctx context.Context, key, endpoint string) (*models.IdempotencyKey, error) {
	return s.repo.FindByKeyAndEndpoint(ctx, key, endpoint)
}

func (s *idempotencyService) Store(ctx context.Context, key, endpoint, requestHash, responsePayload string, statusCode int) error {
	record := &models.IdempotencyKey{
		ID:              uuid.New(),
		Key:             key,
		Endpoint:        endpoint,
		RequestHash:     requestHash,
		ResponsePayload: responsePayload,
		StatusCode:      statusCode,
		ExpiresAt:       time.Now().UTC().Add(s.ttl),
	}
	return s.repo.CreateOrIgnore(ctx, record)
}

func (s *idempotencyService) ValidateRequestHash(existing *models.IdempotencyKey, incomingHash string) error {
	if existing.RequestHash != incomingHash {
		return sharederrors.NewBadRequest(
			"Idempotency key reuse detected: request body differs from original request", nil,
		)
	}
	return nil
}

func (s *idempotencyService) PurgeExpired(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpired(ctx)
}
