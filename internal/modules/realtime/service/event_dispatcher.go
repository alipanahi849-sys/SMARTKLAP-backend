package service

import (
	"context"
	"encoding/json"
	"time"

	schedulerrepo "clap/internal/modules/eventscheduler/repository"
	schedulersvc "clap/internal/modules/eventscheduler/service"
	realtimedto "clap/internal/modules/realtime/dto"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

// DispatchGateway is the transport-agnostic delivery interface used by the
// EventDispatcher.  The production implementation is WebSocketRealtimeGateway.
type DispatchGateway interface {
	PublishMatchEvent(ctx context.Context, matchID uuid.UUID, eventType string, payload any) error
}

// dispatchPayload is the expected structure stored in SchedulerEvent.PayloadJSON
// for events dispatched through the realtime gateway.
type dispatchPayload struct {
	MatchID     string `json:"match_id"`
	SongID      string `json:"song_id,omitempty"`
	ScheduleID  string `json:"schedule_id,omitempty"`
	EventType   string `json:"event_type"`
	Line        string `json:"line,omitempty"`
	Index       int    `json:"index,omitempty"`
	TimestampMs int64  `json:"timestamp_ms,omitempty"`
}

// EventDispatcher is a background service that polls the in-memory scheduler
// for due events and publishes them via the realtime gateway.
//
// It provides "mark executed only after successful publish": the DB row is
// marked executed only after the gateway confirms delivery. Combined with the
// stale-processing watchdog, this yields at-least-once delivery across crashes.
//
// Lifecycle:
//
//	d := NewEventDispatcher(...)
//	go d.Run(ctx)   // blocks until ctx is cancelled
//	<-d.Done()      // wait for clean exit during shutdown
type EventDispatcher struct {
	scheduler    schedulersvc.EventScheduler
	eventRepo    schedulerrepo.SchedulerEventRepository
	gateway      DispatchGateway
	tickInterval time.Duration
	done         chan struct{}
}

const defaultDispatchInterval = 100 * time.Millisecond

// NewEventDispatcher constructs the dispatcher.
// tickInterval controls how often the queue is polled; use 0 for the default (100 ms).
func NewEventDispatcher(
	scheduler schedulersvc.EventScheduler,
	eventRepo schedulerrepo.SchedulerEventRepository,
	gateway DispatchGateway,
	tickInterval time.Duration,
) *EventDispatcher {
	if tickInterval <= 0 {
		tickInterval = defaultDispatchInterval
	}
	return &EventDispatcher{
		scheduler:    scheduler,
		eventRepo:    eventRepo,
		gateway:      gateway,
		tickInterval: tickInterval,
		done:         make(chan struct{}),
	}
}

// Done returns a channel closed when Run() exits, used during graceful shutdown.
func (d *EventDispatcher) Done() <-chan struct{} { return d.done }

// Run starts the dispatch loop.  It exits when ctx is cancelled.
func (d *EventDispatcher) Run(ctx context.Context) {
	defer close(d.done)

	ticker := time.NewTicker(d.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			d.dispatchDue(ctx, now)
		}
	}
}

func (d *EventDispatcher) dispatchDue(ctx context.Context, now time.Time) {
	due, err := d.scheduler.GetPendingEvents(ctx, now)
	if err != nil {
		logger.Error().Err(err).Msg("event dispatcher: failed to poll scheduler")
		return
	}

	for _, item := range due {
		// Stop promptly on shutdown so in-flight work is bounded.
		select {
		case <-ctx.Done():
			return
		default:
		}
		d.process(ctx, item)
	}
}

