package service

import (
	"context"
	"time"

	"clap/internal/modules/realtime/dto"
)

// significantDriftThresholdMs defines the minimum drift that should be
// reported to the client as "significant" and trigger correction logic.
const significantDriftThresholdMs = 500

// Clock abstracts time.Now() to make TimeSyncService fully testable
// without sleeps or real-time dependencies.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// SystemClock returns the production clock backed by time.Now().
func SystemClock() Clock { return systemClock{} }

// TimeSyncService provides server time and drift calculation utilities.
// It has no storage dependency — all operations are pure calculations.
type TimeSyncService interface {
	GetServerTime(ctx context.Context) *dto.ServerTimeResponse
	CalculateDrift(clientTimestampMs, serverTimestampMs int64) *dto.DriftResponse
	BuildSyncPayload(ctx context.Context, clientTimestampMs int64) *dto.TimeSyncResponse
}

type timeSyncService struct {
	clock Clock
}

func NewTimeSyncService(clock Clock) TimeSyncService {
	return &timeSyncService{clock: clock}
}

func (s *timeSyncService) GetServerTime(_ context.Context) *dto.ServerTimeResponse {
	now := s.clock.Now()
	return &dto.ServerTimeResponse{
		ServerTimeMs:  now.UnixMilli(),
		ServerTimeISO: now.Format(time.RFC3339Nano),
	}
}

func (s *timeSyncService) CalculateDrift(clientTimestampMs, serverTimestampMs int64) *dto.DriftResponse {
	drift := serverTimestampMs - clientTimestampMs
	abs := drift
	if abs < 0 {
		abs = -abs
	}
	return &dto.DriftResponse{
		ClientTimestampMs: clientTimestampMs,
		ServerTimestampMs: serverTimestampMs,
		DriftMs:           drift,
		AbsDriftMs:        abs,
		IsSignificant:     abs > significantDriftThresholdMs,
	}
}

func (s *timeSyncService) BuildSyncPayload(ctx context.Context, clientTimestampMs int64) *dto.TimeSyncResponse {
	st := s.GetServerTime(ctx)
	d := s.CalculateDrift(clientTimestampMs, st.ServerTimeMs)
	return &dto.TimeSyncResponse{
		ServerTimeMs:      st.ServerTimeMs,
		ServerTimeISO:     st.ServerTimeISO,
		ClientTimestampMs: clientTimestampMs,
		DriftMs:           d.DriftMs,
		CorrectedClientMs: clientTimestampMs + d.DriftMs,
		IsSignificant:     d.IsSignificant,
	}
}
