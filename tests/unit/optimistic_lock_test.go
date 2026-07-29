package unit_test

import (
	"context"
	"testing"
	"time"

	"clap/internal/modules/matchruntime/models"
	runtimerepo "clap/internal/modules/matchruntime/repository"
	"clap/internal/modules/matchruntime/service"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── Version-aware stub repository ───────────────────────────────────────────

type versionedRuntimeRepo struct {
	state          *models.MatchRuntimeState
	concurrentEdit bool // simulate a concurrent writer already changing the DB row
}

func (r *versionedRuntimeRepo) Create(_ context.Context, s *models.MatchRuntimeState) error {
	r.state = s
	return nil
}

func (r *versionedRuntimeRepo) FindByMatchID(_ context.Context, _ uuid.UUID) (*models.MatchRuntimeState, error) {
	if r.state == nil {
		return nil, sharederrors.NewNotFound("not found", nil)
	}
	return r.state, nil
}

// Update mirrors the optimistic-locking behaviour of the real repository.
// When concurrentEdit is true the DB row version already advanced, so we
// return a Conflict error just like the real repo would.
func (r *versionedRuntimeRepo) Update(_ context.Context, s *models.MatchRuntimeState) error {
	if r.concurrentEdit {
		return sharederrors.NewConflict(
			"Concurrent modification detected on match runtime state — please retry", nil,
		)
	}
	s.Version++
	r.state = s
	return nil
}

// Compile-time interface guard.
var _ runtimerepo.MatchRuntimeRepository = (*versionedRuntimeRepo)(nil)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func svcFromVersioned(repo *versionedRuntimeRepo, clock *testClock) service.MatchRuntimeService {
	return service.NewMatchRuntimeService(repo, clock)
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestOptimisticLock_VersionIncrementedOnUpdate(t *testing.T) {
	repo := &versionedRuntimeRepo{}
	state := &models.MatchRuntimeState{
		ID:      uuid.New(),
		MatchID: uuid.New(),
		Status:  models.RuntimeStatusRunning,
		Version: 0,
	}
	repo.state = state

	// First update: pause.
	state.Status = models.RuntimeStatusPaused
	if err := repo.Update(context.Background(), state); err != nil {
		t.Fatalf("unexpected error on first update: %v", err)
	}
	if state.Version != 1 {
		t.Errorf("expected version=1 after first update, got %d", state.Version)
	}

	// Second update: resume.
	state.Status = models.RuntimeStatusRunning
	if err := repo.Update(context.Background(), state); err != nil {
		t.Fatalf("unexpected error on second update: %v", err)
	}
	if state.Version != 2 {
		t.Errorf("expected version=2 after second update, got %d", state.Version)
	}
}

func TestOptimisticLock_ConflictReturnedWhenVersionMismatch(t *testing.T) {
	repo := &versionedRuntimeRepo{concurrentEdit: true}
	state := &models.MatchRuntimeState{
		ID:      uuid.New(),
		MatchID: uuid.New(),
		Status:  models.RuntimeStatusRunning,
		Version: 0,
	}
	repo.state = state

	state.Status = models.RuntimeStatusPaused
	err := repo.Update(context.Background(), state)
	if err == nil {
		t.Fatal("expected conflict error when concurrent writer already modified the row")
	}

	appErr, ok := err.(*sharederrors.AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T", err)
	}
	if appErr.StatusCode != 409 {
		t.Errorf("expected HTTP 409 conflict, got %d", appErr.StatusCode)
	}
}

func TestOptimisticLock_PauseMatchPropagatesConflict(t *testing.T) {
	matchID := uuid.New()
	repo := &versionedRuntimeRepo{
		state: &models.MatchRuntimeState{
			ID:      uuid.New(),
			MatchID: matchID,
			Status:  models.RuntimeStatusRunning,
			Version: 3,
		},
		concurrentEdit: true,
	}

	svc := svcFromVersioned(repo, &testClock{t: time.Now()})

	_, err := svc.PauseMatch(context.Background(), matchID, adminCtx())
	if err == nil {
		t.Fatal("PauseMatch should return conflict error when optimistic lock fails")
	}

	appErr, ok := err.(*sharederrors.AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T", err)
	}
	if appErr.StatusCode != 409 {
		t.Errorf("expected HTTP 409 conflict, got %d", appErr.StatusCode)
	}
}

func TestOptimisticLock_SuccessfulLifecycleVersionsGrow(t *testing.T) {
	matchID := uuid.New()
	repo := &versionedRuntimeRepo{}
	clock := &testClock{t: fixedTime()}
	svc := svcFromVersioned(repo, clock)

	// start → version goes from 0 to 0 (Create, not Update)
	_, _ = svc.StartMatch(context.Background(), matchID, adminCtx())

	// pause → version 1
	clock.Advance(30 * time.Second)
	_, _ = svc.PauseMatch(context.Background(), matchID, adminCtx())
	if repo.state.Version != 1 {
		t.Errorf("expected version=1 after pause, got %d", repo.state.Version)
	}

	// resume → version 2
	clock.Advance(10 * time.Second)
	_, _ = svc.ResumeMatch(context.Background(), matchID, adminCtx())
	if repo.state.Version != 2 {
		t.Errorf("expected version=2 after resume, got %d", repo.state.Version)
	}

	// end → version 3
	clock.Advance(60 * time.Second)
	_, _ = svc.EndMatch(context.Background(), matchID, adminCtx())
	if repo.state.Version != 3 {
		t.Errorf("expected version=3 after end, got %d", repo.state.Version)
	}
}
