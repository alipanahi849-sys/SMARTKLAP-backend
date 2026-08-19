// Package ws contains the pure WebSocket transport layer.
// No business logic lives here — only connection management and message routing.
package ws

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
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

// usersQuery asks a shard loop for a snapshot of distinct user IDs — either
// every connected user (all=true) or the subscribers of one channel.
// reply MUST be buffered (cap >= 1) so the loop never blocks on a slow caller.
type usersQuery struct {
	channel string
	all     bool
	reply   chan []uuid.UUID
}

// ─── Hub ──────────────────────────────────────────────────────────────────────

// Hub is the central message broker for WebSocket connections.
// Connections are partitioned across independent shards so a broadcast does
// not stall register/subscribe on a single goroutine.
//
// Lifecycle:
//
//	hub := NewHub(m)
//	ctx, cancel := context.WithCancel(parent)
//	go hub.Run(ctx)   // start the shard event loops
//	...
//	cancel()          // stops the loops and disconnects all clients
//	<-hub.Done()      // wait for clean shutdown
type Hub struct {
	shards []*hubShard
	done   chan struct{} // closed when Run() exits

	metrics *metrics.Metrics
	healthy atomic.Bool
	welcome atomic.Value // []byte snapshot delivered to newly registered clients
}

const (
	defaultShardCount = 32

	hubRegisterBuf       = 32
	hubUnregisterBuf     = 32
	hubSubscribeBuf      = 256
	hubPublishBuf        = 1024
	hubDisconnectUserBuf = 32
	hubUsersQueryBuf     = 32

	// hubShutdownDrainTimeout bounds how long teardown waits to drain
	// unregister sends from per-client readPump goroutines so they never
	// block permanently on a stopped Hub (fixes the shutdown goroutine leak).
	hubShutdownDrainTimeout = 5 * time.Second
)

type hubShard struct {
	hub *Hub

	register       chan *Client
	unregister     chan *Client
	subscribe      chan subscriptionMsg
	publish        chan publishMsg
	disconnectUser chan disconnectMsg
	usersQueries   chan usersQuery
}

// NewHub constructs a Hub with the given metrics handle.
func NewHub(m *metrics.Metrics) *Hub {
	return newHub(m, defaultShardCount)
}

func newHub(m *metrics.Metrics, shardCount int) *Hub {
	if shardCount < 1 {
		shardCount = 1
	}
	h := &Hub{
		shards:  make([]*hubShard, shardCount),
		done:    make(chan struct{}),
		metrics: m,
	}
	h.healthy.Store(true)
	h.welcome.Store([]byte(nil))
	for i := 0; i < shardCount; i++ {
		h.shards[i] = &hubShard{
			hub:            h,
			register:       make(chan *Client, hubRegisterBuf),
			unregister:     make(chan *Client, hubUnregisterBuf),
			subscribe:      make(chan subscriptionMsg, hubSubscribeBuf),
			publish:        make(chan publishMsg, hubPublishBuf),
			disconnectUser: make(chan disconnectMsg, hubDisconnectUserBuf),
			usersQueries:   make(chan usersQuery, hubUsersQueryBuf),
		}
	}
	return h
}

func shardIndex(id uuid.UUID, n int) int {
	var hash uint64
	for i := 0; i < 8; i++ {
		hash = hash*131 + uint64(id[i])
	}
	return int(hash % uint64(n))
}

func (h *Hub) shardFor(userID uuid.UUID) *hubShard {
	return h.shards[shardIndex(userID, len(h.shards))]
}

// Done returns a channel that is closed when Run() has fully exited.
// Callers use this during graceful shutdown to wait for the Hub to drain.
func (h *Hub) Done() <-chan struct{} { return h.done }

// Healthy reports whether every shard event loop is operating normally.
// It flips to false if a panic was recovered inside any loop.
func (h *Hub) Healthy() bool { return h.healthy.Load() }

