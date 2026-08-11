package service

import (
	"context"
	"time"

	chantdto "clap/internal/modules/chant/dto"
	chantmodels "clap/internal/modules/chant/models"
	chantrepo "clap/internal/modules/chant/repository"
	realtimedto "clap/internal/modules/realtime/dto"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

const (
	// defaultChantLeadTime is how far before a chant's scheduled start the
	// upcoming notification is emitted.
	defaultChantLeadTime = 2 * time.Minute
	// defaultChantNotifierInterval is how often the DB is polled for chants
	// entering the lead-time window.
	defaultChantNotifierInterval = 15 * time.Second
)

// ChantLyricsProvider builds the full synced-lyrics payload for a chant.
// Implemented by chant/service.ChantService.
type ChantLyricsProvider interface {
	Lyrics(ctx context.Context, chantID uuid.UUID, mode string) (*chantdto.ChantLyricsResponse, error)
}

// UserEnvelopePublisher delivers an EventEnvelope to all connections of one
// user. Implemented by WebSocketRealtimeGateway.
type UserEnvelopePublisher interface {
	PublishToUser(ctx context.Context, userID uuid.UUID, env *realtimedto.EventEnvelope) error
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

// pendingChant tracks one chant inside the lead-time window: its cached lyrics
// and which users have already been notified (so each user gets the event
// exactly once, including users who subscribe mid-window).
type pendingChant struct {
	chant  chantmodels.Chant
	lyrics *chantdto.ChantLyricsResponse
	sent   map[uuid.UUID]bool
}

// ChantUpcomingNotifier is a background service that watches for active chants
// approaching their scheduled start and pushes a personalised chant.upcoming
// event to every connected user, regardless of channel subscriptions — the
// mobile app must surface the countdown no matter which screen is open.
//
// Lifecycle mirrors EventDispatcher:
//
//	n := NewChantUpcomingNotifier(...)
//	go n.Run(ctx)   // blocks until ctx is cancelled
//	<-n.Done()
type ChantUpcomingNotifier struct {
	chantRepo chantrepo.ChantRepository
	lyricsSvc ChantLyricsProvider
	users     ConnectedUserLister
	publisher UserEnvelopePublisher
	interval  time.Duration
	leadTime  time.Duration

	// pending is touched only by the Run goroutine — no locking needed.
	pending map[uuid.UUID]*pendingChant
	done    chan struct{}
}

// NewChantUpcomingNotifier constructs the notifier.
// Pass 0 for interval/leadTime to use the defaults (15s poll, 2min lead).
func NewChantUpcomingNotifier(
	chantRepo chantrepo.ChantRepository,
	lyricsSvc ChantLyricsProvider,
	users ConnectedUserLister,
	publisher UserEnvelopePublisher,
	interval, leadTime time.Duration,
) *ChantUpcomingNotifier {
	if interval <= 0 {
		interval = defaultChantNotifierInterval
	}
	if leadTime <= 0 {
		leadTime = defaultChantLeadTime
	}
	return &ChantUpcomingNotifier{
		chantRepo: chantRepo,
		lyricsSvc: lyricsSvc,
		users:     users,
		publisher: publisher,
		interval:  interval,
		leadTime:  leadTime,
		pending:   make(map[uuid.UUID]*pendingChant),
		done:      make(chan struct{}),
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
			entry = &pendingChant{chant: chant, sent: make(map[uuid.UUID]bool)}
			n.pending[chant.ID] = entry
		}

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

		n.notifyUsers(ctx, entry)
	}

	// Drop chants whose start time has passed.
	for id, entry := range n.pending {
		if entry.chant.ScheduledAt.Before(now) {
			delete(n.pending, id)
		}
	}
}

// notifyUsers sends the event to every connected user that has not been
// notified about this chant yet.
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

	for _, userID := range userIDs {
		if entry.sent[userID] {
			continue
		}

		todayPoints, pointsErr := n.chantRepo.TodayPoints(ctx, userID)
		if pointsErr != nil {
			// Skip so the user is retried on the next tick.
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
		entry.sent[userID] = true

		logger.Info().
			Str("chant_id", chant.ID.String()).
			Str("user_id", userID.String()).
			Int64("starts_in_seconds", startsIn).
			Msg("chant notifier: upcoming chant notified")
	}
}
