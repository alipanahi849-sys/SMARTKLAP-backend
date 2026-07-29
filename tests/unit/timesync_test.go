package unit_test

import (
	"context"
	"testing"
	"time"

	realtimeservice "clap/internal/modules/realtime/service"
)

// mockClock is a test clock returning a fixed time.
type mockClock struct{ fixed time.Time }

func (m mockClock) Now() time.Time { return m.fixed }

func TestTimeSyncService_GetServerTime(t *testing.T) {
	fixed := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	svc := realtimeservice.NewTimeSyncService(mockClock{fixed: fixed})

	result := svc.GetServerTime(context.Background())

	if result.ServerTimeMs != fixed.UnixMilli() {
		t.Errorf("expected server_time_ms=%d, got %d", fixed.UnixMilli(), result.ServerTimeMs)
	}
	if result.ServerTimeISO == "" {
		t.Error("server_time_iso must not be empty")
	}
}

func TestTimeSyncService_CalculateDrift_PositiveClientBehind(t *testing.T) {
	svc := realtimeservice.NewTimeSyncService(mockClock{})

	// Client is 200ms behind server.
	clientMs := int64(1000)
	serverMs := int64(1200)

	d := svc.CalculateDrift(clientMs, serverMs)

	if d.DriftMs != 200 {
		t.Errorf("expected drift=200, got %d", d.DriftMs)
	}
	if d.AbsDriftMs != 200 {
		t.Errorf("expected abs_drift=200, got %d", d.AbsDriftMs)
	}
	if d.IsSignificant {
		t.Error("drift of 200ms should not be significant (threshold=500)")
	}
}

func TestTimeSyncService_CalculateDrift_NegativeClientAhead(t *testing.T) {
	svc := realtimeservice.NewTimeSyncService(mockClock{})

	// Client is 300ms ahead of server.
	clientMs := int64(1300)
	serverMs := int64(1000)

	d := svc.CalculateDrift(clientMs, serverMs)

	if d.DriftMs != -300 {
		t.Errorf("expected drift=-300, got %d", d.DriftMs)
	}
	if d.AbsDriftMs != 300 {
		t.Errorf("expected abs_drift=300, got %d", d.AbsDriftMs)
	}
}

func TestTimeSyncService_CalculateDrift_Significant(t *testing.T) {
	svc := realtimeservice.NewTimeSyncService(mockClock{})

	d := svc.CalculateDrift(0, 1000) // 1 second drift

	if !d.IsSignificant {
		t.Error("drift of 1000ms should be significant (threshold=500)")
	}
}

func TestTimeSyncService_CalculateDrift_ZeroDrift(t *testing.T) {
	svc := realtimeservice.NewTimeSyncService(mockClock{})

	d := svc.CalculateDrift(1000, 1000)

	if d.DriftMs != 0 {
		t.Errorf("expected zero drift, got %d", d.DriftMs)
	}
	if d.IsSignificant {
		t.Error("zero drift should not be significant")
	}
}

func TestTimeSyncService_BuildSyncPayload(t *testing.T) {
	fixed := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	svc := realtimeservice.NewTimeSyncService(mockClock{fixed: fixed})

	clientMs := fixed.UnixMilli() - 250 // client is 250ms behind
	result := svc.BuildSyncPayload(context.Background(), clientMs)

	if result.ServerTimeMs != fixed.UnixMilli() {
		t.Errorf("expected server_time_ms=%d, got %d", fixed.UnixMilli(), result.ServerTimeMs)
	}
	if result.DriftMs != 250 {
		t.Errorf("expected drift_ms=250, got %d", result.DriftMs)
	}
	if result.CorrectedClientMs != fixed.UnixMilli() {
		t.Errorf("expected corrected_client_ms=%d, got %d", fixed.UnixMilli(), result.CorrectedClientMs)
	}
	if result.IsSignificant {
		t.Error("drift of 250ms should not be significant")
	}
}
