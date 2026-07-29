package realtime

import (
	"clap/internal/shared/redis"
	"context"
	"encoding/json"
)

type RedisPubSubService struct{}

func NewRedisPubSubService() *RedisPubSubService {
	return &RedisPubSubService{}
}

func (s *RedisPubSubService) Publish(ctx context.Context, channel string, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return redis.Publish(ctx, channel, data)
}

func (s *RedisPubSubService) Broadcast(ctx context.Context, event *Event) error {
	return s.Publish(ctx, "broadcast", event)
}

func (s *RedisPubSubService) Subscribe(ctx context.Context, channels ...string) (<-chan *Event, error) {
	pubsub := redis.Subscribe(ctx, channels...)

	eventChan := make(chan *Event, 100)

	go func() {
		defer close(eventChan)
		defer pubsub.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-pubsub.Channel():
				var event Event
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					continue
				}
				eventChan <- &event
			}
		}
	}()

	return eventChan, nil
}

func (s *RedisPubSubService) Unsubscribe(ctx context.Context, channels ...string) error {
	return nil
}
