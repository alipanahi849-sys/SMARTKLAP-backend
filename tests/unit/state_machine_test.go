package unit_test

import (
	"testing"

	"clap/internal/modules/matchruntime/models"

	"github.com/google/uuid"
)

// transitionCase describes a single state-machine transition assertion.
type transitionCase struct {
	from    models.RuntimeStatus
	to      models.RuntimeStatus
	wantErr bool
	label   string
}

func TestStateMachine_Transitions(t *testing.T) {
	cases := []transitionCase{
		// ── Allowed ───────────────────────────────────────────────────
		{models.RuntimeStatusPending, models.RuntimeStatusRunning, false, "pending→running (StartMatch)"},
		{models.RuntimeStatusRunning, models.RuntimeStatusPaused, false, "running→paused (PauseMatch)"},
		{models.RuntimeStatusPaused, models.RuntimeStatusRunning, false, "paused→running (ResumeMatch)"},
		{models.RuntimeStatusRunning, models.RuntimeStatusEnded, false, "running→ended (EndMatch)"},
		{models.RuntimeStatusPaused, models.RuntimeStatusEnded, false, "paused→ended (EndMatch)"},

		// ── Forbidden ─────────────────────────────────────────────────
		{models.RuntimeStatusPending, models.RuntimeStatusEnded, true, "pending→ended (must start first)"},
		{models.RuntimeStatusPending, models.RuntimeStatusPaused, true, "pending→paused (invalid)"},
		{models.RuntimeStatusRunning, models.RuntimeStatusRunning, true, "running→running (already running)"},
		{models.RuntimeStatusPaused, models.RuntimeStatusPaused, true, "paused→paused (already paused)"},
		{models.RuntimeStatusEnded, models.RuntimeStatusRunning, true, "ended→running (terminal)"},
		{models.RuntimeStatusEnded, models.RuntimeStatusPaused, true, "ended→paused (terminal)"},
		{models.RuntimeStatusEnded, models.RuntimeStatusEnded, true, "ended→ended (terminal)"},
		{models.RuntimeStatusEnded, models.RuntimeStatusPending, true, "ended→pending (terminal)"},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			err := models.ValidateTransition(tc.from, tc.to)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for %q → %q, got nil", tc.from, tc.to)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for %q → %q: %v", tc.from, tc.to, err)
			}
		})
	}
}

func TestStateMachine_AllowedFrom(t *testing.T) {
	cases := []struct {
		from          models.RuntimeStatus
		expectTargets []models.RuntimeStatus
	}{
		{models.RuntimeStatusPending, []models.RuntimeStatus{models.RuntimeStatusRunning}},
		{models.RuntimeStatusRunning, []models.RuntimeStatus{models.RuntimeStatusPaused, models.RuntimeStatusEnded}},
		{models.RuntimeStatusPaused, []models.RuntimeStatus{models.RuntimeStatusRunning, models.RuntimeStatusEnded}},
		{models.RuntimeStatusEnded, []models.RuntimeStatus{}},
	}

	for _, tc := range cases {
		allowed := models.AllowedFrom(tc.from)
		if len(allowed) != len(tc.expectTargets) {
			t.Errorf("AllowedFrom(%q): got %d targets, want %d", tc.from, len(allowed), len(tc.expectTargets))
		}
	}
}

func TestStateMachine_ServiceEnforcesTransitions(t *testing.T) {
	// Verify that service methods properly reject invalid transitions
	// using the existing matchruntime service tests as the integration point.

	// --- pending → ended is forbidden by EndMatch ---
	repo := &stubMatchRuntimeRepo{}
	clock := &testClock{t: fixedTime()}
	svc := newRuntimeSvc(repo, clock)

	matchID := uuid.New()
	// Create a pending state directly (bypass StartMatch).
	repo.state = &models.MatchRuntimeState{
		MatchID: matchID,
		Status:  models.RuntimeStatusPending,
	}

	_, err := svc.EndMatch(ctx(), matchID, adminCtx())
	if err == nil {
		t.Error("EndMatch on pending state should fail — invalid transition pending→ended")
	}
}

func TestStateMachine_DoubleEndForbidden(t *testing.T) {
	repo := &stubMatchRuntimeRepo{}
	clock := &testClock{t: fixedTime()}
	svc := newRuntimeSvc(repo, clock)

	matchID := uuid.New()
	_, _ = svc.StartMatch(ctx(), matchID, adminCtx())
	_, _ = svc.EndMatch(ctx(), matchID, adminCtx())

	_, err := svc.EndMatch(ctx(), matchID, adminCtx())
	if err == nil {
		t.Error("EndMatch on already-ended state should fail — ended is terminal")
	}
}
