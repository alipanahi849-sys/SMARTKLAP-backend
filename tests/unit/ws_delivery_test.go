package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"clap/internal/modules/realtime/dto"
	"clap/internal/modules/realtime/gateway"
	"clap/internal/modules/realtime/metrics"
	"clap/internal/modules/realtime/ws"

	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Test server helpers ──────────────────────────────────────────────────────

type testWSServer struct {
	server *httptest.Server
	cm     *ws.ConnectionManager
	gw     *gateway.WebSocketRealtimeGateway
	cancel context.CancelFunc
}

func newTestWSServer(t *testing.T) *testWSServer {
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
		userID := uuid.New()
		client := ws.NewClient(hub, conn, userID)
		go client.WritePump()
		go client.ReadPump()
	}))

	t.Cleanup(func() {
		cancel()
		server.Close()
	})

	time.Sleep(20 * time.Millisecond) // let Hub start

	return &testWSServer{server: server, cm: cm, gw: gw, cancel: cancel}
}

func dialWS(t *testing.T, server *httptest.Server) *gorillaws.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := gorillaws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	return conn
}

// readEnvelope reads one message from the WS connection with a timeout.
func readEnvelope(t *testing.T, conn *gorillaws.Conn) *dto.EventEnvelope {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	var env dto.EventEnvelope
	require.NoError(t, json.Unmarshal(data, &env))
	return &env
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestWSDelivery_BroadcastReachesAllClients(t *testing.T) {
	srv := newTestWSServer(t)

	conn1 := dialWS(t, srv.server)
	conn2 := dialWS(t, srv.server)
	defer conn1.Close()
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond) // let registrations propagate

	env := dto.NewEnvelope(dto.EventTypeServerNotification, nil, map[string]any{"msg": "hello all"})
	require.NoError(t, srv.gw.BroadcastEnvelope(context.Background(), env))

	e1 := readEnvelope(t, conn1)
	e2 := readEnvelope(t, conn2)

	assert.Equal(t, dto.EventTypeServerNotification, e1.Type)
	assert.Equal(t, dto.EventTypeServerNotification, e2.Type)
}

func TestWSDelivery_MatchIsolation(t *testing.T) {
	srv := newTestWSServer(t)

	conn1 := dialWS(t, srv.server)
	conn2 := dialWS(t, srv.server)
	defer conn1.Close()
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	matchA := uuid.New()
	matchB := uuid.New()

	// Subscribe conn1 to match A, conn2 to match B.
	subMsgA, _ := json.Marshal(dto.ClientMessage{Type: "subscribe", Channel: ws.MatchChannel(matchA)})
	subMsgB, _ := json.Marshal(dto.ClientMessage{Type: "subscribe", Channel: ws.MatchChannel(matchB)})
	require.NoError(t, conn1.WriteMessage(gorillaws.TextMessage, subMsgA))
	require.NoError(t, conn2.WriteMessage(gorillaws.TextMessage, subMsgB))

	time.Sleep(100 * time.Millisecond) // let subscriptions register

	// Publish to match A.
	env := dto.NewEnvelope(dto.EventTypeMatchRuntimeUpdated, &matchA, dto.MatchRuntimePayload{Status: "running", ElapsedMs: 1000})
	require.NoError(t, srv.gw.PublishToMatch(context.Background(), matchA, env))

	// conn1 (subscribed to A) should receive the event.
	e := readEnvelope(t, conn1)
	assert.Equal(t, dto.EventTypeMatchRuntimeUpdated, e.Type)
	assert.Equal(t, matchA.String(), e.MatchID.String())

	// conn2 (subscribed to B) should NOT receive it — read with short timeout.
	conn2.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, readErr := conn2.ReadMessage()
	assert.Error(t, readErr, "conn2 should not receive events for match A")
}

func TestWSDelivery_ApplicationPingPong(t *testing.T) {
	srv := newTestWSServer(t)

	conn := dialWS(t, srv.server)
	defer conn.Close()

	time.Sleep(30 * time.Millisecond)

	// Send application-level ping.
	ping, _ := json.Marshal(dto.ClientMessage{Type: dto.EventTypePing})
	require.NoError(t, conn.WriteMessage(gorillaws.TextMessage, ping))

	// Expect pong back.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)

	var msg dto.ClientMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, dto.EventTypePong, msg.Type)
}

func TestWSDelivery_MultipleSubscriptions(t *testing.T) {
	srv := newTestWSServer(t)

	conn := dialWS(t, srv.server)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	match1 := uuid.New()
	match2 := uuid.New()

	sub1, _ := json.Marshal(dto.ClientMessage{Type: "subscribe", Channel: ws.MatchChannel(match1)})
	sub2, _ := json.Marshal(dto.ClientMessage{Type: "subscribe", Channel: ws.MatchChannel(match2)})
	require.NoError(t, conn.WriteMessage(gorillaws.TextMessage, sub1))
	require.NoError(t, conn.WriteMessage(gorillaws.TextMessage, sub2))
	time.Sleep(100 * time.Millisecond)

	// Publish to match1 — should be received.
	env1 := dto.NewEnvelope(dto.EventTypeMatchRuntimeUpdated, &match1, nil)
	require.NoError(t, srv.gw.PublishToMatch(context.Background(), match1, env1))

	e := readEnvelope(t, conn)
	assert.Equal(t, match1.String(), e.MatchID.String())

	// Publish to match2 — should also be received.
	env2 := dto.NewEnvelope(dto.EventTypeMatchRuntimeUpdated, &match2, nil)
	require.NoError(t, srv.gw.PublishToMatch(context.Background(), match2, env2))

	e2 := readEnvelope(t, conn)
	assert.Equal(t, match2.String(), e2.MatchID.String())
}

func TestWSDelivery_WelcomeOnConnect(t *testing.T) {
	srv := newTestWSServer(t)

	env := dto.NewEnvelope(dto.EventTypeChantUpcoming, nil, map[string]any{
		"chant_id": uuid.New().String(),
		"title":    "welcome",
	})
	data, err := json.Marshal(env)
	require.NoError(t, err)
	srv.cm.SetWelcomeMessage(data)

	conn := dialWS(t, srv.server)
	defer conn.Close()

	got := readEnvelope(t, conn)
	assert.Equal(t, dto.EventTypeChantUpcoming, got.Type)
}
