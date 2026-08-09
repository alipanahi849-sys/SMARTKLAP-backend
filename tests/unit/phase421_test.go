package unit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	schedulermodels "clap/internal/modules/eventscheduler/models"
	schedulersvc "clap/internal/modules/eventscheduler/service"
	"clap/internal/modules/realtime"
	"clap/internal/modules/realtime/gateway"
	"clap/internal/modules/realtime/metrics"
	realtimemodels "clap/internal/modules/realtime/models"
	realtimerepo "clap/internal/modules/realtime/repository"
	realtimesvc "clap/internal/modules/realtime/service"
	"clap/internal/modules/realtime/ws"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── CR-1: Hub health ─────────────────────────────────────────────────────────

func TestHub_HealthyByDefault(t *testing.T) {
	m := metrics.New()
	hub := ws.NewHub(m)
	assert.True(t, hub.Healthy(), "a freshly constructed hub must report healthy")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	time.Sleep(20 * time.Millisecond)
	assert.True(t, hub.Healthy(), "hub must remain healthy while running normally")
}

// ─── Customisable WS test server ──────────────────────────────────────────────

type ctrlWSServer struct {
	server  *httptest.Server
	cm      *ws.ConnectionManager
	gw      *gateway.WebSocketRealtimeGateway
	metrics *metrics.Metrics
}

// newCtrlWSServer builds a WS test server where each connection is assigned the
// given userID and client options, so tests can drive DisconnectUser and JWT
// expiry deterministically.
func newCtrlWSServer(t *testing.T, userID uuid.UUID, opts ws.ClientOptions) *ctrlWSServer {
	t.Helper()

	m := metrics.New()
	hub := ws.NewHub(m)
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)

	cm := ws.NewConnectionManager(hub)
	gw := gateway.NewWebSocketRealtimeGateway(cm, m)

	upgrader := gorillaws.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := ws.NewClientWithOptions(hub, conn, userID, opts)
		go client.WritePump()
		go client.ReadPump()
	}))

	t.Cleanup(func() {
		cancel()
		server.Close()
	})
	time.Sleep(20 * time.Millisecond)
	return &ctrlWSServer{server: server, cm: cm, gw: gw, metrics: m}
}

func dialCtrl(t *testing.T, srv *ctrlWSServer) *gorillaws.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.server.URL, "http")
	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	return conn
}

// ─── CR-9: DisconnectUser ─────────────────────────────────────────────────────

func TestWS_DisconnectUser_ClosesConnection(t *testing.T) {
	userID := uuid.New()
	srv := newCtrlWSServer(t, userID, ws.ClientOptions{})

	conn := dialCtrl(t, srv)
	defer conn.Close()
	time.Sleep(50 * time.Millisecond)

	require.Equal(t, int64(1), srv.cm.ActiveConnectionCount())

	require.NoError(t, srv.gw.DisconnectUser(context.Background(), userID.String()))

	// The client should be closed by the server.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := conn.ReadMessage()
	assert.Error(t, err, "connection must be closed after DisconnectUser")

	// Give the hub a moment to process the eviction.
	time.Sleep(50 * time.Millisecond)
	assert.GreaterOrEqual(t, srv.metrics.UsersDisconnected.Load(), int64(1))
	assert.Equal(t, int64(0), srv.cm.ActiveConnectionCount())
}

func TestWS_DisconnectUser_InvalidUUID(t *testing.T) {
	userID := uuid.New()
	srv := newCtrlWSServer(t, userID, ws.ClientOptions{})
	assert.Error(t, srv.gw.DisconnectUser(context.Background(), "not-a-uuid"))
}

// ─── CR-8: JWT expiry disconnect ──────────────────────────────────────────────

func TestWS_JWTExpiryDisconnect(t *testing.T) {
	userID := uuid.New()
	// Token expires shortly; the writePump should tear the connection down.
	srv := newCtrlWSServer(t, userID, ws.ClientOptions{
		TokenExpiresAt: time.Now().Add(300 * time.Millisecond),
	})

	conn := dialCtrl(t, srv)
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err := conn.ReadMessage()
	assert.Error(t, err, "connection must be closed after the token expires")

	time.Sleep(50 * time.Millisecond)
	assert.GreaterOrEqual(t, srv.metrics.SessionsExpired.Load(), int64(1))
}

