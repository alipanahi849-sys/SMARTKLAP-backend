package bench_test

import (
	"testing"
	"time"
)

// BenchmarkTimer_ComputeElapsed measures the cost of re-computing elapsed match time
// from first principles. This runs on every GET /matches/:id/runtime call.
func BenchmarkTimer_ComputeElapsed_Running(b *testing.B) {
	startedAt := time.Now().UTC().Add(-90 * time.Minute)
	totalPausedMs := int64(5 * 60 * 1000) // 5 minutes of pauses

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		now := time.Now().UTC()
		_ = computeElapsed(now, &startedAt, nil, totalPausedMs, "running")
	}
}

// BenchmarkTimer_ComputeElapsed_Paused measures the elapsed calculation for a
// currently-paused match.
func BenchmarkTimer_ComputeElapsed_Paused(b *testing.B) {
	startedAt := time.Now().UTC().Add(-60 * time.Minute)
	pausedAt := time.Now().UTC().Add(-10 * time.Minute)
	totalPausedMs := int64(2 * 60 * 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		now := time.Now().UTC()
		_ = computeElapsed(now, &startedAt, &pausedAt, totalPausedMs, "paused")
	}
}

// BenchmarkTimer_ComputeElapsed_Ended measures the calculation for a finished match.
func BenchmarkTimer_ComputeElapsed_Ended(b *testing.B) {
	startedAt := time.Now().UTC().Add(-90 * time.Minute)
	endedAt := time.Now().UTC().Add(-1 * time.Minute)
	totalPausedMs := int64(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		now := time.Now().UTC()
		_ = computeElapsed(now, &startedAt, &endedAt, totalPausedMs, "ended")
	}
}

// ─── Pure computation (mirrors matchRuntimeService.computeElapsed logic) ─────

func computeElapsed(now time.Time, startedAt, reference *time.Time, totalPausedMs int64, status string) int64 {
	if startedAt == nil {
		return 0
	}
	var elapsed int64
	switch status {
	case "running":
		elapsed = now.Sub(*startedAt).Milliseconds() - totalPausedMs
	case "paused":
		if reference != nil {
			elapsed = reference.Sub(*startedAt).Milliseconds() - totalPausedMs
		}
	case "ended":
		if reference != nil {
			elapsed = reference.Sub(*startedAt).Milliseconds() - totalPausedMs
		}
	}
	if elapsed < 0 {
		elapsed = 0
	}
	return elapsed
}
