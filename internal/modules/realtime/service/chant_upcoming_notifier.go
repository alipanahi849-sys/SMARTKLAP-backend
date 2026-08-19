package service

import (
	"context"
	"encoding/json"
	"time"

	chantdto "clap/internal/modules/chant/dto"
	chantmodels "clap/internal/modules/chant/models"
	realtimedto "clap/internal/modules/realtime/dto"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

const (
	// defaultChantLeadTime is how far before a chant's scheduled start the
	// upcoming notification is emitted.
	defaultChantLeadTime = 2 * time.Minute
	// defaultChantNotifierInterval is how often the DB is polled for chants
	// entering the lead-time window. This is a cheap lookup, not a fan-out.
	defaultChantNotifierInterval = 5 * time.Second
	// lyricsDeadline is how close to start we give up waiting for lyrics and
	// still broadcast the countdown so devices can start on time.
	lyricsDeadline = 15 * time.Second
)

// ChantLyricsProvider builds the full synced-lyrics payload for a chant.
// Implemented by chant/service.ChantService.
type ChantLyricsProvider interface {
	Lyrics(ctx context.Context, chantID uuid.UUID, mode string) (*chantdto.ChantLyricsResponse, error)
}

// upcomingChantSource is the subset of ChantRepository used by the notifier.
type upcomingChantSource interface {
	FindStartingBetween(ctx context.Context, from, to time.Time) ([]chantmodels.Chant, error)
}

// BroadcastPublisher delivers one envelope to every connected client.
// Implemented by WebSocketRealtimeGateway.
type BroadcastPublisher interface {
	BroadcastEnvelope(ctx context.Context, env *realtimedto.EventEnvelope) error
}

// WelcomeSetter stores a snapshot pushed to clients that connect after the
// one-shot broadcast. Implemented by ws.ConnectionManager. Optional.
type WelcomeSetter interface {
	SetWelcomeMessage(data []byte)
}

// ChantCountdownPusher sends a one-shot FCM notification when a chant
// countdown window opens. Implemented by notification/service.NotificationService.
// A nil pusher disables push delivery.
type ChantCountdownPusher interface {
	NotifyChantCountdown(ctx context.Context, chantID, matchID, title string, startsAt time.Time, leadTime time.Duration) error
}

// ChantUpcomingPayload is the body of a chant.upcoming event.
// Personalization (today_points) is intentionally omitted — clients fetch it
// over HTTP so the hot path stays a single identical broadcast.
type ChantUpcomingPayload struct {
	ChantID         string                        `json:"chant_id"`
	MatchID         string                        `json:"match_id"`
	Title           string                        `json:"title"`
	StartsAt        time.Time                     `json:"starts_at"`
	StartsInSeconds int64                         `json:"starts_in_seconds"`
	Lyrics          *chantdto.ChantLyricsResponse `json:"lyrics"`
	TodayPoints     int                           `json:"today_points,omitempty"`
	ChantPoints     int                           `json:"chant_points"`
}

// pendingChant caches per-chant data for the duration of the lead-time window.
type pendingChant struct {
	chant           chantmodels.Chant
	lyrics          *chantdto.ChantLyricsResponse
	eventsScheduled bool
	pushSent        bool
	broadcastSent   bool
}

// ChantUpcomingNotifier watches for active chants approaching their scheduled
// start and broadcasts a single chant.upcoming program to every connected
// client. Late joiners receive the same envelope as a welcome snapshot on
// connect — there is no per-user loop and no 5-second rebroadcast.
//
// The countdown is anchored to the absolute starts_at timestamp, so delivery
// latency never shifts the start moment. A single FCM push is also sent the
// first time a chant enters the window so backgrounded devices still wake.
type ChantUpcomingNotifier struct {
	chantRepo      upcomingChantSource
	lyricsSvc      ChantLyricsProvider
	eventScheduler ChantEventScheduler
	publisher      BroadcastPublisher
	welcome        WelcomeSetter
	pusher         ChantCountdownPusher
	interval       time.Duration
	leadTime       time.Duration

	// pending is touched only by the Run goroutine — no locking needed.
	pending map[uuid.UUID]*pendingChant
	done    chan struct{}
}

// NewChantUpcomingNotifier constructs the notifier.
// Pass 0 for interval/leadTime to use the defaults (5s poll, 2min lead).
// pusher may be nil to disable FCM delivery. welcome may be nil to skip
// late-join snapshots.
func NewChantUpcomingNotifier(
	chantRepo upcomingChantSource,
	lyricsSvc ChantLyricsProvider,
	eventScheduler ChantEventScheduler,
	publisher BroadcastPublisher,
	welcome WelcomeSetter,
	pusher ChantCountdownPusher,
	interval, leadTime time.Duration,
) *ChantUpcomingNotifier {
	if interval <= 0 {
		interval = defaultChantNotifierInterval
	}
	if leadTime <= 0 {
		leadTime = defaultChantLeadTime
	}
	return &ChantUpcomingNotifier{
		chantRepo:      chantRepo,
		lyricsSvc:      lyricsSvc,
		eventScheduler: eventScheduler,
		publisher:      publisher,
		welcome:        welcome,
		pusher:         pusher,
		interval:       interval,
		leadTime:       leadTime,
		pending:        make(map[uuid.UUID]*pendingChant),
		done:           make(chan struct{}),
	}
}

// Done returns a channel closed when Run() exits, used during graceful shutdown.
func (n *ChantUpcomingNotifier) Done() <-chan struct{} { return n.done }

// Run starts the polling loop. It exits when ctx is cancelled.
func (n *ChantUpcomingNotifier) Run(ctx context.Context) {
	defer close(n.done)

	ticker := time.NewTicker(n.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			n.tick(ctx, now.UTC())
		}
	}
}

