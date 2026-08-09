package gateway

import (
	"context"
	"encoding/json"
	"errors"

	"clap/internal/modules/realtime/dto"
	"clap/internal/modules/realtime/metrics"
	"clap/internal/modules/realtime/ws"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

// WebSocketRealtimeGateway is the production implementation of RealtimeGateway
// and the supplementary typed-delivery methods used by business services.
//
// Business modules (playback, lyricssync) never import this package
// directly — they use the transport-agnostic interfaces defined at their own layer.
type WebSocketRealtimeGateway struct {
	cm      *ws.ConnectionManager
	metrics *metrics.Metrics
}

// NewWebSocketRealtimeGateway constructs the gateway.
func NewWebSocketRealtimeGateway(cm *ws.ConnectionManager, m *metrics.Metrics) *WebSocketRealtimeGateway {
	return &WebSocketRealtimeGateway{cm: cm, metrics: m}
}

// ─── RealtimeGateway interface ────────────────────────────────────────────────

// Publish serialises event and routes it to all subscribers of channel.
func (g *WebSocketRealtimeGateway) Publish(ctx context.Context, channel string, event *GatewayEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if pubErr := g.cm.PublishToChannel(ctx, channel, data); pubErr != nil {
		return pubErr
	}
	g.metrics.EventsPublished.Add(1)
	logger.Debug().
		Str("channel", channel).
		Str("event_type", event.Type).
		Msg("event_published")
	return nil
}

// Broadcast sends event to every connected client.
func (g *WebSocketRealtimeGateway) Broadcast(ctx context.Context, event *GatewayEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if pubErr := g.cm.Broadcast(ctx, data); pubErr != nil {
		return pubErr
	}
	g.metrics.EventsPublished.Add(1)
	return nil
}

// Subscribe is not supported by the WebSocket push-only gateway.
// Use InMemoryRealtimeGateway if server-side pub/sub is needed.
func (g *WebSocketRealtimeGateway) Subscribe(_ context.Context, _ string) (<-chan *GatewayEvent, error) {
	return nil, errors.New("WebSocketRealtimeGateway does not support server-side Subscribe; use InMemoryRealtimeGateway")
}

// DisconnectUser forcibly terminates every WebSocket connection owned by the
// user, removing them from all channels and updating metrics (CR-9 / F-011).
func (g *WebSocketRealtimeGateway) DisconnectUser(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	if dErr := g.cm.DisconnectUser(ctx, uid); dErr != nil {
		logger.Warn().
			Str("user_id", uid.String()).
			Err(dErr).
			Msg("disconnect_user_failed")
		return dErr
	}
	logger.Info().
		Str("user_id", uid.String()).
		Msg("disconnect_user_requested")
	return nil
}

// ─── Envelope-based delivery helpers ─────────────────────────────────────────

// PublishEnvelope serialises an EventEnvelope and delivers it to a channel.
func (g *WebSocketRealtimeGateway) PublishEnvelope(ctx context.Context, channel string, env *dto.EventEnvelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		g.metrics.EventsFailed.Add(1)
		g.metrics.EventsPublishFailed.Add(1)
		logger.Error().
			Str("event_id", env.ID).
			Str("event_type", env.Type).
			Str("channel", channel).
			Err(err).
			Msg("event_publish_failed")
		return err
	}
	if pubErr := g.cm.PublishToChannel(ctx, channel, data); pubErr != nil {
		g.metrics.EventsFailed.Add(1)
		g.metrics.EventsPublishFailed.Add(1)
		logger.Error().
			Str("event_id", env.ID).
			Str("event_type", env.Type).
			Str("channel", channel).
			Err(pubErr).
			Msg("event_publish_failed")
		return pubErr
	}
	g.metrics.EventsPublished.Add(1)
	logger.Info().
		Str("event_id", env.ID).
		Str("event_type", env.Type).
		Str("channel", channel).
		Msg("event_published")
	return nil
}

// PublishToMatch delivers env to all clients subscribed to the match channel.
func (g *WebSocketRealtimeGateway) PublishToMatch(ctx context.Context, matchID uuid.UUID, env *dto.EventEnvelope) error {
	channel := ws.MatchChannel(matchID)
	return g.PublishEnvelope(ctx, channel, env)
}

// PublishToUser delivers env to all active connections of a specific user.
func (g *WebSocketRealtimeGateway) PublishToUser(ctx context.Context, userID uuid.UUID, env *dto.EventEnvelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		g.metrics.EventsFailed.Add(1)
		return err
	}
	if pubErr := g.cm.PublishToUser(ctx, userID, data); pubErr != nil {
		g.metrics.EventsFailed.Add(1)
		return pubErr
	}
	g.metrics.EventsPublished.Add(1)
	return nil
}

// BroadcastEnvelope sends env to every connected client.
func (g *WebSocketRealtimeGateway) BroadcastEnvelope(ctx context.Context, env *dto.EventEnvelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		g.metrics.EventsFailed.Add(1)
		return err
	}
	if pubErr := g.cm.Broadcast(ctx, data); pubErr != nil {
		g.metrics.EventsFailed.Add(1)
		return pubErr
	}
	g.metrics.EventsPublished.Add(1)
	return nil
}

// PublishMatchEvent creates an EventEnvelope and publishes it to the match channel.
func (g *WebSocketRealtimeGateway) PublishMatchEvent(
	ctx context.Context,
	matchID uuid.UUID,
	eventType string,
	payload any,
) error {
	env := dto.NewEnvelope(eventType, &matchID, payload)
	return g.PublishToMatch(ctx, matchID, env)
}