func TestWS_NoExpiryStaysConnected(t *testing.T) {
	userID := uuid.New()
	srv := newCtrlWSServer(t, userID, ws.ClientOptions{}) // no expiry

	conn := dialCtrl(t, srv)
	defer conn.Close()

	// Should NOT be disconnected within a short window.
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err := conn.ReadMessage()
	assert.Error(t, err) // read timeout (not a close) — connection still open
	assert.Equal(t, int64(0), srv.metrics.SessionsExpired.Load())
}

// ─── CR-4: subscription rate limiter (fail-open semantics) ────────────────────

func TestSubscriptionRateLimiter_EmptyIPAllows(t *testing.T) {
	// No client IP → never blocked (used by non-HTTP transports / tests).
	assert.True(t, ws.SubscriptionRateLimiter(context.Background(), ""))
}

func TestSubscriptionRateLimiter_NoRedisFailsOpen(t *testing.T) {
	// Redis is not initialised in unit tests, so the limiter must fail open
	// rather than reject legitimate subscriptions.
	assert.True(t, ws.SubscriptionRateLimiter(context.Background(), "203.0.113.7"))
}

// ─── CR-2: watchdog / stale processing recovery ───────────────────────────────

func TestRecovery_ResetsStaleProcessingEvents(t *testing.T) {
	repo := &stubSchedulerEventRepo{}

	staleSession := uuid.New()
	// A stale processing event (updated long ago) should be reclaimed to pending.
	repo.events = append(repo.events, &schedulermodels.SchedulerEvent{
		ID:          uuid.New(),
		SessionID:   staleSession,
		EventType:   "song.playback.started",
		ExecuteAt:   time.Now().Add(time.Minute),
		PayloadJSON: `{}`,
		Status:      schedulermodels.SchedulerEventProcessing,
		UpdatedAt:   time.Now().Add(-10 * time.Minute),
	})
	// A fresh processing event (recently updated) must NOT be reclaimed.
	repo.events = append(repo.events, &schedulermodels.SchedulerEvent{
		ID:          uuid.New(),
		SessionID:   staleSession,
		EventType:   "song.playback.started",
		ExecuteAt:   time.Now().Add(time.Minute),
		PayloadJSON: `{}`,
		Status:      schedulermodels.SchedulerEventProcessing,
		UpdatedAt:   time.Now(),
	})

	scheduler := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	recoverySvc := schedulersvc.NewSchedulerRecoveryServiceWithConfig(
		repo, scheduler, 5*time.Minute, nil,
	)

	recovered, err := recoverySvc.RecoverPendingEvents(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, recovered, "only the stale event should be reclaimed and rehydrated")
	assert.Equal(t, 1, scheduler.Size())
}

func TestRecovery_Watchdog_StopsOnContextCancel(t *testing.T) {
	repo := &stubSchedulerEventRepo{}
	scheduler := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	recoverySvc := schedulersvc.NewSchedulerRecoveryService(repo, scheduler)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		recoverySvc.RunWatchdog(ctx, 20*time.Millisecond)
		close(done)
	}()

	time.Sleep(60 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watchdog did not stop within 2s of context cancellation")
	}
}

// ─── CR-14: data retention ────────────────────────────────────────────────────

// stubRealtimeEventRepo is an in-memory RealtimeEventRepository for retention tests.
type stubRealtimeEventRepo struct {
	events []*realtimemodels.RealtimeEvent
}

func (r *stubRealtimeEventRepo) Create(_ context.Context, e *realtimemodels.RealtimeEvent) error {
	r.events = append(r.events, e)
	return nil
}
func (r *stubRealtimeEventRepo) CreateBatch(_ context.Context, events []*realtimemodels.RealtimeEvent) error {
	r.events = append(r.events, events...)
	return nil
}
func (r *stubRealtimeEventRepo) FindBySessionID(_ context.Context, sessionID uuid.UUID) ([]*realtimemodels.RealtimeEvent, error) {
	return nil, nil
}
func (r *stubRealtimeEventRepo) FindBySessionIDOrdered(_ context.Context, sessionID uuid.UUID) ([]*realtimemodels.RealtimeEvent, error) {
	return nil, nil
}
func (r *stubRealtimeEventRepo) DeleteBySessionID(_ context.Context, sessionID uuid.UUID) error {
	return nil
}
func (r *stubRealtimeEventRepo) DeleteOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	var kept []*realtimemodels.RealtimeEvent
	var deleted int64
	for _, e := range r.events {
		if e.CreatedAt.Before(cutoff) {
			deleted++
			continue
		}
		kept = append(kept, e)
	}
	r.events = kept
	return deleted, nil
}

