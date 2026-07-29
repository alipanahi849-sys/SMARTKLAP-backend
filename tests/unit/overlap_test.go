package unit_test

import (
	"context"
	"testing"
	"time"

	playbackmodels "clap/internal/modules/playback/models"
	playbackrepo "clap/internal/modules/playback/repository"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── Stub playback repository (in-memory overlap detection) ───────────────────

type stubPlaybackRepo struct {
	schedules []*playbackmodels.PlaybackSchedule
}

func (r *stubPlaybackRepo) Create(_ context.Context, s *playbackmodels.PlaybackSchedule) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	r.schedules = append(r.schedules, s)
	return nil
}

func (r *stubPlaybackRepo) FindByID(_ context.Context, id uuid.UUID) (*playbackmodels.PlaybackSchedule, error) {
	for _, s := range r.schedules {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, sharederrors.NewNotFound("not found", nil)
}

func (r *stubPlaybackRepo) FindByMatchID(_ context.Context, matchID uuid.UUID) ([]*playbackmodels.PlaybackSchedule, error) {
	var out []*playbackmodels.PlaybackSchedule
	for _, s := range r.schedules {
		if s.MatchID == matchID && s.Status != playbackmodels.PlaybackStatusCancelled {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *stubPlaybackRepo) FindUpcoming(_ context.Context, matchID uuid.UUID, after time.Time) ([]*playbackmodels.PlaybackSchedule, error) {
	var out []*playbackmodels.PlaybackSchedule
	for _, s := range r.schedules {
		if s.MatchID == matchID && s.Status == playbackmodels.PlaybackStatusPending && s.ScheduledAt.After(after) {
			out = append(out, s)
		}
	}
	return out, nil
}

// HasOverlap implements Allen's interval overlap check in memory.
func (r *stubPlaybackRepo) HasOverlap(_ context.Context, matchID uuid.UUID, start time.Time, durationMs int64, excludeID *uuid.UUID) (bool, error) {
	if durationMs <= 0 {
		return false, nil
	}
	newEnd := start.UnixMilli() + durationMs

	for _, s := range r.schedules {
		if s.MatchID != matchID {
			continue
		}
		if s.Status == playbackmodels.PlaybackStatusCancelled {
			continue
		}
		if s.DurationMs <= 0 {
			continue
		}
		if excludeID != nil && s.ID == *excludeID {
			continue
		}
		existEnd := s.ScheduledAt.UnixMilli() + s.DurationMs
		// Overlap: existing.start < new.end  AND  existing.end > new.start
		if s.ScheduledAt.UnixMilli() < newEnd && existEnd > start.UnixMilli() {
			return true, nil
		}
	}
	return false, nil
}

func (r *stubPlaybackRepo) UpdateStatus(_ context.Context, id uuid.UUID, status playbackmodels.PlaybackStatus) error {
	for _, s := range r.schedules {
		if s.ID == id {
			s.Status = status
			return nil
		}
	}
	return sharederrors.NewNotFound("not found", nil)
}

func (r *stubPlaybackRepo) Update(_ context.Context, s *playbackmodels.PlaybackSchedule) error {
	for i, existing := range r.schedules {
		if existing.ID == s.ID {
			r.schedules[i] = s
			return nil
		}
	}
	return sharederrors.NewNotFound("not found", nil)
}

var _ playbackrepo.PlaybackRepository = (*stubPlaybackRepo)(nil)

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestOverlap_NoConflictWhenEmpty(t *testing.T) {
	repo := &stubPlaybackRepo{}
	matchID := uuid.New()
	start := time.Now().Add(time.Hour)

	overlaps, err := repo.HasOverlap(context.Background(), matchID, start, 30_000, nil)
	if err != nil {
		t.Fatalf("HasOverlap error: %v", err)
	}
	if overlaps {
		t.Error("expected no overlap for empty schedule")
	}
}

func TestOverlap_DirectOverlapDetected(t *testing.T) {
	repo := &stubPlaybackRepo{}
	matchID := uuid.New()

	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

	// Song A: 10:00 → 10:30 (30 min = 1_800_000 ms)
	repo.schedules = append(repo.schedules, &playbackmodels.PlaybackSchedule{
		ID:          uuid.New(),
		MatchID:     matchID,
		SongID:      uuid.New(),
		ScheduledAt: base,
		DurationMs:  30 * 60 * 1000,
		Status:      playbackmodels.PlaybackStatusPending,
	})

	// Song B tries: 10:15 → 10:45 — overlaps with A.
	songBStart := base.Add(15 * time.Minute)
	overlaps, err := repo.HasOverlap(context.Background(), matchID, songBStart, 30*60*1000, nil)
	if err != nil {
		t.Fatalf("HasOverlap error: %v", err)
	}
	if !overlaps {
		t.Error("expected overlap detected for 10:15–10:45 against existing 10:00–10:30")
	}
}

func TestOverlap_AdjacentWindowsAllowed(t *testing.T) {
	repo := &stubPlaybackRepo{}
	matchID := uuid.New()

	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

	// Song A: 10:00 → 10:30.
	repo.schedules = append(repo.schedules, &playbackmodels.PlaybackSchedule{
		ID:          uuid.New(),
		MatchID:     matchID,
		SongID:      uuid.New(),
		ScheduledAt: base,
		DurationMs:  30 * 60 * 1000,
		Status:      playbackmodels.PlaybackStatusPending,
	})

	// Song B tries: 10:30 → 11:00 — starts exactly when A ends (no overlap).
	songBStart := base.Add(30 * time.Minute)
	overlaps, err := repo.HasOverlap(context.Background(), matchID, songBStart, 30*60*1000, nil)
	if err != nil {
		t.Fatalf("HasOverlap error: %v", err)
	}
	if overlaps {
		t.Error("expected no overlap for adjacent windows 10:00–10:30 and 10:30–11:00")
	}
}

func TestOverlap_CancelledScheduleNotChecked(t *testing.T) {
	repo := &stubPlaybackRepo{}
	matchID := uuid.New()

	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

	// Song A is cancelled.
	repo.schedules = append(repo.schedules, &playbackmodels.PlaybackSchedule{
		ID:          uuid.New(),
		MatchID:     matchID,
		SongID:      uuid.New(),
		ScheduledAt: base,
		DurationMs:  30 * 60 * 1000,
		Status:      playbackmodels.PlaybackStatusCancelled, // cancelled — ignore
	})

	// Song B overlaps in time — but A is cancelled, so no conflict.
	overlaps, err := repo.HasOverlap(context.Background(), matchID, base.Add(15*time.Minute), 30*60*1000, nil)
	if err != nil {
		t.Fatalf("HasOverlap error: %v", err)
	}
	if overlaps {
		t.Error("cancelled schedule should not block new overlapping window")
	}
}

func TestOverlap_NoDurationSkipsCheck(t *testing.T) {
	repo := &stubPlaybackRepo{}
	matchID := uuid.New()

	// New request with duration=0 → check is skipped entirely.
	overlaps, err := repo.HasOverlap(context.Background(), matchID, time.Now(), 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overlaps {
		t.Error("zero-duration schedule should never overlap")
	}
}

func TestOverlap_ExcludeIDIgnoresSelf(t *testing.T) {
	repo := &stubPlaybackRepo{}
	matchID := uuid.New()

	base := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	selfID := uuid.New()

	repo.schedules = append(repo.schedules, &playbackmodels.PlaybackSchedule{
		ID:          selfID,
		MatchID:     matchID,
		SongID:      uuid.New(),
		ScheduledAt: base,
		DurationMs:  30 * 60 * 1000,
		Status:      playbackmodels.PlaybackStatusPending,
	})

	// Same window, excluding the self record (update scenario).
	overlaps, err := repo.HasOverlap(context.Background(), matchID, base, 30*60*1000, &selfID)
	if err != nil {
		t.Fatalf("HasOverlap error: %v", err)
	}
	if overlaps {
		t.Error("self-exclusion should prevent self-conflict during update")
	}
}
