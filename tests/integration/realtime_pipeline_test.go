package integration

import (
	"context"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	schedulermodels "clap/internal/modules/eventscheduler/models"
	schedulerrepo "clap/internal/modules/eventscheduler/repository"
	schedulersvc "clap/internal/modules/eventscheduler/service"
	realtimemetrics "clap/internal/modules/realtime/metrics"
	realtimesvc "clap/internal/modules/realtime/service"
	realtimews "clap/internal/modules/realtime/ws"
	"clap/internal/shared/config"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain initialises a minimal config so JWT utilities work in the
// WebSocket authentication integration test. DB-backed tests in this package
// establish their own connections and are unaffected.
func TestMain(m *testing.M) {
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{
			Environment: "test",
			JWT: config.JWT{
				Secret:        "integration-test-secret-key-32-characters",
				AccessExpiry:  3600,
				RefreshExpiry: 86400,
				Issuer:        "clap-integration-test",
				RefreshSecret: "integration-test-refresh-secret-key-32c",
			},
		}
	}
	os.Exit(m.Run())
}

// ─── In-memory scheduler repository (no DB) ───────────────────────────────────

type fakeSchedulerRepo struct {
	mu     sync.Mutex
	events map[uuid.UUID]*schedulermodels.SchedulerEvent
}

func newFakeSchedulerRepo() *fakeSchedulerRepo {
	return &fakeSchedulerRepo{events: make(map[uuid.UUID]*schedulermodels.SchedulerEvent)}
}

func (r *fakeSchedulerRepo) Create(_ context.Context, ev *schedulermodels.SchedulerEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	if ev.UpdatedAt.IsZero() {
		ev.UpdatedAt = time.Now()
	}
	r.events[ev.ID] = ev
	return nil
}
func (r *fakeSchedulerRepo) FindByID(_ context.Context, id uuid.UUID) (*schedulermodels.SchedulerEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev, ok := r.events[id]; ok {
		return ev, nil
	}
	return nil, sharederrors.NewNotFound("not found", nil)
}
func (r *fakeSchedulerRepo) FindPendingUpTo(_ context.Context, upTo time.Time) ([]*schedulermodels.SchedulerEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*schedulermodels.SchedulerEvent
	for _, ev := range r.events {
		if ev.Status == schedulermodels.SchedulerEventPending && !ev.ExecuteAt.After(upTo) {
			out = append(out, ev)
		}
	}
	return out, nil
}
func (r *fakeSchedulerRepo) FindAllPending(_ context.Context) ([]*schedulermodels.SchedulerEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*schedulermodels.SchedulerEvent
	for _, ev := range r.events {
		if ev.Status == schedulermodels.SchedulerEventPending {
			out = append(out, ev)
		}
	}
	return out, nil
}
func (r *fakeSchedulerRepo) UpdateStatus(_ context.Context, id uuid.UUID, status schedulermodels.SchedulerEventStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev, ok := r.events[id]; ok {
		ev.Status = status
		ev.UpdatedAt = time.Now()
		return nil
	}
	return sharederrors.NewNotFound("not found", nil)
}
func (r *fakeSchedulerRepo) UpdateExecuteAt(_ context.Context, id uuid.UUID, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev, ok := r.events[id]; ok {
		ev.ExecuteAt = at
		return nil
	}
	return sharederrors.NewNotFound("not found", nil)
}
func (r *fakeSchedulerRepo) ClaimForProcessing(_ context.Context, id uuid.UUID) (*schedulermodels.SchedulerEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev, ok := r.events[id]; ok && ev.Status == schedulermodels.SchedulerEventPending {
		ev.Status = schedulermodels.SchedulerEventProcessing
		ev.UpdatedAt = time.Now()
		return ev, nil
	}
	return nil, sharederrors.NewNotFound("not available", nil)
}
func (r *fakeSchedulerRepo) MarkExecuted(_ context.Context, id uuid.UUID) error {
	return r.UpdateStatus(context.Background(), id, schedulermodels.SchedulerEventExecuted)
}
func (r *fakeSchedulerRepo) MarkFailed(_ context.Context, id uuid.UUID, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev, ok := r.events[id]; ok {
		ev.Status = schedulermodels.SchedulerEventFailed
		ev.FailReason = reason
		ev.UpdatedAt = time.Now()
		return nil
	}
	return sharederrors.NewNotFound("not found", nil)
}
func (r *fakeSchedulerRepo) ResetStaleProcessing(_ context.Context, olderThan time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var reset int64
	for _, ev := range r.events {
		if ev.Status == schedulermodels.SchedulerEventProcessing && ev.UpdatedAt.Before(olderThan) {
			ev.Status = schedulermodels.SchedulerEventPending
			reset++
		}
	}
	return reset, nil
}
func (r *fakeSchedulerRepo) DeleteTerminalOlderThan(_ context.Context, olderThan time.Time) (int64, error) {
	return 0, nil
}

func (r *fakeSchedulerRepo) statusOf(id uuid.UUID) schedulermodels.SchedulerEventStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev, ok := r.events[id]; ok {
		return ev.Status
	}
	return ""
}

var _ schedulerrepo.SchedulerEventRepository = (*fakeSchedulerRepo)(nil)

// ─── Capturing gateway ────────────────────────────────────────────────────────

