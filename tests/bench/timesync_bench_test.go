package bench_test

import (
	"testing"
	"time"
)

// BenchmarkDrift_Calculate measures the cost of the drift calculation.
// This must stay sub-microsecond — it runs once per heartbeat.
func BenchmarkDrift_Calculate(b *testing.B) {
	serverTime := time.Now().UTC()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clientTime := serverTime.Add(-47 * time.Millisecond)
		_ = calculateDrift(serverTime, clientTime)
	}
}

// BenchmarkDrift_BuildSyncPayload measures JSON-payload construction.
func BenchmarkDrift_BuildSyncPayload(b *testing.B) {
	serverTime := time.Now().UTC()
	clientTime := serverTime.Add(-23 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildSyncPayload(serverTime, clientTime)
	}
}

// ─── Pure-function implementations for benchmarking without DB deps ───────────

type syncPayload struct {
	ServerTimeMs  int64 `json:"server_time_ms"`
	ClientTimeMs  int64 `json:"client_time_ms"`
	DriftMs       int64 `json:"drift_ms"`
	RecommendedMs int64 `json:"recommended_adjust_ms"`
}

func calculateDrift(serverTime, clientTime time.Time) int64 {
	return serverTime.UnixMilli() - clientTime.UnixMilli()
}

func buildSyncPayload(serverTime, clientTime time.Time) syncPayload {
	drift := calculateDrift(serverTime, clientTime)
	return syncPayload{
		ServerTimeMs:  serverTime.UnixMilli(),
		ClientTimeMs:  clientTime.UnixMilli(),
		DriftMs:       drift,
		RecommendedMs: -drift / 2,
	}
}
