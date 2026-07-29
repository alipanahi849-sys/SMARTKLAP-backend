package realtime

import (
	"context"
	"encoding/json"
)

type EventPublisher interface {
	Publish(ctx context.Context, channel string, event *Event) error
	PublishToUser(ctx context.Context, userID string, event *Event) error
	Broadcast(ctx context.Context, event *Event) error
}

type RedisEventPublisher struct {
	publisher Publisher
}

func NewRedisEventPublisher(publisher Publisher) *RedisEventPublisher {
	return &RedisEventPublisher{
		publisher: publisher,
	}
}

func (p *RedisEventPublisher) Publish(ctx context.Context, channel string, event *Event) error {
	return p.publisher.Publish(ctx, channel, event)
}

func (p *RedisEventPublisher) PublishToUser(ctx context.Context, userID string, event *Event) error {
	channel := "user:" + userID
	return p.publisher.Publish(ctx, channel, event)
}

func (p *RedisEventPublisher) Broadcast(ctx context.Context, event *Event) error {
	return p.publisher.Broadcast(ctx, event)
}

func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

func EventFromJSON(data []byte) (*Event, error) {
	var event Event
	err := json.Unmarshal(data, &event)
	return &event, err
}
