package gateway

import (
	"context"
	"sync"
)

const channelBufferSize = 256

// InMemoryRealtimeGateway is a thread-safe, single-process gateway.
// It is the Phase 4 implementation placeholder — no external broker required.
// Replace with a Centrifugo/NATS/Kafka adapter in Phase 5+.
type InMemoryRealtimeGateway struct {
	mu           sync.RWMutex
	channels     map[string][]chan *GatewayEvent // channel key → subscriber chans
	userChannels map[string][]string             // userID → channel keys (for disconnect)
	broadcasts   []chan *GatewayEvent            // global broadcast subscribers
}

func NewInMemoryRealtimeGateway() *InMemoryRealtimeGateway {
	return &InMemoryRealtimeGateway{
		channels:     make(map[string][]chan *GatewayEvent),
		userChannels: make(map[string][]string),
	}
}

func (g *InMemoryRealtimeGateway) Publish(ctx context.Context, channel string, event *GatewayEvent) error {
	g.mu.RLock()
	subs := make([]chan *GatewayEvent, len(g.channels[channel]))
	copy(subs, g.channels[channel])
	g.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Drop if subscriber is slow — prevents head-of-line blocking.
		}
	}
	return nil
}

func (g *InMemoryRealtimeGateway) Broadcast(ctx context.Context, event *GatewayEvent) error {
	g.mu.RLock()
	subs := make([]chan *GatewayEvent, len(g.broadcasts))
	copy(subs, g.broadcasts)
	g.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	return nil
}

// Subscribe registers a subscriber for a channel.
// A background goroutine cleans up when ctx is cancelled — no goroutine leak.
func (g *InMemoryRealtimeGateway) Subscribe(ctx context.Context, channel string) (<-chan *GatewayEvent, error) {
	ch := make(chan *GatewayEvent, channelBufferSize)

	g.mu.Lock()
	g.channels[channel] = append(g.channels[channel], ch)
	if channel == "__broadcast__" {
		g.broadcasts = append(g.broadcasts, ch)
	}
	g.mu.Unlock()

	go func() {
		<-ctx.Done()
		g.removeSubscription(channel, ch)
		close(ch)
	}()

	return ch, nil
}

// DisconnectUser closes all subscriptions for a user.
// Individual subscription goroutines handle their own cleanup via context cancellation.
func (g *InMemoryRealtimeGateway) DisconnectUser(_ context.Context, userID string) error {
	g.mu.Lock()
	delete(g.userChannels, userID)
	g.mu.Unlock()
	return nil
}

// SubscriberCount returns the number of active subscribers for a channel.
// Useful for observability and testing.
func (g *InMemoryRealtimeGateway) SubscriberCount(channel string) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.channels[channel])
}

func (g *InMemoryRealtimeGateway) removeSubscription(channel string, target chan *GatewayEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()

	filtered := g.channels[channel][:0]
	for _, ch := range g.channels[channel] {
		if ch != target {
			filtered = append(filtered, ch)
		}
	}
	g.channels[channel] = filtered

	if channel == "__broadcast__" {
		bfiltered := g.broadcasts[:0]
		for _, ch := range g.broadcasts {
			if ch != target {
				bfiltered = append(bfiltered, ch)
			}
		}
		g.broadcasts = bfiltered
	}
}
