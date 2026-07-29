package bench_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	schedulersvc "clap/internal/modules/eventscheduler/service"

	"github.com/google/uuid"
)

// BenchmarkScheduler_Enqueue measures the cost of inserting events into the
// in-memory priority queue under sequential and parallel load.
func BenchmarkScheduler_Enqueue(b *testing.B) {
	sched := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	ctx := context.Background()
	base := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := &schedulersvc.SchedulerItem{
			ID:          fmt.Sprintf("evt-%d", i),
			SessionID:   uuid.New().String(),
			EventType:   "timer_sync",
			ExecuteAt:   base.Add(time.Duration(i) * time.Millisecond),
			PayloadJSON: `{"tick":true}`,
		}
		if err := sched.RegisterEvent(ctx, item); err != nil {
			b.Fatalf("RegisterEvent failed: %v", err)
		}
	}
}

// BenchmarkScheduler_Dequeue measures reading due events without removal.
func BenchmarkScheduler_Dequeue(b *testing.B) {
	sched := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	ctx := context.Background()
	now := time.Now()

	// Pre-load 10 000 events, half due, half in the future.
	for i := 0; i < 10_000; i++ {
		var executeAt time.Time
		if i%2 == 0 {
			executeAt = now.Add(-time.Duration(i) * time.Millisecond) // already due
		} else {
			executeAt = now.Add(time.Duration(i) * time.Hour) // far future
		}
		_ = sched.RegisterEvent(ctx, &schedulersvc.SchedulerItem{
			ID:        fmt.Sprintf("e-%d", i),
			EventType: "song_start",
			ExecuteAt: executeAt,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sched.GetPendingEvents(ctx, now); err != nil {
			b.Fatalf("GetPendingEvents failed: %v", err)
		}
	}
}

// BenchmarkScheduler_Cancel measures the cost of removing events from the heap.
func BenchmarkScheduler_Cancel(b *testing.B) {
	ctx := context.Background()
	sched := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())

	ids := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		ids[i] = fmt.Sprintf("cancel-%d", i)
		_ = sched.RegisterEvent(ctx, &schedulersvc.SchedulerItem{
			ID:        ids[i],
			EventType: "vibrate",
			ExecuteAt: time.Now().Add(time.Hour),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := sched.CancelEvent(ctx, ids[i]); err != nil {
			b.Fatalf("CancelEvent failed: %v", err)
		}
	}
}

// BenchmarkScheduler_EnqueueParallel measures concurrent enqueue throughput.
func BenchmarkScheduler_EnqueueParallel(b *testing.B) {
	sched := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	ctx := context.Background()
	base := time.Now()
	counter := 0

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := uuid.New().String()
			_ = sched.RegisterEvent(ctx, &schedulersvc.SchedulerItem{
				ID:        id,
				EventType: "lyric_sync",
				ExecuteAt: base.Add(time.Duration(counter) * time.Microsecond),
			})
			counter++
		}
	})
}
