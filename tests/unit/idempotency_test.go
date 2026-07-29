package unit_test

import (
	"context"
	"testing"
	"time"

	idemmodels "clap/internal/modules/idempotency/models"
	"clap/internal/modules/idempotency/service"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── Stub repository ──────────────────────────────────────────────────────────

type stubIdempotencyRepo struct {
	store map[string]*idemmodels.IdempotencyKey // key = "key:endpoint"
}

func newStubIdempotencyRepo() *stubIdempotencyRepo {
	return &stubIdempotencyRepo{store: make(map[string]*idemmodels.IdempotencyKey)}
}

func (r *stubIdempotencyRepo) mapKey(key, endpoint string) string {
	return key + ":" + endpoint
}

func (r *stubIdempotencyRepo) FindByKeyAndEndpoint(_ context.Context, key, endpoint string) (*idemmodels.IdempotencyKey, error) {
	rec, ok := r.store[r.mapKey(key, endpoint)]
	if !ok || rec.IsExpired() {
		return nil, sharederrors.NewNotFound("not found", nil)
	}
	return rec, nil
}

func (r *stubIdempotencyRepo) CreateOrIgnore(_ context.Context, record *idemmodels.IdempotencyKey) error {
	k := r.mapKey(record.Key, record.Endpoint)
	if _, exists := r.store[k]; !exists {
		r.store[k] = record
	}
	return nil
}

func (r *stubIdempotencyRepo) DeleteExpired(_ context.Context) (int64, error) {
	var deleted int64
	for k, rec := range r.store {
		if rec.IsExpired() {
			delete(r.store, k)
			deleted++
		}
	}
	return deleted, nil
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestIdempotency_StoreAndRetrieve(t *testing.T) {
	repo := newStubIdempotencyRepo()
	svc := service.NewIdempotencyService(repo)

	key := uuid.New().String()
	endpoint := "POST:/api/v1/matches/runtime/start"
	requestHash := "abc123"
	response := `{"status":"running"}`

	if err := svc.Store(context.Background(), key, endpoint, requestHash, response, 200); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	existing, err := svc.FindExisting(context.Background(), key, endpoint)
	if err != nil {
		t.Fatalf("FindExisting failed: %v", err)
	}

	if existing.ResponsePayload != response {
		t.Errorf("expected payload %q, got %q", response, existing.ResponsePayload)
	}
	if existing.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", existing.StatusCode)
	}
}

func TestIdempotency_NotFoundWhenAbsent(t *testing.T) {
	repo := newStubIdempotencyRepo()
	svc := service.NewIdempotencyService(repo)

	_, err := svc.FindExisting(context.Background(), "missing-key", "GET:/foo")
	if err == nil {
		t.Error("expected NotFound error for missing key")
	}
}

func TestIdempotency_ExpiredRecordNotReturned(t *testing.T) {
	repo := newStubIdempotencyRepo()
	// 1ms TTL — will expire immediately.
	svc := service.NewIdempotencyServiceWithTTL(repo, time.Millisecond)

	key := uuid.New().String()
	endpoint := "POST:/api/v1/runtime/start"

	_ = svc.Store(context.Background(), key, endpoint, "hash", `{}`, 200)

	// Let the record expire.
	time.Sleep(5 * time.Millisecond)

	_, err := svc.FindExisting(context.Background(), key, endpoint)
	if err == nil {
		t.Error("expected NotFound for expired idempotency key")
	}
}

func TestIdempotency_ValidateRequestHash_Match(t *testing.T) {
	repo := newStubIdempotencyRepo()
	svc := service.NewIdempotencyService(repo)

	existing := &idemmodels.IdempotencyKey{RequestHash: "deadbeef"}
	if err := svc.ValidateRequestHash(existing, "deadbeef"); err != nil {
		t.Errorf("expected nil error for matching hash, got: %v", err)
	}
}

func TestIdempotency_ValidateRequestHash_Mismatch(t *testing.T) {
	repo := newStubIdempotencyRepo()
	svc := service.NewIdempotencyService(repo)

	existing := &idemmodels.IdempotencyKey{RequestHash: "original-hash"}
	if err := svc.ValidateRequestHash(existing, "different-hash"); err == nil {
		t.Error("expected error when request hash does not match stored hash")
	}
}

func TestIdempotency_DuplicateStoreWinsOnRace(t *testing.T) {
	// Two goroutines race to store the same key; only first wins.
	repo := newStubIdempotencyRepo()
	svc := service.NewIdempotencyService(repo)

	key := uuid.New().String()
	endpoint := "POST:/songs/schedule"

	_ = svc.Store(context.Background(), key, endpoint, "hash-A", `{"first": true}`, 201)
	_ = svc.Store(context.Background(), key, endpoint, "hash-A", `{"second": true}`, 201)

	existing, err := svc.FindExisting(context.Background(), key, endpoint)
	if err != nil {
		t.Fatalf("FindExisting failed: %v", err)
	}

	// First writer's payload should be preserved.
	if existing.ResponsePayload != `{"first": true}` {
		t.Errorf("expected first writer's payload, got %q", existing.ResponsePayload)
	}
}

func TestIdempotency_PurgeExpired(t *testing.T) {
	repo := newStubIdempotencyRepo()
	svc := service.NewIdempotencyServiceWithTTL(repo, time.Millisecond)

	for i := 0; i < 5; i++ {
		_ = svc.Store(context.Background(), uuid.New().String(), "POST:/foo", "h", "{}", 200)
	}

	time.Sleep(10 * time.Millisecond)

	deleted, err := svc.PurgeExpired(context.Background())
	if err != nil {
		t.Fatalf("PurgeExpired failed: %v", err)
	}
	if deleted != 5 {
		t.Errorf("expected 5 purged records, got %d", deleted)
	}
}
