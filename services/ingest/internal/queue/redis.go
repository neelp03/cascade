package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Enqueuer publishes events to a Redis Stream.
type Enqueuer struct {
	client *redis.Client
	stream string
}

func NewEnqueuer(client *redis.Client, stream string) *Enqueuer {
	return &Enqueuer{client: client, stream: stream}
}

// Enqueue serialises payload and appends it to the stream.
// Returns the stream entry ID on success.
func (e *Enqueuer) Enqueue(ctx context.Context, payload any) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal event: %w", err)
	}
	id, err := e.client.XAdd(ctx, &redis.XAddArgs{
		Stream: e.stream,
		Values: map[string]any{"data": string(b)},
	}).Result()
	if err != nil {
		return "", fmt.Errorf("xadd to %s: %w", e.stream, err)
	}
	return id, nil
}

// Ping checks Redis connectivity.
func (e *Enqueuer) Ping(ctx context.Context) error {
	return e.client.Ping(ctx).Err()
}
