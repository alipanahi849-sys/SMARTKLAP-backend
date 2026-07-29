package unit_test

import (
	"context"
	"testing"
	"time"

	"clap/internal/modules/matchruntime/models"
	"clap/internal/modules/matchruntime/service"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

// ─── Stub repository ──────────────────────────────────────────────────────────

type stubMatchRuntimeRepo struct {
	state *models.MatchRuntimeState
}

func (r *stubMatchRuntimeRepo) Create(_ context.Context, s *models.MatchRuntimeState) error {
	r.state = s
	return nil
}
func (r *stubMatchRuntimeRepo) FindByMatchID(_ context.Context, _ uuid.UUID) (*models.MatchRuntimeState, error) {
	if r.state == nil {
		return nil, sharederrors.NewNotFound("Match runtime state not found", nil)
	}
	return r.state, nil
}
func (r *stubMatchRuntimeRepo) Update(_ context.Context, s *models.MatchRuntimeState) error {
	r.state = s
	return nil
}

// ─── Test clock ───────────────────────────────────────────────────────────────

type testClock struct{ t time.Time }

func (c *testClock) Now() time.Time          { return c.t }
func (c *testClock) Advance(d time.Duration) { c.t = c.t.Add(d) }

// ─── Helpers ──────────────────────────────────────────────────────────────────

func adminCtx() *utils.AuthorizationContext {
	return utils.NewAuthorizationContext(uuid.New(), []string{"admin"}, nil)
}

func fixedTime() time.Time {
	return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
}

func ctx() context.Context {
	return context.Background()
}

func newRuntimeSvc(repo *stubMatchRuntimeRepo, clock *testClock) service.MatchRuntimeService {
	return service.NewMatchRuntimeService(repo, clock)
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestMatchRuntime_StartCreatesState(t *testing.T) {
	repo := &stubMatchRuntimeRepo{}
	clock := &testClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	svc := service.NewMatchRuntimeService(repo, clock)

	matchID := uuid.New()
	resp, err := svc.StartMatch(context.Background(), matchID, adminCtx())
	if err != nil {
		t.Fatalf("StartMatch failed: %v", err)
	}

	if resp.Status != models.RuntimeStatusRunning {
		t.Errorf("expected status=running, got %s", resp.Status)
	}
	if resp.StartedAt == nil {
		t.Error("started_at must be set")
	}
}

func TestMatchRuntime_DoubleStartConflict(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &stubMatchRuntimeRepo{}
	clock := &testClock{t: now}
	svc := service.NewMatchRuntimeService(repo, clock)

	matchID := uuid.New()
	_, _ = svc.StartMatch(context.Background(), matchID, adminCtx())
	_, err := svc.StartMatch(context.Background(), matchID, adminCtx())
	if err == nil {
		t.Error("expected conflict error on double start")
	}
}

func TestMatchRuntime_PauseAndResume(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &stubMatchRuntimeRepo{}
	clock := &testClock{t: now}
	svc := service.NewMatchRuntimeService(repo, clock)

	matchID := uuid.New()
	_, _ = svc.StartMatch(context.Background(), matchID, adminCtx())

	// Advance 30s and pause.
	clock.Advance(30 * time.Second)
	_, err := svc.PauseMatch(context.Background(), matchID, adminCtx())
	if err != nil {
		t.Fatalf("PauseMatch failed: %v", err)
	}
	if repo.state.Status != models.RuntimeStatusPaused {
		t.Errorf("expected status=paused, got %s", repo.state.Status)
	}

	// Advance 10s while paused and resume.
	clock.Advance(10 * time.Second)
	_, err = svc.ResumeMatch(context.Background(), matchID, adminCtx())
	if err != nil {
		t.Fatalf("ResumeMatch failed: %v", err)
	}
	if repo.state.Status != models.RuntimeStatusRunning {
		t.Errorf("expected status=running after resume, got %s", repo.state.Status)
	}
	if repo.state.TotalPausedMs != 10_000 {
		t.Errorf("expected total_paused_ms=10000, got %d", repo.state.TotalPausedMs)
	}
}

func TestMatchRuntime_ElapsedTimeCorrectness(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &stubMatchRuntimeRepo{}
	clock := &testClock{t: now}
	svc := service.NewMatchRuntimeService(repo, clock)

	matchID := uuid.New()
	_, _ = svc.StartMatch(context.Background(), matchID, adminCtx())

	// Run for 60s.
	clock.Advance(60 * time.Second)

	// Pause for 15s.
	_, _ = svc.PauseMatch(context.Background(), matchID, adminCtx())
	clock.Advance(15 * time.Second)

	// Resume and run another 30s.
	_, _ = svc.ResumeMatch(context.Background(), matchID, adminCtx())
	clock.Advance(30 * time.Second)

	// CurrentMatchTime should report 90s elapsed (60 + 30, pause excluded).
	timeResp, err := svc.CurrentMatchTime(context.Background(), matchID)
	if err != nil {
		t.Fatalf("CurrentMatchTime failed: %v", err)
	}

	expectedMs := int64(90_000)
	if timeResp.ElapsedMs != expectedMs {
		t.Errorf("expected elapsed_ms=%d, got %d", expectedMs, timeResp.ElapsedMs)
	}
}

func TestMatchRuntime_PauseWhileNotRunning(t *testing.T) {
	repo := &stubMatchRuntimeRepo{}
	clock := &testClock{t: time.Now()}
	svc := service.NewMatchRuntimeService(repo, clock)

	matchID := uuid.New()
	_, _ = svc.StartMatch(context.Background(), matchID, adminCtx())
	_, _ = svc.PauseMatch(context.Background(), matchID, adminCtx())
	_, err := svc.PauseMatch(context.Background(), matchID, adminCtx())
	if err == nil {
		t.Error("expected error when pausing an already paused match")
	}
}

func TestMatchRuntime_EndMatch(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &stubMatchRuntimeRepo{}
	clock := &testClock{t: now}
	svc := service.NewMatchRuntimeService(repo, clock)

	matchID := uuid.New()
	_, _ = svc.StartMatch(context.Background(), matchID, adminCtx())
	clock.Advance(90 * time.Second)

	resp, err := svc.EndMatch(context.Background(), matchID, adminCtx())
	if err != nil {
		t.Fatalf("EndMatch failed: %v", err)
	}
	if resp.Status != models.RuntimeStatusEnded {
		t.Errorf("expected status=ended, got %s", resp.Status)
	}
	if resp.EndedAt == nil {
		t.Error("ended_at must be set")
	}
}

func TestMatchRuntime_ForbiddenForNonAdmin(t *testing.T) {
	repo := &stubMatchRuntimeRepo{}
	svc := service.NewMatchRuntimeService(repo, &testClock{t: time.Now()})

	nonAdmin := utils.NewAuthorizationContext(uuid.New(), []string{"user"}, nil)
	_, err := svc.StartMatch(context.Background(), uuid.New(), nonAdmin)
	if err == nil {
		t.Error("expected forbidden error for non-admin user")
	}
}
