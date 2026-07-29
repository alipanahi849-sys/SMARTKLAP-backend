// Package ws contains the pure WebSocket transport layer.
// No business logic lives here — only connection management and message routing.
package ws

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"

	"clap/internal/modules/realtime/metrics"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

// ─── Internal message types ───────────────────────────────────────────────────

type subscriptionMsg struct {
	client  *Client
	channel string
	add     bool // true=subscribe, false=unsubscribe
}

type publishMsg struct {
	// Exactly one of channel/userID should be set; both nil = broadcast.
	channel *string
	userID  *uuid.UUID
	data    []byte
}

// disconnectMsg requests that every connection belonging to a user be closed.
type disconnectMsg struct {
	userID uuid.UUID
}

// ─── Hub ──────────────────────────────────────────────────────────────────────

// Hub is the central message broker for WebSocket connections.
// All state mutations happen exclusively inside the Run() goroutine to guarantee
// thread-safety without holding locks while writing to client channels.
//
// Lifecycle:
//
//	hub := NewHub(m)
//	ctx, cancel := context.WithCancel(parent)
//	go hub.Run(ctx)   // start the event loop
//	...
//	cancel()          // stops the loop and disconnects all clients
//	<-hub.Done()      // wait for clean shutdown
type Hub struct {
	// Channels are buffered to avoid blocking callers when the loop is busy.
	register       chan *Client
	unregister     chan *Client
	subscribe      chan subscriptionMsg
	publish        chan publishMsg
	disconnectUser chan disconnectMsg
	done           chan struct{} // closed when Run() exits

	metrics *metrics.Metrics
	healthy atomic.Bool
}

const (
	hubRegisterBuf       = 32
	hubUnregisterBuf     = 32
	hubSubscribeBuf      = 256
	hubPublishBuf        = 1024
	hubDisconnectUserBuf = 32

	// hubShutdownDrainTimeout bounds how long teardown waits to drain
	// unregister sends from per-client readPump goroutines so they never
	// block permanently on a stopped Hub (fixes the shutdown goroutine leak).
	hubShutdownDrainTimeout = 5 * time.Second
)

// NewHub constructs a Hub with the given metrics handle.
func NewHub(m *metrics.Metrics) *Hub {
	h := &Hub{
		register:       make(chan *Client, hubRegisterBuf),
		unregister:     make(chan *Client, hubUnregisterBuf),
		subscribe:      make(chan subscriptionMsg, hubSubscribeBuf),
		publish:        make(chan publishMsg, hubPublishBuf),
		disconnectUser: make(chan disconnectMsg, hubDisconnectUserBuf),
		done:           make(chan struct{}),
		metrics:        m,
	}
	h.healthy.Store(true)
	return h
}

// Done returns a channel that is closed when Run() has fully exited.
// Callers use this during graceful shutdown to wait for the Hub to drain.
func (h *Hub) Done() <-chan struct{} { return h.done }

// Healthy reports whether the Hub event loop is operating normally.
// It flips to false if a panic was recovered inside the loop.
func (h *Hub) Healthy() bool { return h.healthy.Load() }

// Run is the single goroutine that owns all Hub state.
// It exits cleanly when ctx is cancelled, closing every connected client.
//
// Every state mutation is wrapped in a panic-isolating helper so that a bug in
// one operation can never terminate the event loop or corrupt the system for
// other clients (CR-1).
func (h *Hub) Run(ctx context.Context) {
	defer close(h.done)

	// Hub-owned state — never touched outside this goroutine.
	clients := make(map[*Client]bool)
	channels := make(map[string]map[*Client]bool) // channel → client set
	users := make(map[uuid.UUID]map[*Client]bool) // userID → client set

	for {
		select {
		case <-ctx.Done():
			h.teardown(clients, channels, users)
			return

		case c := <-h.register:
			h.safe("register", func() { h.doRegister(c, clients, users) })

		case c := <-h.unregister:
			h.safe("unregister", func() { h.doUnregister(c, clients, channels, users) })

		case s := <-h.subscribe:
			h.safe("subscribe", func() { h.doSubscribe(s, channels) })

		case d := <-h.disconnectUser:
			h.safe("disconnect_user", func() { h.doDisconnectUser(d, clients, channels, users) })

		case msg := <-h.publish:
			h.safe("publish", func() { h.dispatch(msg, clients, channels, users) })
		}
	}
}

// safe runs a Hub state mutation with panic isolation.  A recovered panic is
// logged with a full stack trace and recorded in metrics; the loop continues
// serving all other clients (CR-1 / F-007).
func (h *Hub) safe(op string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			h.metrics.HubPanics.Add(1)
			h.healthy.Store(false)
			logger.Error().
				Str("hub_op", op).
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("hub_panic_recovered")
		}
	}()
	fn()
}

// ─── State mutations (run only inside Run goroutine) ──────────────────────────

func (h *Hub) doRegister(c *Client, clients map[*Client]bool, users map[uuid.UUID]map[*Client]bool) {
	clients[c] = true
	if users[c.UserID] == nil {
		users[c.UserID] = make(map[*Client]bool)
	}
	users[c.UserID][c] = true
	h.metrics.ActiveConnections.Add(1)
	logger.Info().
		Str("user_id", c.UserID.String()).
		Msg("connection_opened")
}

