package ws

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"clap/internal/modules/realtime/dto"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Timing constants for WebSocket liveness management.
const (
	writeWait      = 10 * time.Second // max time to write a message
	pongWait       = 60 * time.Second // max idle time before closing
	pingInterval   = 30 * time.Second // WS ping frame sent to client (< pongWait)
	maxMessageSize = 1024             // max inbound message bytes
	sendBufferSize = 256              // per-client outbound queue depth

	// expiryCheckInterval is how often a connection with a known JWT expiry is
	// re-evaluated for token expiry.
	expiryCheckInterval = 15 * time.Second
	// subscriptionRateLimitTimeout bounds the Redis call for subscription
	// rate limiting so a slow Redis can never stall the read pump.
	subscriptionRateLimitTimeout = 2 * time.Second
)

// Client represents a single authenticated WebSocket connection.
//
// Two goroutines run per client:
//   - readPump — reads messages from the WebSocket, handles application-level pings and subscriptions.
//   - writePump — drains the Send channel and forwards data to the WebSocket; sends WS-level ping frames.
//
// All Hub state mutations are funnelled through the Hub's channel-based API to avoid data races.
type Client struct {
	UserID          uuid.UUID
	Conn            *websocket.Conn
	Send            chan []byte
	closed          chan struct{} // closed by Hub when the client is evicted
	hub             *Hub
	lastHeartbeatMs atomic.Int64 // unix milliseconds; updated on pong receipt
	connectedAtMs   int64        // immutable; set at construction

	// clientIP is the originating IP, used for subscription rate limiting.
	// Empty disables per-client subscription rate limiting (e.g. tests).
	clientIP string
	// tokenExpiresAt is the JWT expiry. When non-zero the connection is closed
	// once the token expires (CR-8 session expiry). Zero disables enforcement.
	tokenExpiresAt time.Time
}

// ClientOptions carries optional, production-only client configuration.
// The zero value is valid and preserves the original NewClient behaviour.
type ClientOptions struct {
	// ClientIP is the originating remote IP for subscription rate limiting.
	ClientIP string
	// TokenExpiresAt is the JWT expiry; zero means no expiry enforcement.
	TokenExpiresAt time.Time
}

// NewClient creates a Client and registers it with the Hub.
// The caller must start readPump and writePump goroutines immediately after.
func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID) *Client {
	return NewClientWithOptions(hub, conn, userID, ClientOptions{})
}

// NewClientWithOptions creates a Client with optional IP and JWT-expiry
// configuration and registers it with the Hub.
func NewClientWithOptions(hub *Hub, conn *websocket.Conn, userID uuid.UUID, opts ClientOptions) *Client {
	c := &Client{
		UserID:         userID,
		Conn:           conn,
		Send:           make(chan []byte, sendBufferSize),
		closed:         make(chan struct{}),
		hub:            hub,
		connectedAtMs:  time.Now().UnixMilli(),
		clientIP:       opts.ClientIP,
		tokenExpiresAt: opts.TokenExpiresAt,
	}
	c.lastHeartbeatMs.Store(time.Now().UnixMilli())
	hub.enqueueRegister(c)
	return c
}

// ReadPump pumps messages from the WebSocket to the Hub / handler.
// One goroutine per client; when it returns the client is deregistered.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.enqueueUnregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))

	// Extend the read deadline on every pong frame received.
	c.Conn.SetPongHandler(func(string) error {
		c.lastHeartbeatMs.Store(time.Now().UnixMilli())
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure,
			) {
				logger.Warn().
					Str("user_id", c.UserID.String()).
					Err(err).
					Msg("websocket read error")
			}
			return
		}
		c.handleInbound(raw)
	}
}