func (n *ChantUpcomingNotifier) tick(ctx context.Context, now time.Time) {
	chants, err := n.chantRepo.FindStartingBetween(ctx, now, now.Add(n.leadTime))
	if err != nil {
		logger.Error().Err(err).Msg("chant notifier: failed to poll upcoming chants")
		return
	}

	seen := make(map[uuid.UUID]struct{}, len(chants))
	for i := range chants {
		chant := chants[i]
		seen[chant.ID] = struct{}{}
		entry := n.pending[chant.ID]
		if entry == nil {
			entry = &pendingChant{chant: chant}
			n.pending[chant.ID] = entry
			logger.Info().
				Str("chant_id", chant.ID.String()).
				Time("starts_at", chant.ScheduledAt).
				Msg("chant notifier: chant entered notify window")
		}

		n.sendCountdownPush(ctx, entry)

		if entry.lyrics == nil {
			lyrics, lyricsErr := n.lyricsSvc.Lyrics(ctx, chant.ID, "upcoming_notification")
			if lyricsErr != nil {
				logger.Warn().
					Str("chant_id", chant.ID.String()).
					Err(lyricsErr).
					Msg("chant notifier: lyrics unavailable; will retry next tick")
			} else {
				entry.lyrics = lyrics
			}
		}

		if n.eventScheduler != nil && !entry.eventsScheduled {
			if schedErr := n.eventScheduler.ScheduleChantEvents(ctx, chant); schedErr != nil {
				logger.Warn().
					Str("chant_id", chant.ID.String()).
					Err(schedErr).
					Msg("chant notifier: failed to schedule chant events; will retry next tick")
			} else {
				entry.eventsScheduled = true
			}
		}

		n.maybeBroadcast(ctx, entry, now)
	}

	for id, entry := range n.pending {
		if _, ok := seen[id]; ok && !entry.chant.ScheduledAt.Before(now) {
			continue
		}
		delete(n.pending, id)
	}
	n.refreshWelcome()
}

func (n *ChantUpcomingNotifier) sendCountdownPush(ctx context.Context, entry *pendingChant) {
	if n.pusher == nil || entry.pushSent {
		return
	}

	chant := entry.chant
	if err := n.pusher.NotifyChantCountdown(
		ctx,
		chant.ID.String(),
		chant.MatchID.String(),
		chant.Title,
		chant.ScheduledAt,
		n.leadTime,
	); err != nil {
		logger.Warn().
			Str("chant_id", chant.ID.String()).
			Err(err).
			Msg("chant notifier: countdown push failed; will retry next tick")
		return
	}
	entry.pushSent = true
}

func (n *ChantUpcomingNotifier) maybeBroadcast(ctx context.Context, entry *pendingChant, now time.Time) {
	if entry.broadcastSent || n.publisher == nil {
		return
	}

	untilStart := entry.chant.ScheduledAt.Sub(now)
	ready := entry.lyrics != nil || untilStart <= lyricsDeadline
	if !ready {
		return
	}

	n.broadcast(ctx, entry)
}

func (n *ChantUpcomingNotifier) broadcast(ctx context.Context, entry *pendingChant) {
	chant := entry.chant
	startsIn := int64(time.Until(chant.ScheduledAt).Seconds())
	if startsIn < 0 {
		startsIn = 0
	}

	payload := &ChantUpcomingPayload{
		ChantID:         chant.ID.String(),
		MatchID:         chant.MatchID.String(),
		Title:           chant.Title,
		StartsAt:        chant.ScheduledAt,
		StartsInSeconds: startsIn,
		Lyrics:          entry.lyrics,
		ChantPoints:     chant.Points,
	}
	env := realtimedto.NewEnvelope(realtimedto.EventTypeChantUpcoming, &chant.MatchID, payload)

	data, err := json.Marshal(env)
	if err != nil {
		logger.Error().
			Str("chant_id", chant.ID.String()).
			Err(err).
			Msg("chant notifier: failed to marshal upcoming envelope")
		return
	}
	if n.welcome != nil {
		n.welcome.SetWelcomeMessage(data)
	}

	if pubErr := n.publisher.BroadcastEnvelope(ctx, env); pubErr != nil {
		logger.Warn().
			Str("chant_id", chant.ID.String()).
			Err(pubErr).
			Msg("chant notifier: broadcast failed")
		return
	}

	entry.broadcastSent = true
	logger.Info().
		Str("chant_id", chant.ID.String()).
		Int64("starts_in_seconds", startsIn).
		Msg("chant notifier: upcoming chant broadcast")
}

func (n *ChantUpcomingNotifier) refreshWelcome() {
	if n.welcome == nil {
		return
	}
	for _, entry := range n.pending {
		if !entry.broadcastSent {
			continue
		}
		payload := &ChantUpcomingPayload{
			ChantID:         entry.chant.ID.String(),
			MatchID:         entry.chant.MatchID.String(),
			Title:           entry.chant.Title,
			StartsAt:        entry.chant.ScheduledAt,
			StartsInSeconds: int64(time.Until(entry.chant.ScheduledAt).Seconds()),
			Lyrics:          entry.lyrics,
			ChantPoints:     entry.chant.Points,
		}
		env := realtimedto.NewEnvelope(realtimedto.EventTypeChantUpcoming, &entry.chant.MatchID, payload)
		if data, err := json.Marshal(env); err == nil {
			n.welcome.SetWelcomeMessage(data)
			return
		}
	}
	n.welcome.SetWelcomeMessage(nil)
}