func (h *Hub) doUnregister(
	c *Client,
	clients map[*Client]bool,
	channels map[string]map[*Client]bool,
	users map[uuid.UUID]map[*Client]bool,
) {
	if !clients[c] {
		return
	}
	delete(clients, c)
	h.closeClient(c)

	// Remove from all channel subscriptions.
	for ch, members := range channels {
		if members[c] {
			delete(members, c)
			h.metrics.ActiveSubscriptions.Add(-1)
			if len(members) == 0 {
				delete(channels, ch)
			}
		}
	}

	// Remove from user index.
	if userClients := users[c.UserID]; userClients != nil {
		delete(userClients, c)
		if len(userClients) == 0 {
			delete(users, c.UserID)
		}
	}

	h.metrics.ActiveConnections.Add(-1)
	logger.Info().
		Str("user_id", c.UserID.String()).
		Msg("connection_closed")
}

func (h *Hub) doSubscribe(s subscriptionMsg, channels map[string]map[*Client]bool) {
	if s.add {
		if channels[s.channel] == nil {
			channels[s.channel] = make(map[*Client]bool)
		}
		if !channels[s.channel][s.client] {
			channels[s.channel][s.client] = true
			h.metrics.ActiveSubscriptions.Add(1)
			logger.Info().
				Str("user_id", s.client.UserID.String()).
				Str("channel", s.channel).
				Msg("subscription_added")
		}
		return
	}

	if channels[s.channel] != nil && channels[s.channel][s.client] {
		delete(channels[s.channel], s.client)
		h.metrics.ActiveSubscriptions.Add(-1)
		if len(channels[s.channel]) == 0 {
			delete(channels, s.channel)
		}
		logger.Info().
			Str("user_id", s.client.UserID.String()).
			Str("channel", s.channel).
			Msg("subscription_removed")
	}
}

// doDisconnectUser forcibly terminates every connection owned by a user and
// removes them from all Hub state (CR-9).
func (h *Hub) doDisconnectUser(
	d disconnectMsg,
	clients map[*Client]bool,
	channels map[string]map[*Client]bool,
	users map[uuid.UUID]map[*Client]bool,
) {
	targets := users[d.userID]
	if len(targets) == 0 {
		return
	}
	// Snapshot first; doUnregister mutates the users map.
	victims := make([]*Client, 0, len(targets))
	for c := range targets {
		victims = append(victims, c)
	}
	for _, c := range victims {
		h.doUnregister(c, clients, channels, users)
		h.metrics.UsersDisconnected.Add(1)
	}
	logger.Info().
		Str("user_id", d.userID.String()).
		Int("connections_closed", len(victims)).
		Msg("user_disconnected")
}

// dispatch routes a publishMsg to the appropriate clients.
func (h *Hub) dispatch(
	msg publishMsg,
	clients map[*Client]bool,
	channels map[string]map[*Client]bool,
	users map[uuid.UUID]map[*Client]bool,
) {
	switch {
	case msg.channel != nil:
		for c := range channels[*msg.channel] {
			h.sendToClient(c, msg.data)
		}

	case msg.userID != nil:
		for c := range users[*msg.userID] {
			h.sendToClient(c, msg.data)
		}

	default:
		for c := range clients {
			h.sendToClient(c, msg.data)
		}
	}
}

// sendToClient writes to a client's send buffer.
//
// It first checks whether the client has already been closed: sending on a
// closed channel panics in Go, and a select/default does NOT protect against a
// closed channel (only a full one). Guarding on c.closed makes delivery safe
// even when the client is still present in a channel map but pending eviction
// (fixes F-001).
//
// If the buffer is full the client is considered slow/dead and is closed; its
// readPump/writePump goroutines then unblock and the client is fully removed on
// the subsequent unregister.
func (h *Hub) sendToClient(c *Client, data []byte) {
	select {
	case <-c.closed:
		// Already evicted — never write to a closed Send channel.
		return
	default:
	}

	select {
	case c.Send <- data:
		h.metrics.EventsDelivered.Add(1)
		logger.Debug().
			Str("user_id", c.UserID.String()).
			Int("bytes", len(data)).
			Msg("event_delivered")
	default:
		h.closeClient(c)
		h.metrics.EventsFailed.Add(1)
		h.metrics.ClientsDroppedBufferFull.Add(1)
		logger.Warn().
			Str("user_id", c.UserID.String()).
			Str("reason", "send_buffer_full").
			Msg("client_dropped")
	}
}

// closeClient drains and closes the client's Send channel.
// Idempotent — safe to call more than once.
func (h *Hub) closeClient(c *Client) {
	select {
	case <-c.closed:
		// already closed
	default:
		close(c.closed)
		close(c.Send)
	}
}

// teardown closes every active client on shutdown and then drains the
// unregister channel for a bounded period so that per-client readPump
// goroutines blocked on `hub.unregister <- c` are released rather than leaked
// (fixes F-006).
func (h *Hub) teardown(
	clients map[*Client]bool,
	channels map[string]map[*Client]bool,
	users map[uuid.UUID]map[*Client]bool,
) {
	remaining := len(clients)
	for c := range clients {
		h.closeClient(c)
	}
	// Clear maps eagerly to release references.
	for k := range channels {
		delete(channels, k)
	}
	for k := range users {
		delete(users, k)
	}
	for k := range clients {
		delete(clients, k)
	}

	if remaining == 0 {
		return
	}

	deadline := time.After(hubShutdownDrainTimeout)
	for remaining > 0 {
		select {
		case <-h.unregister:
			remaining--
		case <-deadline:
			logger.Warn().
				Int("undrained_clients", remaining).
				Msg("hub teardown drain timeout; remaining readPumps will exit on connection close")
			return
		}
	}
}

// ─── Channel helpers ──────────────────────────────────────────────────────────

// MatchChannel returns the canonical channel name for a match.
func MatchChannel(matchID uuid.UUID) string {
	return fmt.Sprintf("match:%s", matchID.String())
}
