package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cascade-analytics/cascade/services/writer/internal/schema"
	"github.com/cascade-analytics/cascade/services/writer/internal/writer"
	"github.com/redis/go-redis/v9"
)

const (
	defaultBatchSize      = 1000
	defaultFlushInterval  = 500 * time.Millisecond
	blockDuration         = 2 * time.Second
)

// StreamConsumer reads from a Redis Stream consumer group and writes to ClickHouse.
type StreamConsumer struct {
	rdb      *redis.Client
	ch       *writer.ClickHouseWriter
	stream   string
	group    string
	consumer string
	batch    int
	flush    time.Duration
}

func New(
	rdb *redis.Client,
	ch *writer.ClickHouseWriter,
	stream, group, consumerName string,
	batchSize int,
	flushInterval time.Duration,
) *StreamConsumer {
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	if flushInterval <= 0 {
		flushInterval = defaultFlushInterval
	}
	return &StreamConsumer{
		rdb:      rdb,
		ch:       ch,
		stream:   stream,
		group:    group,
		consumer: consumerName,
		batch:    batchSize,
		flush:    flushInterval,
	}
}

// EnsureGroup creates the consumer group if it doesn't exist.
func (c *StreamConsumer) EnsureGroup(ctx context.Context) error {
	err := c.rdb.XGroupCreateMkStream(ctx, c.stream, c.group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("create consumer group: %w", err)
	}
	return nil
}

// Run reads messages in a loop, batches them, and writes to ClickHouse.
// At-least-once delivery: XACK only after successful ClickHouse write.
func (c *StreamConsumer) Run(ctx context.Context) error {
	buf := make([]schema.StoredEvent, 0, c.batch)
	ids := make([]string, 0, c.batch)
	ticker := time.NewTicker(c.flush)
	defer ticker.Stop()

	flush := func() error {
		if len(buf) == 0 {
			return nil
		}
		if err := c.ch.WriteBatch(ctx, buf); err != nil {
			return fmt.Errorf("write batch: %w", err)
		}
		if err := c.rdb.XAck(ctx, c.stream, c.group, ids...).Err(); err != nil {
			return fmt.Errorf("xack: %w", err)
		}
		buf = buf[:0]
		ids = ids[:0]
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return flush()
		case <-ticker.C:
			if err := flush(); err != nil {
				return err
			}
		default:
		}

		msgs, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.consumer,
			Streams:  []string{c.stream, ">"},
			Count:    int64(c.batch - len(buf)),
			Block:    blockDuration,
		}).Result()
		if err != nil && err != redis.Nil {
			if ctx.Err() != nil {
				return flush()
			}
			// Recreate the group if the stream or group was deleted.
			if isNoGroup(err) {
				if egErr := c.EnsureGroup(ctx); egErr != nil {
					return fmt.Errorf("re-create group after NOGROUP: %w", egErr)
				}
				continue
			}
			return fmt.Errorf("xreadgroup: %w", err)
		}

		for _, stream := range msgs {
			for _, msg := range stream.Messages {
				data, ok := msg.Values["data"].(string)
				if !ok {
					continue
				}
				var ev schema.StoredEvent
				if err := json.Unmarshal([]byte(data), &ev); err != nil {
					continue
				}
				buf = append(buf, ev)
				ids = append(ids, msg.ID)
			}
		}

		if len(buf) >= c.batch {
			if err := flush(); err != nil {
				return err
			}
		}
	}
}

func isNoGroup(err error) bool {
	return strings.Contains(err.Error(), "NOGROUP")
}