// SetWelcomeMessage stores a pre-serialised envelope delivered to every
// newly registered connection (late joiners during a chant countdown).
// Pass nil or an empty slice to clear.
func (h *Hub) SetWelcomeMessage(data []byte) {
	if len(data) == 0 {
		h.welcome.Store([]byte(nil))
		return
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	h.welcome.Store(cp)
}

func (h *Hub) welcomeBytes() []byte {
	v, _ := h.welcome.Load().([]byte)
	return v
}

func (h *Hub) enqueueRegister(c *Client) {
	h.shardFor(c.UserID).register <- c
}

func (h *Hub) enqueueUnregister(c *Client) {
	select {
	case h.shardFor(c.UserID).unregister <- c:
	case <-h.done:
	}
}

func (h *Hub) enqueueSubscribe(msg subscriptionMsg) {
	select {
	case h.shardFor(msg.client.UserID).subscribe <- msg:
	case <-h.done:
	}
}

func (h *Hub) enqueueDisconnectUser(ctx context.Context, userID uuid.UUID) error {
	select {
	case h.shardFor(userID).disconnectUser <- disconnectMsg{userID: userID}:
		return nil
	case <-h.done:
		return ErrHubStopped
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *Hub) enqueuePublish(ctx context.Context, msg publishMsg) error {
	if msg.userID != nil {
		select {
		case h.shardFor(*msg.userID).publish <- msg:
			return nil
		case <-h.done:
			return ErrHubStopped
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	for _, shard := range h.shards {
		select {
		case shard.publish <- msg:
		case <-h.done:
			return ErrHubStopped
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (h *Hub) queryUsers(ctx context.Context, q usersQuery) ([]uuid.UUID, error) {
	replies := make([]chan []uuid.UUID, len(h.shards))
	for i, shard := range h.shards {
		replies[i] = make(chan []uuid.UUID, 1)
		sq := usersQuery{channel: q.channel, all: q.all, reply: replies[i]}
		select {
		case shard.usersQueries <- sq:
		case <-h.done:
			return nil, ErrHubStopped
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	seen := make(map[uuid.UUID]struct{})
	ids := make([]uuid.UUID, 0)
	for _, reply := range replies {
		select {
		case part := <-reply:
			for _, id := range part {
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		case <-h.done:
			return nil, ErrHubStopped
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return ids, nil
}

// Run starts every shard event loop. It exits when ctx is cancelled and all
// shards have finished teardown.
func (h *Hub) Run(ctx context.Context) {
	defer close(h.done)

	var wg sync.WaitGroup
	wg.Add(len(h.shards))
	for _, shard := range h.shards {
		go func(s *hubShard) {
			defer wg.Done()
			s.run(ctx)
		}(shard)
	}
	wg.Wait()
}

func (s *hubShard) run(ctx context.Context) {
	clients := make(map[*Client]bool)
	channels := make(map[string]map[*Client]bool)
	users := make(map[uuid.UUID]map[*Client]bool)

	for {
		select {
		case <-ctx.Done():
			s.teardown(clients, channels, users)
			return

		case c := <-s.register:
			s.safe("register", func() { s.doRegister(c, clients, users) })

		case c := <-s.unregister:
			s.safe("unregister", func() { s.doUnregister(c, clients, channels, users) })

		case sub := <-s.subscribe:
			s.safe("subscribe", func() { s.doSubscribe(sub, channels) })

		case d := <-s.disconnectUser:
			s.safe("disconnect_user", func() { s.doDisconnectUser(d, clients, channels, users) })

		case q := <-s.usersQueries:
			s.safe("users_query", func() { s.doUsersQuery(q, channels, users) })

		case msg := <-s.publish:
			s.safe("publish", func() { s.dispatch(msg, clients, channels, users) })
		}
	}
}

func (s *hubShard) safe(op string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			s.hub.metrics.HubPanics.Add(1)
			s.hub.healthy.Store(false)
			logger.Error().
				Str("hub_op", op).
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("hub_panic_recovered")
		}
	}()
	fn()
}

func (s *hubShard) doRegister(c *Client, clients map[*Client]bool, users map[uuid.UUID]map[*Client]bool) {
	clients[c] = true
	if users[c.UserID] == nil {
		users[c.UserID] = make(map[*Client]bool)
	}
	users[c.UserID][c] = true
	s.hub.metrics.ActiveConnections.Add(1)
	logger.Info().
		Str("user_id", c.UserID.String()).
		Msg("connection_opened")

	if data := s.hub.welcomeBytes(); len(data) > 0 {
		s.sendToClient(c, data)
	}
}

func (s *hubShard) doUnregister(
	c *Client,
	clients map[*Client]bool,
	channels map[string]map[*Client]bool,
	users map[uuid.UUID]map[*Client]bool,
) {
	if !clients[c] {
		return
	}
	delete(clients, c)
	s.closeClient(c)

	for ch, members := range channels {
		if members[c] {
			delete(members, c)
			s.hub.metrics.ActiveSubscriptions.Add(-1)
			if len(members) == 0 {
				delete(channels, ch)
			}
		}
	}

	if userClients := users[c.UserID]; userClients != nil {
		delete(userClients, c)
		if len(userClients) == 0 {
			delete(users, c.UserID)
		}
	}

	s.hub.metrics.ActiveConnections.Add(-1)
	logger.Info().
		Str("user_id", c.UserID.String()).
		Msg("connection_closed")
}

func (s *hubShard) doSubscribe(sub subscriptionMsg, channels map[string]map[*Client]bool) {
	if sub.add {
		if channels[sub.channel] == nil {
			channels[sub.channel] = make(map[*Client]bool)
		}
		if !channels[sub.channel][sub.client] {
			channels[sub.channel][sub.client] = true
			s.hub.metrics.ActiveSubscriptions.Add(1)
			logger.Info().
				Str("user_id", sub.client.UserID.String()).
				Str("channel", sub.channel).
				Msg("subscription_added")
		}
		return
	}

	if channels[sub.channel] != nil && channels[sub.channel][sub.client] {
		delete(channels[sub.channel], sub.client)
		s.hub.metrics.ActiveSubscriptions.Add(-1)
		if len(channels[sub.channel]) == 0 {
			delete(channels, sub.channel)
		}
		logger.Info().
			Str("user_id", sub.client.UserID.String()).
			Str("channel", sub.channel).
			Msg("subscription_removed")
	}
}

func (s *hubShard) doDisconnectUser(
	d disconnectMsg,
	clients map[*Client]bool,
	channels map[string]map[*Client]bool,
	users map[uuid.UUID]map[*Client]bool,
) {
	targets := users[d.userID]
	if len(targets) == 0 {
		return
	}
	victims := make([]*Client, 0, len(targets))
	for c := range targets {
		victims = append(victims, c)
	}
	for _, c := range victims {
		s.doUnregister(c, clients, channels, users)
		s.hub.metrics.UsersDisconnected.Add(1)
	}
	logger.Info().
		Str("user_id", d.userID.String()).
		Int("connections_closed", len(victims)).
		Msg("user_disconnected")
}

func (s *hubShard) doUsersQuery(
	q usersQuery,
	channels map[string]map[*Client]bool,
	users map[uuid.UUID]map[*Client]bool,
) {
	if q.all {
		ids := make([]uuid.UUID, 0, len(users))
		for id := range users {
			ids = append(ids, id)
		}
		q.reply <- ids
		return
	}

	members := channels[q.channel]
	seen := make(map[uuid.UUID]bool, len(members))
	ids := make([]uuid.UUID, 0, len(members))
	for c := range members {
		if !seen[c.UserID] {
			seen[c.UserID] = true
			ids = append(ids, c.UserID)
		}
	}
	q.reply <- ids
}

func (s *hubShard) dispatch(
	msg publishMsg,
	clients map[*Client]bool,
	channels map[string]map[*Client]bool,
	users map[uuid.UUID]map[*Client]bool,
) {
	switch {
	case msg.channel != nil:
		for c := range channels[*msg.channel] {
			s.sendToClient(c, msg.data)
		}

	case msg.userID != nil:
		for c := range users[*msg.userID] {
			s.sendToClient(c, msg.data)
		}

	default:
		for c := range clients {
			s.sendToClient(c, msg.data)
		}
	}
}

func (s *hubShard) sendToClient(c *Client, data []byte) {
	select {
	case <-c.closed:
		return
	default:
	}

	select {
	case c.Send <- data:
		s.hub.metrics.EventsDelivered.Add(1)
	default:
		s.closeClient(c)
		s.hub.metrics.EventsFailed.Add(1)
		s.hub.metrics.ClientsDroppedBufferFull.Add(1)
		logger.Warn().
			Str("user_id", c.UserID.String()).
			Str("reason", "send_buffer_full").
			Msg("client_dropped")
	}
}

func (s *hubShard) closeClient(c *Client) {
	select {
	case <-c.closed:
	default:
		close(c.closed)
		close(c.Send)
	}
}

func (s *hubShard) teardown(
	clients map[*Client]bool,
	channels map[string]map[*Client]bool,
	users map[uuid.UUID]map[*Client]bool,
) {
	remaining := len(clients)
	for c := range clients {
		s.closeClient(c)
	}
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
		case <-s.unregister:
			remaining--
		case <-deadline:
			logger.Warn().
				Int("undrained_clients", remaining).
				Msg("hub teardown drain timeout; remaining readPumps will exit on connection close")
			return
		}
	}
}

// MatchChannel returns the canonical channel name for a match.
func MatchChannel(matchID uuid.UUID) string {
	return fmt.Sprintf("match:%s", matchID.String())
}