// WritePump pumps messages from the Send channel to the WebSocket.
// It also sends periodic WS ping frames to detect dead connections.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingInterval)

	// Periodically check JWT expiry. When no expiry is configured the channel
	// is nil and the corresponding select case blocks forever (never fires).
	// The interval is capped so that a connection is torn down promptly around
	// its token's expiry rather than up to expiryCheckInterval later.
	var expiryTicker *time.Ticker
	var expiryC <-chan time.Time
	if !c.tokenExpiresAt.IsZero() {
		interval := expiryCheckInterval
		if remaining := time.Until(c.tokenExpiresAt); remaining < interval {
			interval = remaining + 500*time.Millisecond
		}
		if interval < time.Second {
			interval = time.Second
		}
		expiryTicker = time.NewTicker(interval)
		expiryC = expiryTicker.C
	}

	defer func() {
		ticker.Stop()
		if expiryTicker != nil {
			expiryTicker.Stop()
		}
		c.Conn.Close()
	}()

	for {
		select {
		case <-c.closed:
			// Hub closed the connection; send a normal close frame.
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = c.Conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return

		case <-expiryC:
			// JWT expired while connected — terminate the session (CR-8 / F-008).
			if c.isTokenExpired() {
				c.hub.metrics.SessionsExpired.Add(1)
				logger.Info().
					Str("user_id", c.UserID.String()).
					Msg("session_expired")
				_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
				_ = c.Conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "token expired"))
				c.hub.enqueueUnregister(c)
				return
			}

		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel was closed — the Hub already issued a close.
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Warn().
					Str("user_id", c.UserID.String()).
					Err(err).
					Msg("websocket write error")
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// LastHeartbeatMs returns the unix-millisecond timestamp of the last pong
// received from this client, useful for observability.
func (c *Client) LastHeartbeatMs() int64 {
	return c.lastHeartbeatMs.Load()
}

// ConnectedAtMs returns the connection creation time in unix milliseconds.
func (c *Client) ConnectedAtMs() int64 {
	return c.connectedAtMs
}

// ─── Inbound message dispatch ─────────────────────────────────────────────────

func (c *Client) handleInbound(raw []byte) {
	var msg dto.ClientMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		logger.Debug().Str("user_id", c.UserID.String()).Msg("unparseable client message; ignored")
		return
	}

	switch msg.Type {
	case dto.EventTypePing:
		c.sendPong(msg.ClientTimeMs)

	case "subscribe":
		channel := sanitiseChannel(msg.Channel)
		if channel == "" {
			return
		}
		// Enforce per-IP subscription rate limit before mutating Hub state (CR-4).
		ctx, cancel := context.WithTimeout(context.Background(), subscriptionRateLimitTimeout)
		allowed := SubscriptionRateLimiter(ctx, c.clientIP)
		cancel()
		if !allowed {
			c.hub.metrics.SubscriptionsRejected.Add(1)
			c.sendError("subscription_rate_limited", "Subscription rate limit exceeded. Slow down.", channel)
			return
		}
		c.hub.enqueueSubscribe(subscriptionMsg{client: c, channel: channel, add: true})

	case "unsubscribe":
		channel := sanitiseChannel(msg.Channel)
		if channel != "" {
			c.hub.enqueueSubscribe(subscriptionMsg{client: c, channel: channel, add: false})
		}
	}
}

// isTokenExpired reports whether the connection's JWT expiry has passed.
func (c *Client) isTokenExpired() bool {
	return !c.tokenExpiresAt.IsZero() && time.Now().After(c.tokenExpiresAt)
}

// sendError delivers a structured error envelope to the client (best-effort).
func (c *Client) sendError(code, message, channel string) {
	env := dto.NewEnvelope(dto.EventTypeError, nil, dto.ErrorPayload{
		Code:    code,
		Message: message,
		Channel: channel,
	})
	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	select {
	case <-c.closed:
	case c.Send <- data:
	default:
		// Buffer full — drop; writePump will close the connection if needed.
	}
}

// sendPong replies with an application-level pong, echoing the client's
// timestamp (when provided) alongside the server clock for WS time sync.
func (c *Client) sendPong(clientTimeMs int64) {
	pong, _ := json.Marshal(dto.PongMessage{
		Type:         dto.EventTypePong,
		ClientTimeMs: clientTimeMs,
		ServerTimeMs: time.Now().UnixMilli(),
	})
	select {
	case c.Send <- pong:
		c.lastHeartbeatMs.Store(time.Now().UnixMilli())
	default:
		// Drop if buffer full — writePump will eventually close the connection.
	}
}

// sanitiseChannel validates and normalises the channel name.
// Only "match:<uuid>" patterns are accepted from clients to prevent abuse.
func sanitiseChannel(ch string) string {
	ch = strings.TrimSpace(ch)
	if strings.HasPrefix(ch, "match:") {
		rest := strings.TrimPrefix(ch, "match:")
		if _, err := uuid.Parse(rest); err == nil {
			return ch
		}
	}
	return ""
}
