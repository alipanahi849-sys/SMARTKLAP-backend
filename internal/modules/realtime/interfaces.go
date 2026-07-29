package realtime

import (
	"context"
)

type Event struct {
	Channel string      `json:"channel"`
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
}

type Publisher interface {
	Publish(ctx context.Context, channel string, event *Event) error
	Broadcast(ctx context.Context, event *Event) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, channels ...string) (<-chan *Event, error)
	Unsubscribe(ctx context.Context, channels ...string) error
}

type RealtimeService interface {
	Publisher
	Subscriber
}
