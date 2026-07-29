package unit_test

import (
	"context"
	"testing"
	"time"

	realtimemodels "clap/internal/modules/realtime/models"
	realtimerepo "clap/internal/modules/realtime/repository"
	"clap/internal/modules/realtime/service"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── Stub heartbeat repository ────────────────────────────────────────────────

type stubHeartbeatRepo struct {
	heartbeats []*realtimemodels.ClientHeartbeat
}

func (r *stubHeartbeatRepo) Record(_ context.Context, hb *realtimemodels.ClientHeartbeat) error {
	if hb.ID == uuid.Nil {
		hb.ID = uuid.New()
	}
	r.heartbeats = append(r.heartbeats, hb)
	return nil
}

func (r *stubHeartbeatRepo) FindBySessionID(_ context.Context, sessionID uuid.UUID, limit int) ([]*realtimemodels.ClientHeartbeat, error) {
	var out []*realtimemodels.ClientHeartbeat
	for _, hb := range r.heartbeats {
		if hb.SessionID == sessionID {
			out = append(out, hb)
			if limit > 0 && len(out) == limit {
				break
			}
		}
	}
	return out, nil
}

func (r *stubHeartbeatRepo) FindLatestByUser(_ context.Context, sessionID, userID uuid.UUID) (*realtimemodels.ClientHeartbeat, error) {
	for i := len(r.heartbeats) - 1; i >= 0; i-- {
		hb := r.heartbeats[i]
		if hb.SessionID == sessionID && hb.UserID == userID {
			return hb, nil
		}
	}
	return nil, sharederrors.NewNotFound("not found", nil)
}

func (r *stubHeartbeatRepo) AverageDriftBySession(_ context.Context, sessionID uuid.UUID) (float64, error) {
	var total int64
	var count int64
	for _, hb := range r.heartbeats {
		if hb.SessionID == sessionID {
			total += hb.DriftMs
			count++
		}
	}
	if count == 0 {
		return 0, nil
	}
	return float64(total) / float64(count), nil
}

func (r *stubHeartbeatRepo) DeleteOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	var remaining []*realtimemodels.ClientHeartbeat
	var deleted int64
	for _, hb := range r.heartbeats {
		if hb.CreatedAt.Before(cutoff) {
			deleted++
		} else {
			remaining = append(remaining, hb)
		}
	}
	r.heartbeats = remaining
	return deleted, nil
}

var _ realtimerepo.ClientHeartbeatRepository = (*stubHeartbeatRepo)(nil)

// ─── Tests ────────────────────────────────────────────────────────────────────

func seedHeartbeats(repo *stubHeartbeatRepo, count int, createdAt time.Time) {
	sessionID := uuid.New()
	for i := 0; i < count; i++ {
		repo.heartbeats = append(repo.heartbeats, &realtimemodels.ClientHeartbeat{
			ID:              uuid.New(),
			SessionID:       sessionID,
			UserID:          uuid.New(),
			ClientTimestamp: createdAt.UnixMilli(),
			ServerTimestamp: createdAt.UnixMilli(),
			DriftMs:         0,
			CreatedAt:       createdAt,
		})
	}
}

func TestCleanup_DeletesOldHeartbeats(t *testing.T) {
	repo := &stubHeartbeatRepo{}

	old := time.Now().UTC().AddDate(0, 0, -35)   // 35 days ago
	recent := time.Now().UTC().AddDate(0, 0, -5) // 5 days ago

	seedHeartbeats(repo, 4, old)
	seedHeartbeats(repo, 2, recent)

	svc := service.NewHeartbeatCleanupService(repo)
	deleted, err := svc.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Default retention is 30 days; 35-day-old records should be deleted.
	if deleted != 4 {
		t.Errorf("expected 4 deleted, got %d", deleted)
	}
	if len(repo.heartbeats) != 2 {
		t.Errorf("expected 2 remaining heartbeats, got %d", len(repo.heartbeats))
	}
}

func TestCleanup_CustomRetentionDays(t *testing.T) {
	repo := &stubHeartbeatRepo{}

	tenDaysAgo := time.Now().UTC().AddDate(0, 0, -10)
	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2)

	seedHeartbeats(repo, 3, tenDaysAgo)
	seedHeartbeats(repo, 5, twoDaysAgo)

	svc := service.NewHeartbeatCleanupServiceWithConfig(repo, service.HeartbeatCleanupConfig{
		RetentionDays: 7, // only keep 7 days
	})

	deleted, err := svc.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	if deleted != 3 {
		t.Errorf("expected 3 deleted (10-day-old records), got %d", deleted)
	}
	if len(repo.heartbeats) != 5 {
		t.Errorf("expected 5 remaining heartbeats, got %d", len(repo.heartbeats))
	}
}

func TestCleanup_NothingDeletedWhenAllRecent(t *testing.T) {
	repo := &stubHeartbeatRepo{}
	seedHeartbeats(repo, 5, time.Now().UTC().Add(-time.Hour))

	svc := service.NewHeartbeatCleanupService(repo)
	deleted, err := svc.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted for recent heartbeats, got %d", deleted)
	}
}

func TestCleanup_DefaultRetentionFallback(t *testing.T) {
	repo := &stubHeartbeatRepo{}
	// Config with invalid retention (0) should fall back to default (30 days).
	svc := service.NewHeartbeatCleanupServiceWithConfig(repo, service.HeartbeatCleanupConfig{RetentionDays: 0})

	// Just ensure it doesn't panic or error.
	_, err := svc.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