func (d *EventDispatcher) process(ctx context.Context, item *schedulersvc.SchedulerItem) {
	itemID, err := uuid.Parse(item.ID)
	if err != nil {
		logger.Error().Str("item_id", item.ID).Msg("event dispatcher: invalid item UUID; dropping from queue")
		_ = d.scheduler.CancelEvent(ctx, item.ID)
		return
	}

	// Atomically claim the event from the DB (prevents double-execution).
	ev, err := d.eventRepo.ClaimForProcessing(ctx, itemID)
	if err != nil {
		// If the row is simply not claimable (already executed, cancelled, or
		// claimed by another instance), drop the now-stale in-memory entry to
		// avoid re-polling it every tick (zombie heap entry). Transient DB
		// errors are left in place so the event can be retried.
		if isNotFound(err) {
			_ = d.scheduler.CancelEvent(ctx, item.ID)
		}
		return
	}

	// Parse the payload stored at scheduling time.
	var p dispatchPayload
	if jsonErr := json.Unmarshal([]byte(ev.PayloadJSON), &p); jsonErr != nil {
		reason := "invalid payload JSON: " + jsonErr.Error()
		_ = d.eventRepo.MarkFailed(ctx, itemID, reason)
		_ = d.scheduler.CancelEvent(ctx, item.ID)
		logger.Error().
			Str("event_id", itemID.String()).
			Str("session_id", item.SessionID).
			Str("event_type", ev.EventType).
			Err(jsonErr).
			Msg("event dispatcher: payload parse error")
		return
	}

	matchID, err := uuid.Parse(p.MatchID)
	if err != nil {
		reason := "invalid match_id in payload: " + err.Error()
		_ = d.eventRepo.MarkFailed(ctx, itemID, reason)
		_ = d.scheduler.CancelEvent(ctx, item.ID)
		return
	}

	// Determine the effective event type and build the client payload.
	eventType := ev.EventType
	if p.EventType != "" {
		eventType = p.EventType
	}
	clientPayload := d.buildClientPayload(eventType, &p)

	publishErr := d.gateway.PublishMatchEvent(ctx, matchID, eventType, clientPayload)
	if publishErr != nil {
		reason := "gateway publish failed: " + publishErr.Error()
		_ = d.eventRepo.MarkFailed(ctx, itemID, reason)
		_ = d.scheduler.CancelEvent(ctx, item.ID)
		logger.Error().
			Str("event_id", itemID.String()).
			Str("event_type", eventType).
			Str("match_id", matchID.String()).
			Err(publishErr).
			Msg("event dispatcher: publish failed")
		return
	}

	// Mark executed only after confirmed delivery.
	if markErr := d.eventRepo.MarkExecuted(ctx, itemID); markErr != nil {
		// Delivery succeeded but the executed-mark failed. Leave the in-memory
		// entry so it is retried; the watchdog will also reclaim it if this
		// process dies. At-least-once semantics may produce a duplicate.
		logger.Warn().
			Str("event_id", itemID.String()).
			Str("event_type", eventType).
			Str("match_id", matchID.String()).
			Err(markErr).
			Msg("event dispatcher: MarkExecuted failed after successful publish")
		return
	}

	// Remove from in-memory queue to avoid re-dispatch.
	_ = d.scheduler.CancelEvent(ctx, item.ID)

	logger.Info().
		Str("event_id", itemID.String()).
		Str("event_type", eventType).
		Str("match_id", matchID.String()).
		Msg("event dispatcher: event dispatched")
}

func (d *EventDispatcher) buildClientPayload(eventType string, p *dispatchPayload) map[string]any { //nolint:unparam
	switch eventType {
	case realtimedto.EventTypeSongPlaybackStarted, "song_start":
		return map[string]any{
			"schedule_id": p.ScheduleID,
			"song_id":     p.SongID,
			"match_id":    p.MatchID,
		}
	case realtimedto.EventTypeLyricsLineChanged, "lyric_sync":
		return map[string]any{
			"line":         p.Line,
			"index":        p.Index,
			"timestamp_ms": p.TimestampMs,
			"match_id":     p.MatchID,
			"song_id":      p.SongID,
		}
	default:
		return map[string]any{
			"event_type":  eventType,
			"match_id":    p.MatchID,
			"song_id":     p.SongID,
			"schedule_id": p.ScheduleID,
		}
	}
}