type capturedEvent struct {
	MatchID   uuid.UUID
	EventType string
	Payload   any
}

type capturingGateway struct {
	mu       sync.Mutex
	captured []capturedEvent
}

func (g *capturingGateway) PublishMatchEvent(_ context.Context, matchID uuid.UUID, eventType string, payload any) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.captured = append(g.captured, capturedEvent{MatchID: matchID, EventType: eventType, Payload: payload})
	return nil
}

func (g *capturingGateway) BroadcastEvent(_ context.Context, matchID uuid.UUID, eventType string, payload any) error {
	return g.PublishMatchEvent(context.Background(), matchID, eventType, payload)
}
func (g *capturingGateway) count() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.captured)
}

var _ realtimesvc.DispatchGateway = (*capturingGateway)(nil)

// ─── Event dispatch flow ──────────────────────────────────────────────────────

func TestIntegration_EventDispatchFlow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeSchedulerRepo()
	sched := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	svc := schedulersvc.NewEventSchedulerService(repo, sched)

	matchID := uuid.New()
	ev, err := svc.RegisterEvent(ctx, &schedulersvc.RegisterEventRequest{
		SessionID: uuid.New(),
		EventType: "song.playback.started",
		ExecuteAt: time.Now().Add(-time.Second), // already due
		Payload: map[string]any{
			"match_id":    matchID.String(),
			"event_type":  "song.playback.started",
			"song_id":     uuid.New().String(),
			"schedule_id": uuid.New().String(),
		},
	})
	require.NoError(t, err)

	gw := &capturingGateway{}
	disp := realtimesvc.NewEventDispatcher(sched, repo, gw, 20*time.Millisecond)

	runCtx, cancel := context.WithCancel(ctx)
	go disp.Run(runCtx)

	require.Eventually(t, func() bool { return gw.count() >= 1 }, 3*time.Second, 20*time.Millisecond,
		"dispatcher should publish the due event")

	cancel()
	select {
	case <-disp.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("dispatcher did not stop after context cancel")
	}

	// Event must be marked executed after successful publish.
	assert.Equal(t, schedulermodels.SchedulerEventExecuted, repo.statusOf(ev.ID))

	gw.mu.Lock()
	got := gw.captured[0]
	gw.mu.Unlock()
	assert.Equal(t, matchID, got.MatchID)
	assert.Equal(t, "song.playback.started", got.EventType)
}

// ─── Stale processing crash recovery ──────────────────────────────────────────

func TestIntegration_StaleProcessingRecovery(t *testing.T) {
	ctx := context.Background()
	repo := newFakeSchedulerRepo()

	// Simulate an event orphaned in processing by a crashed dispatcher.
	stale := &schedulermodels.SchedulerEvent{
		ID:          uuid.New(),
		SessionID:   uuid.New(),
		EventType:   "lyrics.line.changed",
		ExecuteAt:   time.Now().Add(time.Minute),
		PayloadJSON: `{}`,
		Status:      schedulermodels.SchedulerEventProcessing,
		UpdatedAt:   time.Now().Add(-10 * time.Minute),
	}
	require.NoError(t, repo.Create(ctx, stale))

	sched := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	recovery := schedulersvc.NewSchedulerRecoveryServiceWithConfig(repo, sched, 5*time.Minute, nil)

	recovered, err := recovery.RecoverPendingEvents(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, recovered)
	assert.Equal(t, schedulermodels.SchedulerEventPending, repo.statusOf(stale.ID))
	assert.Equal(t, 1, sched.Size())
}

// ─── Graceful shutdown ordering ───────────────────────────────────────────────

func TestIntegration_GracefulShutdownStopsHubAndDispatcher(t *testing.T) {
	repo := newFakeSchedulerRepo()
	sched := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	gw := &capturingGateway{}

	m := realtimemetrics.New()
	hub := realtimews.NewHub(m)
	disp := realtimesvc.NewEventDispatcher(sched, repo, gw, 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	go disp.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	// Cancel the shared application context and verify both drain.
	cancel()

	select {
	case <-disp.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("dispatcher did not stop")
	}
	select {
	case <-hub.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("hub did not stop")
	}
	assert.True(t, hub.Healthy(), "hub should remain healthy through a clean shutdown")
}

// ─── WebSocket authentication ─────────────────────────────────────────────────

func TestIntegration_WebSocketAuthentication(t *testing.T) {
	// Valid token authenticates and exposes the expiry for session enforcement.
	userID := uuid.New()
	token, _, err := utils.GenerateAccessToken(userID, "user@example.com", []string{"user"})
	require.NoError(t, err)

	req := newAuthRequest("Bearer " + token)
	res, err := realtimews.Authenticate(req)
	require.NoError(t, err)
	assert.Equal(t, userID, res.UserID)
	assert.False(t, res.ExpiresAt.IsZero(), "expiry must be populated for session-expiry enforcement")

	// Missing token is rejected.
	_, err = realtimews.Authenticate(newAuthRequest(""))
	assert.ErrorIs(t, err, realtimews.ErrMissingToken)

	// Invalid token is rejected.
	_, err = realtimews.Authenticate(newAuthRequest("Bearer not.a.jwt"))
	assert.ErrorIs(t, err, realtimews.ErrInvalidToken)
}

func newAuthRequest(authHeader string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/realtime/ws", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	return req
}
