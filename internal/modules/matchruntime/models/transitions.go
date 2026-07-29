package models

import (
	"fmt"

	sharederrors "clap/internal/shared/errors"
)

// allowedTransitions is the canonical state-transition table for MatchRuntimeState.
//
//	Matrix (from → to):
//	  pending  → running   ✓  (StartMatch)
//	  running  → paused    ✓  (PauseMatch)
//	  paused   → running   ✓  (ResumeMatch)
//	  running  → ended     ✓  (EndMatch)
//	  paused   → ended     ✓  (EndMatch)
//
//	Explicitly forbidden:
//	  pending  → ended     ✗  (must start before ending)
//	  ended    → *         ✗  (terminal state)
//	  running  → running   ✗  (already running)
//	  paused   → paused    ✗  (already paused)
var allowedTransitions = map[RuntimeStatus]map[RuntimeStatus]bool{
	RuntimeStatusPending: {
		RuntimeStatusRunning: true,
	},
	RuntimeStatusRunning: {
		RuntimeStatusPaused: true,
		RuntimeStatusEnded:  true,
	},
	RuntimeStatusPaused: {
		RuntimeStatusRunning: true,
		RuntimeStatusEnded:   true,
	},
	RuntimeStatusEnded: {}, // terminal — no transitions permitted
}

// ValidateTransition returns a domain error if the requested state change is forbidden.
// Callers MUST invoke this before any state mutation.
func ValidateTransition(from, to RuntimeStatus) error {
	targets, ok := allowedTransitions[from]
	if !ok {
		return sharederrors.NewBadRequest(
			fmt.Sprintf("unknown runtime status: %q", from), nil,
		)
	}
	if !targets[to] {
		return sharederrors.NewBadRequest(
			fmt.Sprintf("invalid state transition: %q → %q", from, to), nil,
		)
	}
	return nil
}

// AllowedFrom returns the set of states that can be reached from a given status.
// Useful for producing descriptive error messages and documentation.
func AllowedFrom(from RuntimeStatus) []RuntimeStatus {
	targets, ok := allowedTransitions[from]
	if !ok {
		return nil
	}
	out := make([]RuntimeStatus, 0, len(targets))
	for s, allowed := range targets {
		if allowed {
			out = append(out, s)
		}
	}
	return out
}