var _ realtimerepo.RealtimeEventRepository = (*stubRealtimeEventRepo)(nil)

func TestRetention_CleanupSchedulerAndRealtimeEvents(t *testing.T) {
	schedRepo := &stubSchedulerEventRepo{}
	rtRepo := &stubRealtimeEventRepo{}

	old := time.Now().UTC().AddDate(0, 0, -30)
	recent := time.Now().UTC()

	// Terminal scheduler events: 2 old (deletable), 1 recent (kept), 1 pending old (kept).
	schedRepo.events = append(schedRepo.events,
		&schedulermodels.SchedulerEvent{ID: uuid.New(), Status: schedulermodels.SchedulerEventExecuted, UpdatedAt: old},
		&schedulermodels.SchedulerEvent{ID: uuid.New(), Status: schedulermodels.SchedulerEventFailed, UpdatedAt: old},
		&schedulermodels.SchedulerEvent{ID: uuid.New(), Status: schedulermodels.SchedulerEventExecuted, UpdatedAt: recent},
		&schedulermodels.SchedulerEvent{ID: uuid.New(), Status: schedulermodels.SchedulerEventPending, UpdatedAt: old},
	)
	// Realtime events: 3 old (deletable), 2 recent (kept).
	rtRepo.events = append(rtRepo.events,
		&realtimemodels.RealtimeEvent{ID: uuid.New(), CreatedAt: old},
		&realtimemodels.RealtimeEvent{ID: uuid.New(), CreatedAt: old},
		&realtimemodels.RealtimeEvent{ID: uuid.New(), CreatedAt: old},
		&realtimemodels.RealtimeEvent{ID: uuid.New(), CreatedAt: recent},
		&realtimemodels.RealtimeEvent{ID: uuid.New(), CreatedAt: recent},
	)

	svc := realtimesvc.NewDataRetentionService(schedRepo, rtRepo, realtimesvc.RetentionConfig{
		SchedulerEventRetentionDays: 7,
		RealtimeEventRetentionDays:  7,
	})

	result, err := svc.CleanupAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.SchedulerEventsDeleted, "only terminal old scheduler events are deleted")
	assert.Equal(t, int64(3), result.RealtimeEventsDeleted)
	assert.Len(t, schedRepo.events, 2, "recent terminal + old pending events are retained")
	assert.Len(t, rtRepo.events, 2)
}

func TestRetention_DefaultsApplied(t *testing.T) {
	schedRepo := &stubSchedulerEventRepo{}
	rtRepo := &stubRealtimeEventRepo{}
	// Zero config must fall back to defaults without panicking.
	svc := realtimesvc.NewDataRetentionService(schedRepo, rtRepo, realtimesvc.RetentionConfig{})
	_, err := svc.CleanupAll(context.Background())
	require.NoError(t, err)
}

// ─── CR-6 / CR-7: endpoint authorization ──────────────────────────────────────

func newRealtimeRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	m := metrics.New()
	hub := ws.NewHub(m)
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)
	t.Cleanup(cancel)
	cm := ws.NewConnectionManager(hub)

	recoverySvc := realtimesvc.NewReconnectionRecoveryService(nil, nil)

	r := gin.New()
	v1 := r.Group("/api/v1")
	realtime.RegisterRoutesWithWS(v1, realtime.WSConfig{
		CM:          cm,
		RecoverySvc: recoverySvc,
		Metrics:     m,
	})
	return r
}

func adminToken(t *testing.T) string {
	t.Helper()
	tok, _, err := utils.GenerateAccessToken(uuid.New(), "admin@example.com", []string{string(utils.RoleAdmin)})
	require.NoError(t, err)
	return tok
}

func userToken(t *testing.T) string {
	t.Helper()
	tok, _, err := utils.GenerateAccessToken(uuid.New(), "user@example.com", []string{string(utils.RoleUser)})
	require.NoError(t, err)
	return tok
}

func TestRecoveryEndpoint_RequiresAuth(t *testing.T) {
	r := newRealtimeRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/realtime/session/"+uuid.New().String(), nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "recovery endpoint must reject unauthenticated requests")
}

func TestMetricsEndpoint_RequiresAuth(t *testing.T) {
	r := newRealtimeRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/realtime/metrics", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMetricsEndpoint_RejectsNonAdmin(t *testing.T) {
	r := newRealtimeRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/realtime/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code, "non-admin users must not read metrics")
}

func TestMetricsEndpoint_AllowsAdmin(t *testing.T) {
	r := newRealtimeRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/realtime/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken(t))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
