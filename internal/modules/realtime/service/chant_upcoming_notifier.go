package service

import (
	"context"
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
	// defaultChantNotifierInterval is how often the DB is polled and the
	// upcoming event is re-broadcast. Kept short so a device that connects
	// (or reconnects) mid-window joins the countdown almost immediately.
	defaultChantNotifierInterval = 5 * time.Second
)

// ChantLyricsProvider builds the full synced-lyrics payload for a chant.
// Implemented by chant/service.ChantService.
type ChantLyricsProvider interface {
	Lyrics(ctx context.Context, chantID uuid.UUID, mode string) (*chantdto.ChantLyricsResponse, error)
}

// upcomingChantSource is the subset of ChantRepository used by the notifier.
type upcomingChantSource interface {
	FindStartingBetween(ctx context.Context, from, to time.Time) ([]chantmodels.Chant, error)
	TodayPoints(ctx context.Context, userID uuid.UUID) (int, error)
}

// UserEnvelopePublisher delivers an EventEnvelope to all connections of one
// user. Implemented by WebSocketRealtimeGateway.
type UserEnvelopePublisher interface {
	PublishToUser(ctx context.Context, userID uuid.UUID, env *realtimedto.EventEnvelope) error
}

// ChantCountdownPusher sends a one-shot FCM notification when a chant
// countdown window opens. Implemented by notification/service.NotificationService.
// A nil pusher disables push delivery.
type ChantCountdownPusher interface {
	NotifyChantCountdown(ctx context.Context, chantID, matchID, title string, startsAt time.Time, leadTime time.Duration) error
}

// ConnectedUserLister returns the distinct user IDs of all connected clients.
// Implemented by ws.ConnectionManager.
type ConnectedUserLister interface {
	ConnectedUserIDs(ctx context.Context) ([]uuid.UUID, error)
}

// ChantUpcomingPayload is the body of a chant.upcoming event.
// Field order follows the contract: lyrics first, then the user's points
// earned today, then the chant's own points.
type ChantUpcomingPayload struct {
	ChantID         string                        `json:"chant_id"`
	MatchID         string                        `json:"match_id"`
	Title           string                        `json:"title"`
	StartsAt        time.Time                     `json:"starts_at"`
	StartsInSeconds int64                         `json:"starts_in_seconds"`
	Lyrics          *chantdto.ChantLyricsResponse `json:"lyrics"`
	TodayPoints     int                           `json:"today_points"`
	ChantPoints     int                           `json:"chant_points"`
}

// pendingChant caches per-chant data (lyrics) for the duration of the
// lead-time window so they are loaded from the DB only once.
type pendingChant struct {
	chant           chantmodels.Chant
	lyrics          *chantdto.ChantLyricsResponse
	eventsScheduled bool
	pushSent        bool
}

// ChantUpcomingNotifier is a background service that watches for active chants
// approaching their scheduled start and pushes a personalised chant.upcoming
// event to every connected user, regardless of channel subscriptions — the
// mobile app must surface the countdown no matter which screen is open.
//
// The event is deliberately re-broadcast on every tick for the whole window:
// devices that connect or reconnect mid-window still receive it (at most one
// interval late), and multiple devices on the same account all get their own
// copy. The app deduplicates by chant_id and simply refreshes the countdown
// data on repeats. Because the countdown is anchored to the absolute starts_at
// timestamp, late delivery never shifts the actual start moment.
//
// A single FCM push is also sent the first time a chant enters the window so
// backgrounded / killed devices still see the song name and the 2-minute warning.
//
// Lifecycle mirrors EventDispatcher:
//
//	n := NewChantUpcomingNotifier(...)
//	go n.Run(ctx)   // blocks until ctx is cancelled
//	<-n.Done()
type ChantUpcomingNotifier struct {
	chantRepo      upcomingChantSource
	lyricsSvc      ChantLyricsProvider
	eventScheduler ChantEventScheduler
	users          ConnectedUserLister
	publisher      UserEnvelopePublisher
	pusher         ChantCountdownPusher
	interval       time.Duration
	leadTime       time.Duration

	// pending is touched only by the Run goroutine — no locking needed.
	pending map[uuid.UUID]*pendingChant
	done    chan struct{}
}

// NewChantUpcomingNotifier constructs the notifier.
// Pass 0 for interval/leadTime to use the defaults (5s poll, 2min lead).
// pusher may be nil to disable FCM delivery.
func NewChantUpcomingNotifier(
	chantRepo upcomingChantSource,
	lyricsSvc ChantLyricsProvider,
	eventScheduler ChantEventScheduler,
	users ConnectedUserLister,
	publisher UserEnvelopePublisher,
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
		users:          users,
		publisher:      publisher,
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

	for i := range chants {
		chant := chants[i]
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

		// Load lyrics once per chant; retry on later ticks if it failed.
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

		n.notifyUsers(ctx, entry)
	}

	// Drop chants whose start time has passed.
	for id, entry := range n.pending {
		if entry.chant.ScheduledAt.Before(now) {
			delete(n.pending, id)
		}
	}
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

// notifyUsers re-broadcasts the event to every connected user.
// Repeats are intentional (see the type comment); per-user failures are
// logged and simply retried on the next tick.
func (n *ChantUpcomingNotifier) notifyUsers(ctx context.Context, entry *pendingChant) {
	chant := entry.chant

	userIDs, err := n.users.ConnectedUserIDs(ctx)
	if err != nil {
		logger.Warn().
			Str("chant_id", chant.ID.String()).
			Err(err).
			Msg("chant notifier: failed to list connected users")
		return
	}

	startsIn := int64(time.Until(chant.ScheduledAt).Seconds())
	if startsIn < 0 {
		startsIn = 0
	}

	notified := 0
	for _, userID := range userIDs {
		todayPoints, pointsErr := n.chantRepo.TodayPoints(ctx, userID)
		if pointsErr != nil {
			logger.Warn().
				Str("chant_id", chant.ID.String()).
				Str("user_id", userID.String()).
				Err(pointsErr).
				Msg("chant notifier: failed to load today's points")
			continue
		}

		payload := &ChantUpcomingPayload{
			ChantID:         chant.ID.String(),
			MatchID:         chant.MatchID.String(),
			Title:           chant.Title,
			StartsAt:        chant.ScheduledAt,
			StartsInSeconds: startsIn,
			Lyrics:          entry.lyrics,
			TodayPoints:     todayPoints,
			ChantPoints:     chant.Points,
		}
		env := realtimedto.NewEnvelope(realtimedto.EventTypeChantUpcoming, &chant.MatchID, payload)

		if pubErr := n.publisher.PublishToUser(ctx, userID, env); pubErr != nil {
			logger.Warn().
				Str("chant_id", chant.ID.String()).
				Str("user_id", userID.String()).
				Err(pubErr).
				Msg("chant notifier: publish failed")
			continue
		}
		notified++
	}

	if notified > 0 {
		logger.Debug().
			Str("chant_id", chant.ID.String()).
			Int("users", notified).
			Int64("starts_in_seconds", startsIn).
			Msg("chant notifier: upcoming chant broadcast")
	}
}
